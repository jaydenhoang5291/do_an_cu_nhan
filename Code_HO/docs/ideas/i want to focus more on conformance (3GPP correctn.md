<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# i want to focus more on conformance (3GPP correctness). So i think security (fuzzing/attacks) is a good choice.

Also, i have library protocol, inside source of 5g core and emulator (ue + gnb) so if i can say all the test will be white-box!

now, i dont have any idea to implement this feature security (fuzzing/attacks). Find some paper, even those papers both do not relate + relate with 5g core emulator (ue, gnb), I just need the idea to concrete!!!

Yes—if you own the protocol libraries and the source of both the 5GC and the UE/gNB emulator, then **white-box security testing** is probably the strongest direction for you, especially for 3GPP conformance plus robustness of the control plane.[^1][^2][^3]
The most concrete path is to combine 5G-specific work on RRC/NAS/NGAP fuzzing with general stateful-fuzzing ideas from protocol and distributed-systems research, then turn them into emulator features that can deliberately violate ordering, timing, identity consistency, and state transitions.[^2][^3][^1]

## Papers to mine

| Title | Why it is useful for you | Concrete emulator idea |
| :-- | :-- | :-- |
| **Prior Knowledge Adaptive 5G Vulnerability Detection via Multi-Fuzzing** (Yang et al., 2024, arXiv) [^1] | Proposes three levels of fuzzing—LAL, SyAL, and **SoAL**—and the SoAL mode is explicitly white-box, targeting significant bits and high-risk commands after earlier exploration narrows the search space. [^1] | Build a 3-stage engine in your emulator: broad procedure exploration first, then risky-message prioritization, then white-box bit/field fuzzing only on the dangerous NAS/NGAP/RRC messages. [^1] |
| **5Greplay: a 5G Network Traffic Fuzzer** (Salazar et al., 2023 arXiv / ARES 2021) [^2] | Shows a practical 5G packet mutation framework over SCTP, NAS-5G, and NGAP, with atomic operators like delete, change attribute, duplicate, and planned reorder, and it was evaluated with UERANSIM plus free5GC/open5GS. [^2] | Add a rule engine to your emulator so every test scenario is a sequence of operators such as `drop`, `duplicate`, `mutate IE`, `wrong SCTP PPID`, and `reorder`, all bound to specific procedure points. [^2] |
| **AFLNet Five Years Later: On Coverage-Guided Protocol Fuzzing** (Meng et al., 2024, arXiv) [^3] | AFLNet’s main value is that it treats a **message sequence** as the fuzz seed and uses both state coverage and code coverage to steer the campaign through protocol states. [^3] | Represent each registration, authentication, service request, handover, and PDU session procedure as a mutable message-sequence seed instead of single-packet fuzzing. [^3] |
| **Model-Guided Fuzzing of Distributed Systems** (Gulcan et al., 2024, arXiv) [^4] | Uses an abstract formal model, event schedules, and mutations such as swapping message delivery order or crash/restart actions to find deep concurrency bugs faster. [^4] | Treat AMF/SMF/UDM/AUSF interactions like a distributed protocol and fuzz **event schedules**, not just packet bytes, to expose race conditions and out-of-order control-plane bugs. [^4] |

## Best ideas to steal

Your most valuable feature is a **stateful white-box campaign manager** that knows the current 3GPP procedure state, the internal decoder/parser path, and the target NF code region, then focuses mutations there instead of mutating packets blindly.[^3][^1]
That idea comes directly from SoAL’s risk-prioritized white-box fuzzing and AFLNet’s state-plus-code-coverage approach, and it fits your setup because you control both ends and can instrument the emulator and the core simultaneously.[^1][^3]

A second strong idea is to make the emulator support **event-schedule fuzzing**: mutate not only message content, but also delivery order, inter-message delay, retransmission timing, stream choice, and crash/restart timing of logical components.[^4][^2]
This is where out-of-order delivery, lost events, duplicated events, and race conditions become first-class test objects rather than ad hoc scripts, which is exactly the gap between normal emulators and research-grade conformance/security platforms.[^2][^4]

A third idea is to build a **spec-aware oracle** that checks whether the observed transition is legal under the intended 3GPP state machine and whether the implementation took an unexpected but reachable path.[^3][^4]
The distributed-systems paper shows how an abstract model can guide testing, while AFLNet shows that inferred or partial state models still help drive exploration even when implementation behavior differs from the written model.[^4][^3]

## Concrete features

1. **Procedure-seed fuzzing.**
Store complete seeds for Registration, Authentication, Security Mode, UE Context Release, Paging response, Service Request, PDU Session Establishment, and Handover as mutable sequences of N1/N2 messages.[^3]
For each seed, allow mutations on message content, optional step removal, step duplication, and alternative timing between steps.[^2][^3]
2. **Field-class white-box fuzzing.**
Tag every IE and field by class: identity, length, cause value, timer, security header, AMF/RAN UE ID, NAS sequence number, optional IE presence, and integrity-related fields.[^1][^2]
Then prioritize fields that are both decoder-sensitive and state-sensitive, which mirrors SoAL’s “significant bits” idea but at a protocol-engineering level more useful for NAS/NGAP/RRC.[^1]
3. **Event-schedule mutators.**
Implement operators such as `Swap(M_i, M_j)`, `Drop(M_i)`, `Duplicate(M_i,k)`, `Delay(M_i,dt)`, `Replay(M_i, state=s)`, `WrongStream(M_i)`, and `AbortAssociation(at_step=n)`.[^4][^2]
The key improvement is to apply them at semantically meaningful points like “after Authentication Request but before Security Mode Command,” not just by packet index.[^2][^3]
4. **Coverage from both sides.**
Collect code coverage in the core and in the emulator plus state coverage from the current protocol procedure, then rank testcases by novelty across both dimensions.[^3]
This is especially powerful for conformance work because a testcase may be “new” even when code coverage is flat, if it drives the core into a spec-forbidden or rarely seen control-plane state.[^4][^3]

## Attack patterns to implement first

Start with **identity-consistency violations** because 5Greplay already showed that changing identifiers in NGAP exchanges can expose suspicious or unstable handling in cores.[^2]
Concrete tests include wrong `RAN_UE_NGAP_ID`, reused `AMF_UE_NGAP_ID`, cross-UE ID borrowing, and replay of a valid uplink NAS inside the wrong NGAP UE context.[^2]

Next, implement **security-procedure replays and desynchronization** because 5Greplay directly evaluated replay of NAS Security Mode Complete and connected it to AMF security-assurance testing.[^2]
Concrete tests include duplicate Security Mode Complete, stale NAS SQN, replay after context release, replay under a new SCTP association, and mismatched NAS security context versus NGAP UE context.[^2]

Then add **ordering and race suites** inspired by ModelFuzz: late Authentication Response, early Registration Complete, duplicate Initial UE Message, delayed Initial Context Setup Response, SCTP reconnect during Security Mode, and concurrent deregistration during session setup.[^4]
These are especially good for white-box testing because you can instrument which timer expiry, lock, queue, or state-variable interleaving caused the failure.[^4]

## A practical architecture

Use a four-layer design: **scenario DSL**, message mutator, event scheduler, and verdict/oracle engine.[^4][^2]
The DSL should describe procedures and fault injections, the mutator should edit fields using protocol-library knowledge, the scheduler should control order/timing/replay/crash behavior, and the oracle should combine 3GPP transition checks, crash detection, timeout detection, and coverage feedback.[^3][^4][^2]

A very good first milestone is to support one full path—Registration plus Authentication plus Security Mode—and implement 20–30 high-value schedule and field mutations around it.[^1][^3][^2]
That path is rich enough to exercise NAS, NGAP, SCTP, identity binding, timer behavior, and security-context establishment, so it gives you publishable results early without needing the whole 5GC feature set.[^3][^2]

Would you like a **module-by-module design** next, for example: testcase format, mutator API, scheduler design, coverage hooks, and verdict logic for a white-box conformance-fuzzing emulator?
<span style="display:none">[^10][^11][^12][^13][^14][^15][^16][^17][^18][^19][^20][^21][^22][^23][^24][^25][^26][^27][^28][^29][^5][^6][^7][^8][^9]</span>

<div align="center">⁂</div>

[^1]: https://arxiv.org/html/2602.21794v1

[^2]: https://arxiv.org/html/2510.14348v1

[^3]: https://arxiv.org/pdf/2112.04257.pdf

[^4]: https://arxiv.org/pdf/2510.14348.pdf

[^5]: https://arxiv.org/html/2305.08039v2

[^6]: https://web3.arxiv.org/abs/2601.18690

[^7]: https://arxiv.org/html/2601.18690v1

[^8]: https://arxiv.org/pdf/2305.08039.pdf

[^9]: https://ar5iv.labs.arxiv.org/html/2309.12994

[^10]: https://arxiv.org/html/2412.20324v1

[^11]: https://arxiv.org/html/2410.02307v3

[^12]: https://arxiv.org/html/2305.08039v2/

[^13]: https://arxiv.org/pdf/2412.20324.pdf

[^14]: https://arxiv.org/html/2312.14479v1

[^15]: https://www.arxiv.org/pdf/2601.18690.pdf

[^16]: https://arxiv.org/pdf/2408.06844.pdf

[^17]: https://arxiv.org/html/2410.02307v2

[^18]: https://arxiv.org/html/2601.18690v2

[^19]: https://arxiv.org/pdf/2112.15498.pdf

[^20]: https://arxiv.org/abs/2305.08039

[^21]: https://dl.acm.org/doi/10.1145/3680207.3723469

[^22]: https://arxiv.org/pdf/2304.05719.pdf

[^23]: https://d-nb.info/1279934484/34

[^24]: https://ceur-ws.org/Vol-3962/paper51.pdf

[^25]: https://arxiv.org/abs/2410.02307

[^26]: https://github.com/stateafl/stateafl

[^27]: https://taesoo.kim/pubs/2020/xu:krace.pdf

[^28]: https://vtechworks.lib.vt.edu/server/api/core/bitstreams/915f870c-479c-4253-91c4-9c9f249bcc1c/content

[^29]: https://www.usenix.org/system/files/sec22-ba.pdf

