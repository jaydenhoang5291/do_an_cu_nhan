<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# i also survey on many paper about fuzz test on other system (not 5g but i think the idea of fuzz can re-use). let consider them.

I want to implement feature security (white-grey box fuzzing/vulnerability attacks) for my emulator (ue gnb) to test 5g core.

analysis then give me ideas for this feature!

Your survey direction is good: the strongest ideas for your emulator do **not** need to come only from 5G papers. A 5G core is also a distributed, stateful, partially asynchronous system, so the best reusable ideas come from microservice fuzzing, protocol fuzzing, and distributed-systems schedule fuzzing.[^1][^2][^3][^4][^5][^6][^7]

## Core design

The best feature to implement is a **hybrid white-grey box conformance-security fuzzer** where the UE/gNB emulator generates valid-enough 3GPP procedures, then mutates both message contents and event schedules while using internal code coverage, protocol-state coverage, and conformance oracles as feedback.[^6][^7][^8][^9][^1]
This matches the main conclusion of your survey file: the most reusable ideas are session/grammar-based generation, structure-aware mutation, model-guided schedule fuzzing, RL-guided exploration, and uncertainty-driven fault injection.[^1]

A plain packet mutator is too weak for your goal because most interesting 5GC bugs are in multi-step procedures, cross-NF interactions, timer races, and identity/context mismatches rather than in one malformed PDU alone.[^7][^9][^6][^1]
So your emulator should treat a testcase as a **scenario**: a sequence of UE, gNB, timer, transport, and NF events, plus assertions about what the 5GC should or should not do.[^6][^7][^1]

## Best borrowed ideas

From **Zero-Config Fuzzing for Microservices**, the key transferable idea is an auto-generated session grammar: instead of fuzzing one request, they fuzz a sequence of dependent calls using a structure-aware representation.[^3][^10][^1]
For you, this maps almost directly to a “FuzzSession” IR for Registration, Authentication, Security Mode, Service Request, PDU Session Establishment, Handover, Deregistration, and Paging, with fields and optional branches exposed for mutation.[^3][^1]

From **Model-Guided Fuzzing of Distributed Systems**, the most valuable idea is to fuzz **event schedules** and use an abstract formal model as the coverage oracle rather than relying only on code coverage.[^11][^12][^7][^1]
This is highly relevant to AMF/SMF/UDM/AUSF/PCF interactions because many bugs will only appear when delivery order, timeout order, or restart order changes while the logical procedure remains the same.[^11][^7]

From **AFLNet Five Years Later** and related stateful protocol fuzzing, the critical lesson is that the input should be a **message sequence seed** and the fuzzer should reason about protocol state, not raw bytes alone.[^6]
That gives you a direct engineering pattern for NAS/NGAP/RRC: store one corpus entry per whole 3GPP procedure trace, then mutate sequence length, branch choice, field values, and semantic timing.[^6]

From **MicroFuzz**, the useful part is less about mutation theory and more about deployment: mocking-assisted execution, distributed tracing, seed refresh, and pipeline parallelism make fuzzing practical in large distributed systems.[^13][^2][^14][^1]
That is valuable for a cloud-native 5GC because fuzzing will otherwise drown in setup cost, nondeterminism, and poor throughput.[^13][^1]

## Concrete emulator features

1. **Scenario IR / FuzzSession**
Define an intermediate representation like:

- `Action`: UE_SEND_NAS, GNB_SEND_NGAP, DROP_NEXT_DL, DELAY_MSG, DUPLICATE_MSG, RESTART_NF, EXPIRE_TIMER, RELEASE_UE_CTX.
- `Params`: UE identity fields, IE overrides, SCTP stream, delay, target NF, target state.
- `Oracle`: expected reject cause, forbidden crash, forbidden hang, state-machine legality, latency bound.[^9][^7][^1][^3]

This gives you one unified language for conformance and security testing. A testcase is no longer “a bad packet,” but a whole controlled procedure with semantic mutations and verdicts.[^7][^9][^1]

2. **Structure-aware protocol mutation**
Use your protocol libraries to mutate ASN.1/NAS fields structurally instead of bit-flipping serialized bytes.[^8][^9][^1]
Good field classes are:

- Identity binding: SUCI/SUPI, GUTI, GUAMI, RAN_UE_NGAP_ID, AMF_UE_NGAP_ID.
- State progression: registration type, follow-on request, PDU session ID, procedure transaction ID.
- Security context: KSI, security header type, NAS sequence number, replay-sensitive fields.
- Optional IE presence: remove mandatory-like combinations, insert rare optional IEs, create inconsistent IE combinations.[^8][^9]

3. **Schedule fuzzing engine**
Borrowing from ModelFuzz, define test mutations over event order:

- Swap two message deliveries.
- Delay one event until after a timer expiry.
- Deliver a duplicate before the original.
- Replay an old valid message in a new context.
- Restart or disconnect one NF between two steps.
- Rebind a UE to a new SCTP association or gNB context mid-procedure.[^1][^11][^7]

This is probably where your highest-value bugs live, because 5GC correctness depends heavily on ordering across NAS, NGAP, timers, and internal SBI calls.[^7][^1]

4. **White-box risk ranking**
Because you own the code, score each testcase with a weighted objective:

- New code coverage in parser/handler/state logic.
- New protocol-state coverage.
- New cross-NF path coverage.
- Oracle violations, unexpected reject causes, deadlock, crash, excessive retry, state desynchronization.[^8][^1][^6]

Then prioritize mutations near sensitive code regions such as UE context creation, authentication state updates, security mode transitions, session creation, handover context transfer, and cleanup/release logic.[^1][^8]

## Attack families to implement

A good first family is **identity/context mismatch attacks**. 5Greplay already showed that replaying or mutating identifiers in NGAP and NAS can expose weak context binding and unstable behavior.[^9]
Concrete tests:

- Valid NAS PDU under the wrong `RAN_UE_NGAP_ID`.
- Reuse old `AMF_UE_NGAP_ID` after context release.
- Cross-UE mixup where UE-A’s NAS goes through UE-B’s NGAP context.
- gNB-side handover request with stale UE context handles.[^9]

A second family is **security-state desynchronization**. These are excellent for conformance plus security because the core should reject or recover cleanly, not crash or silently accept.[^9]
Concrete tests:

- Duplicate Security Mode Complete.
- Replay old Authentication Response after new RAND.
- Send Registration Complete before security establishment is complete.
- Mismatch NAS sequence number, KSI, or selected security algorithm across resumed context.[^9]

A third family is **timer and race attacks**. Distributed-systems fuzzing strongly suggests that unusual schedules reveal bugs missed by content mutation alone.[^11][^7][^1]
Concrete tests:

- T3510/T3560 expiry exactly before a delayed response arrives.
- Concurrent Deregistration and PDU Session Establishment.
- UE reconnect under new SCTP association while old UE context still exists.
- Handover start while AMF is still processing Security Mode or session setup.
- AMF restart or NRF temporary unavailability between two logically linked steps.[^7][^1]

A fourth family is **uncertainty and partial-failure scenarios** from your microservice survey. That means you fuzz not just messages, but also infrastructure conditions.[^1]
Concrete tests:

- Delay SBI responses from UDM/AUSF/PCF while keeping N2 active.
- Return inconsistent internal error codes from a mocked dependency.
- Inject overload/back-pressure so retries overlap with pending UE contexts.
- Restart one NF and test whether context reconstruction causes spec violations.[^13][^1]


## Best architecture

The most practical architecture is a six-module design:

- **Scenario generator**: builds legal baseline procedures from 3GPP state machines.[^6][^1]
- **Mutator**: field-aware NAS/NGAP/RRC mutation plus semantic operators.[^8][^9]
- **Scheduler**: controls ordering, delay, drops, duplicates, timer expiry, and NF restart events.[^7]
- **Instrumentation**: code coverage, state coverage, distributed tracing, per-NF event logs.[^13][^1]
- **Oracle engine**: conformance assertions, crash/hang detection, forbidden state transitions, cross-context consistency checks.[^6][^7]
- **Corpus manager**: stores interesting scenario seeds and refreshes them as the core changes, similar to MicroFuzz’s seed lifecycle idea.[^13][^1]

If your emulator will scale to many UEs, you should also copy the **microservice architecture for the fuzzer itself**: separate mutation, execution, tracing, and triage workers, because that pattern improves throughput and keeps the system maintainable.[^15][^1]
This is especially useful if you want to run thousands of scenario variants against multiple 5GC versions in CI.[^15][^1][^13]

## Suggested roadmap

Start with one narrow but rich target: **Registration + Authentication + Security Mode + Initial Context Setup**.[^9][^6]
This path covers NAS, NGAP, SCTP, UE identity binding, security context establishment, timer handling, and cross-NF state management, so it is the best first playground for white-grey box fuzzing.[^6][^9]

Then implement in this order:

- Session grammar for the procedure.[^3][^1]
- Structure-aware field mutators.[^8][^9]
- Schedule mutators for delay, reorder, duplicate, replay.[^7]
- Conformance oracles for legal/illegal transitions and expected rejects.[^7][^6]
- Coverage-guided prioritization.[^8][^6]
- Dependency fault injection and NF restart scenarios.[^1][^13]

The most publishable contribution would be: **a white-box, schedule-aware, structure-aware fuzzing emulator for 5GC conformance and robustness**, with model-based state coverage and distributed-system fault injection.[^11][^1][^7]
That is more concrete and novel than “yet another packet mutator,” and it naturally bridges 5G conformance, security, and cloud-native systems testing.[^13][^1][^9]

Would you like the next step to be a **detailed module design** with APIs/data structures, or a **catalog of 50 concrete testcases** for Registration/Security Mode fuzzing?
<span style="display:none">[^16][^17][^18][^19][^20][^21][^22][^23][^24][^25][^26][^27][^28][^29][^30][^31][^32][^33][^34][^35][^36][^37][^38][^39][^40][^41][^42][^43][^44][^45][^46][^47][^48]</span>

<div align="center">⁂</div>

[^1]: Fuzz-Testing-Ideas-from-Recent-Microservice-Distributed-Systems-Work-for-a-UE-gNB-Based-5G-Core-C.md

[^2]: https://arxiv.org/abs/2401.05529

[^3]: https://conf.researchr.org/details/ase-2023/ase-2023-industry-showcase-papers/3/Zero-Config-Fuzzing-for-Microservices

[^4]: https://arxiv.org/abs/2409.02137

[^5]: https://arxiv.org/html/2501.10269v1

[^6]: https://arxiv.org/pdf/2112.04257.pdf

[^7]: https://arxiv.org/pdf/2510.14348.pdf

[^8]: https://arxiv.org/html/2602.21794v1

[^9]: https://arxiv.org/html/2510.14348v1

[^10]: https://research.google/pubs/zero-config-fuzzing-for-microservices/

[^11]: https://arxiv.org/html/2410.02307v3

[^12]: https://arxiv.org/abs/2410.02307

[^13]: https://arxiv.org/pdf/2401.05529.pdf

[^14]: https://arxiv.org/html/2401.05529v1

[^15]: https://huhong789.github.io/papers/chen:mufuzz.pdf

[^16]: https://arxiv.org/pdf/2507.20848.pdf

[^17]: https://arxiv.org/pdf/2507.22442.pdf

[^18]: https://arxiv.org/pdf/2512.04260.pdf

[^19]: https://www.arxiv.org/pdf/2512.06906.pdf

[^20]: https://arxiv.org/pdf/2512.04680.pdf

[^21]: https://www.semanticscholar.org/paper/MICRoFuzz:-An-Efficient-Fuzzing-Framework-for-Di-Liu/a359de873610e2c59e46baae071451674450f32a

[^22]: https://arxiv.org/pdf/2512.08698.pdf

[^23]: https://arxiv.org/html/2407.00225v4

[^24]: https://arxiv.org/pdf/2501.10269.pdf

[^25]: https://arxiv.org/html/2506.15648v2

[^26]: https://arxiv.org/html/2510.10407v2

[^27]: https://www.arxiv.org/pdf/2602.00972.pdf

[^28]: https://dl.acm.org/doi/10.1109/ASE56229.2023.00036

[^29]: https://ieeexplore.ieee.org/iel7/10298258/10298286/10298503.pdf

[^30]: https://arxiv.org/pdf/2208.03988.pdf

[^31]: https://arxiv.org/html/2410.02307v2

[^32]: https://thesai.org/Downloads/Volume15No12/Paper_18-Security_Gap_in_Microservices.pdf

[^33]: https://dl.acm.org/doi/10.1145/3639477.3639723

[^34]: https://arxiv.org/html/2603.08566v1

[^35]: https://arxiv.org/html/2509.19533v1

[^36]: https://arxiv.org/pdf/2509.19533.pdf

[^37]: https://arxiv.org/html/2409.02137v1

[^38]: https://arxiv.org/pdf/2505.03425.pdf

[^39]: https://arxiv.org/pdf/2603.08566.pdf

[^40]: https://arxiv.org/pdf/2503.13551.pdf

[^41]: https://arxiv.org/pdf/2506.23063.pdf

[^42]: https://arxiv.org/html/2508.20816v1

[^43]: https://arxiv.org/pdf/2310.13639.pdf

[^44]: https://www.sciencedirect.com/science/article/pii/S0167642325000619

[^45]: https://pure.mpg.de/rest/items/item_3625115_1/component/file_3625116/content

[^46]: https://arxiv.org/html/2501.10269v2

[^47]: https://github.com/googleprojectzero/SockFuzzer

[^48]: https://dl.acm.org/doi/abs/10.1145/3689779

