# Fuzz Testing Ideas from Recent Microservice & Distributed-Systems Work for a UE+gNB‑Based 5G Core Conformance Emulator

## Overview

This report analyzes ten recent open-access papers on fuzzing for microservices, web APIs, and distributed systems, with the specific goal of identifying ideas that transfer well to a UE+gNB emulator used for 5G core conformance testing.

The main conclusions are:

- The most directly reusable ideas are: schema/grammar-based generation of request *sequences* (not just single calls), structure-aware mutations, and backend/response fuzzing from the microservice/API fuzzing papers; model-guided schedule fuzzing and RL-based guidance from the distributed-systems work; and uncertainty-aware, system-level fault injection from the microservice uncertainty paper.[^1][^2][^3][^4][^5][^6][^7]
- White-box schemes (EvoMaster, MicroFuzz, SandBoxFuzz) are highly relevant if the 5G core under test is your own code (or an open stack you can instrument), but much less so if you only have black-box access via UE+gNB interfaces.[^2][^3][^5][^8]
- µFUZZ contributes a scalable fuzzer *architecture* (microservice-based parallel fuzzing) that can help you drive very high test volumes from a UE+gNB emulator, but does not change what you fuzz.[^9]
- The web-API fuzzing survey is valuable for design space awareness (strategies for HTTP/REST fuzzing that can map to 5G SBI interfaces), but it does not itself propose a concrete new engine.[^10]

## Your Testing Context (Abstraction)

For mapping, it is useful to view your UE+gNB + 5G core setup as follows:

- **System under test (SUT):** 5G core, likely a cloud-native microservice system with HTTP/2+JSON or gRPC SBI between NFs, plus control- and user-plane protocols on N1/N2/N3, PFCP, etc.
- **Fuzzing handle:** UE+gNB emulator that can send arbitrary sequences of NAS/NGAP/RRC messages, open many UEs, manipulate timers, simulate radio failures, and possibly inject traffic patterns that exercise data-plane behavior.
- **Observability:** At minimum, protocol logs and KPIs; possibly deeper logs and internal metrics if the core is under your control.
- **Constraints:** For conformance you care about spec adherence, robustness to malformed messages and sequences, and behavior under timing/fault uncertainties, not only memory-safety bugs.

All selected papers target *service- or protocol-level* fuzzing and are therefore conceptually quite close to this setting.

## High-Relevance Ideas to Reuse

### Session- and Schema-Based Fuzzing (Zero-Config Microservices, EvoMaster RPC/REST)

**Zero-Config Fuzzing for Microservices (Wang et al., ASE 2023)** proposes structure-aware, coverage-guided fuzzing for gRPC microservices, auto-generating a `FuzzSession` message that represents a *sequence* of RPC calls plus (in an extended form) backend responses.[^7]

Key ideas that map well to a UE+gNB-based 5G core fuzzer:

- **Automatically derived grammar for sequences:** They build a Protobuf `FuzzSession` with a repeated `RpcCall` union over all service methods, allowing the fuzzer to generate *sequences* of mixed RPCs, not just single calls.[^7]
  - Analog in 5G: define an intermediate grammar (ASN.1-derived, Protobuf, or similar) for a `FuzzSession` that encodes a sequence of NAS/NGAP procedures (e.g., Registration, Service Request, PDU Session Establishment, Handover) and their parameters.
- **Structure-aware mutations:** Use libprotobuf-mutator-style structure-aware mutation so that most generated messages remain syntactically valid at the protocol layer while exercising corner cases in fields and combinations.[^7]
  - Analog: structure-aware mutation of NAS/NGAP IEs (e.g., GUAMI, PLMN, S-NSSAI, QoS rules), preserving basic constraints but exploring rare combinations.
- **Backend fuzzing:** They intercept and fuzz *responses* from dependent microservices, generating mutated backend responses and errors in a hermetic environment to explore how the front service reacts to abnormal backends.[^7]
  - Analog: if you also own a 5G core implementation (or a test double of it), you can invert the setup and fuzz UE+gNB behavior by mutating core responses (e.g., SMF responses, handover commands) to probe robustness of RRC/NAS handling in the emulator or in a reference stack.

**White-Box Fuzzing RPC-Based APIs with EvoMaster (Zhang et al., TOSEM 2023)** and **Advanced White-Box Heuristics for Search-Based Fuzzing of REST APIs (Arcuri et al., TOSEM 2024)** extend EvoMaster with:

- **Schema extraction:** Parsing RPC IDLs (Thrift, gRPC, SOFARPC, Dubbo) or OpenAPI/SQL schemas to derive an internal model of endpoints, data types, and dependencies.[^3][^5]
- **Search-based test generation:** Using evolutionary search (MIO) with white-box guidance and domain-specific heuristics to generate sequences of calls that satisfy data dependencies and drive coverage.[^5][^3]
- **Handling under-specified schemas and DB constraints:** Extra heuristics for missing schema constraints (e.g., ranges) and for joint reasoning over HTTP payloads and database state.[^5]

Potential mapping:

- For **SBI and management APIs** (REST/gRPC) between NFs (e.g., AMF–SMF–UDM–PCF) you can adopt the EvoMaster pattern directly: generate API call sequences per NF, guided by coverage and domain constraints.[^3][^5]
- For **UE+gNB side**, you can treat the NGAP/NAS ASN.1 and 3GPP state machines as your “schema” and define search operators over message sequences and IE values, using coverage or oracle violations in the core as the fitness signal.

These works are white-box (they instrument code), but the *high-level ideas—schema/grammar extraction, sequence generation, and search-based mutation—remain valid in a black-box protocol setting*.

### Uncertainty-Driven, System-Level Microservice Fuzzing

**Fuzzing Microservices in Face of Intrinsic Uncertainties (Zhang et al., arXiv 2026)** is a position paper that argues for “uncertainty-driven” and “system-level” fuzzing of microservice systems.[^1]

Core ideas relevant to a 5G deployment:

- **Modeling multi-dimensional uncertainties:** They classify uncertainties across inputs, network, runtime environment, and internal logic (e.g., randomization, contention), and propose an UncerMML model to describe them.[^1]
  - For 5G, uncertainties include radio conditions, UE mobility, timer jitter, partial failures of NFs, and overload on specific interfaces.
- **Service virtualization and uncertainty simulation:** The vision includes virtualizing dependent services and injecting controlled uncertainty (e.g., delays, drops, wrong versions) to observe propagation through the microservice graph.[^1]
  - For a 5G core, this is very close to injecting impairments on specific internal links (e.g., between AMF–SMF–UPF or towards UDM) while driving realistic UE traffic.
- **Dynamic causal graphs and impact analysis:** They propose building causal graphs of uncertainty propagation to identify critical services and paths, and quantifying impact on QoS metrics (availability, throughput, error rates).[^1]

For your UE+gNB emulator, this suggests:

- Design fuzzing not only at the **syntax/sequence level** of NAS/NGAP but also at the **uncertainty level**: controlled variation in timing, loss, reordering, back-pressure, and NF availability.
- Combine protocol-level fuzzing with **metrics collection** (e.g., success rate, latency distributions, stability of UE count) and simple causal analysis to localize where the core starts misbehaving under joint protocol and uncertainty stress.

### Model-Guided and RL-Guided Schedule Fuzzing for Distributed Systems

**Model-Guided Fuzzing of Distributed Systems (Gulcan et al., OOPSLA 2025)** introduces ModelFuzz, which uses a TLA+ model of a distributed protocol to define coverage over *abstract states* and guides a fuzzing engine that explores message delivery schedules.[^6]

Key concepts highly applicable to 5G protocol testing:

- **Abstract protocol model as coverage oracle:** They use an abstract TLA+ model of protocols (Raft, Two-Phase Commit) and treat newly reached model states as coverage goals; the fuzzer mutates message schedules to find executions that reach new states.[^6]
- **Schedule-level mutations:** Test cases are sequences of message-delivery and crash events; mutations reorder deliveries or change which process crashes/restarts to explore different behaviors.[^6]
- **Bug finding in distributed implementations:** The approach found multiple previously unknown bugs in Etcd and RedisRaft, particularly in corner-case schedules that standard fuzzers or RL-guided testers missed.[^6]

For 5G:

- 3GPP already gives you **state machines** for procedures like Registration, Service Request, Handover, and PDU Session Establishment; these can be encoded in TLA+ or another formalism.
- Your UE+gNB fuzzer can then treat **test cases as schedules** of events such as “UE i sends NGAP X”, “core sends Y”, “message dropped”, “timer expired”, “NF restarted”.
- You can guide test generation by **model state coverage**, targeting rarely visited abstract states such as race conditions between handover and re-authentication, or between SMF and UPF updates.

**Reward Augmentation in Reinforcement Learning for Testing Distributed Systems (Borgarelli et al., OOPSLA 2024)** (BonusMaxRL) uses RL with a carefully designed reward structure to explore distributed protocol state spaces (RedisRaft, Etcd, RSL).[^4]

Core ideas with clear analogues in your domain:

- **Decaying exploration bonus:** Rewards for visiting new abstract “colors” (compressed states) decay as they are revisited; this is a principled form of coverage-guided exploration that works well with RL.[^4]
- **Waypoints (semantic predicates):** They define predicates that capture semantically meaningful scenarios (e.g., “there exists a leader in term t” in Raft) and use them as intermediate rewards to bias exploration toward deep, interesting behaviors.[^4]
- **Environment design parameters:** Colors, crash injection limits, and tick granularity are tuned to balance tractability and expressivity.[^4]

Analog in 5G:

- Use an RL agent to control **which UEs, which procedures, and which faults/impairments** your emulator applies over time.
- Define **waypoint predicates** in terms of 5G-core states, such as “X% of UEs are in CM-CONNECTED with active PDU sessions”, “UPF has non-empty forwarding table but SMF thinks session is released”, “AMF is in overload mode while new emergency calls arrive”.
- Use RL reward shaping to drive the system into these scenarios and then encourage exploration around them with additional fuzzing of NAS/NGAP parameters.

Together, ModelFuzz and BonusMaxRL give a blueprint for **stateful, schedule-aware fuzzing of distributed 5G cores**, which is more powerful than stateless mutation of individual messages.

### Microservice Fuzzer Architectures and Integration in Industrial Pipelines

Several papers address how to *integrate* fuzzing into large-scale, production-like environments, which is important if your UE+gNB emulator must plug into existing CI/CD or test frameworks.

**MicroFuzz: An Efficient Fuzzing Framework for Microservices (Di et al., ICSE-SEIP 2024)** describes a microservice fuzzing framework deployed at Ant Group.[^2]

Key ideas:

- **Mocking-assisted seed execution:** Identify system, internal, and external dependencies as “mocking points” and record/replace their values during fuzzing runs to reduce non-determinism and avoid hitting real backends.[^2]
- **Distributed tracing for coverage:** Use distributed tracing agents and a central collector to reconstruct full coverage across multiple services.[^2]
- **Seed refresh lifecycle management:** Periodically refresh seeds when upstream services change versions to maintain consistency with an evolving microservice landscape.[^2]
- **Pipeline parallelism:** Decouple fuzzing phases (mutation, execution, coverage analysis) so they can run in parallel across the networked microservice deployment.[^2]

If you own or instrument the 5G core (e.g., open 5G core stack):

- Use **mocking points** to isolate external dependencies (e.g., external databases, charging systems) while fuzzing internal NFs under UE load.
- Apply **distributed tracing** or similar observability to get end-to-end coverage of NF chains triggered by UE procedures.
- Use **pipeline parallelism** to run large numbers of UE-driven tests while asynchronously analyzing logs and coverage.

**SandBoxFuzz: Grey-Box Fuzzing in Constrained Ultra-Large Systems (Yu et al., FSE Companion 2025)** tackles constraints common in enterprise settings: limited source access, fixed test frameworks (iTest), and high startup costs.[^8]

Important techniques:

- **Sandbox interception:** Intercept inputs at specific “pointcuts” inside the test framework and mutate *in memory*, while leaving the surrounding test harness and configuration files untouched.[^8]
- **Specification mining via reflection:** Automatically infer input structure (types, nested fields) at runtime from Java objects, avoiding manual spec definition.[^8]
- **Log-based coverage:** Instrument bytecode with print statements and reconstruct coverage offline from test logs, which is compatible with locked-down environments.[^8]

Possible transfer to 5G testing:

- If you must work inside an existing **commercial test framework** (e.g., vendor-specific conformance environment or TTCN-3 harness) that you cannot modify deeply, SandBoxFuzz’s idea of **intercepting and mutating inputs at runtime** is directly applicable: you can run standard conformance tests but, just before they send messages, intercept and fuzz parts of the NAS/NGAP/RRC messages.
- If your emulator is built in a language with rich reflection (Java/Kotlin/Scala for AMF/SMF, or maybe C++/Rust with metadata), you can mine message structure dynamically rather than hardcoding ASN.1 mappings.

**µFUZZ: Redesign of Parallel Fuzzing using Microservice Architecture (Chen et al., USENIX Security 2023)** is less about what to fuzz, more about *how to build a high-throughput fuzzer*.[^9]

Notable ideas:

- **Microservice decomposition of the fuzzer itself:** Separate services for corpus management, mutation, execution, coverage, etc., each with multiple workers, orchestrated by an async runtime (Tokio).[^9]
- **Zero-copy communication via shared memory and pointer passing:** To avoid bottlenecks when transferring many test cases among components.[^9]
- **State partitioning to avoid synchronization:** Partition fuzzing state across workers to reduce synchronization overhead, yet achieve strong aggregate performance.[^9]

For a UE+gNB emulator that must drive **many thousands of concurrent UEs and high message volumes**, adopting µFUZZ’s architectural patterns can significantly improve throughput, especially if you deploy the fuzzer itself as a microservice application.

### Design Space Awareness from Web-API Fuzzing Survey

**Fuzzing frameworks for server-side web applications: a survey (Dharmaadi et al., Int. J. Inf. Security 2025)** systematically reviews web-API fuzzers (RESTler, RestTestGen, BackREST, etc.) and classifies strategies along input generation, feedback, and mutation dimensions.[^10]

Useful takeaways:

- **Input generation:** Grammar-based vs. mutation-based; REST fuzzers often use OpenAPI specs and dependency graphs to generate request templates and sequences.[^10]
- **Feedback:** HTTP status codes, taint-based detection of vulnerabilities (XSS/SQLi), code coverage via source instrumentation or interpreter augmentation.[^10]
- **Mutation operators:** Randomized value mutation, response dictionaries (reusing values seen in responses), adaptive hypermutation, rule-based schema mutators, vulnerability dictionaries.[^10]

Mapping to 5G:

- Treat **3GPP specs as “schema”** and 5G ASN.1 as your grammar; adopt response dictionaries (e.g., reusing real GUTIs, TEIDs, QoS rules) to keep tests realistic but still explore new combinations.
- Consider **vulnerability-focused fuzzing** analogues for 5G conformance, where you target specific property violations (e.g., security properties from TS 33.501) rather than only crashes.

## Per-Paper Summary and Suitability Table

The table below summarizes each paper, its main contribution, and how suitable its ideas are for a UE+gNB-based 5G core conformance fuzzer.

| Paper | Domain/Target | Core idea | Suitability for UE+gNB 5GC conformance | Key ideas to borrow |
|------|---------------|-----------|-----------------------------------------|----------------------|
| Zhang et al., “Fuzzing Microservices in Face of Intrinsic Uncertainties” (arXiv 2026)[^1] | Microservice systems under uncertainty | Vision for uncertainty-driven, system-level microservice fuzzing with explicit modeling, injection, and analysis of diverse uncertainties and their propagation. | **High** if you want to go beyond pure protocol fuzzing to include timing, network, and NF-failure uncertainties in your 5G tests. | Taxonomy of uncertainties, service virtualization, dynamic causal graphs, and metrics-driven impact analysis; can structure a 5G “uncertainty test plan.” |
| Di et al., “MicroFuzz” (ICSE-SEIP 2024)[^2] | Microservices (CICD, Ant Group) | Industrial microservice fuzzing with mocking-assisted seed execution, distributed tracing for coverage, seed refresh lifecycle, and pipeline parallelism. | **Medium–High** if you instrument an in-house or open 5G core; lower if testing closed vendor cores. | Mocking points to isolate dependencies, distributed coverage collection, and decoupled fuzzing pipeline for large-scale UE-driven tests. |
| Zhang et al., “White-Box Fuzzing RPC-Based APIs with EvoMaster” (TOSEM 2023)[^3] | RPC-based microservices | White-box, search-based fuzzing of Thrift/gRPC-like RPC APIs, with schema extraction and specialized heuristics, deployed in industrial pipeline. | **High** for SBI/RPC interfaces or if your 5G NFs expose gRPC APIs; conceptually strong for message-grammar and sequence-based fuzzing of NAS/NGAP too. | RPC schema abstraction, search-based sequence generation, and industrial integration patterns. |
| Borgarelli et al., “Reward Augmentation in RL for Testing Distributed Systems” (OOPSLA 2024)[^4] | Distributed protocols (Raft, etc.) | RL-based test generation with decaying exploration bonus and waypoint predicates over abstract protocol states. | **High** if you want to use RL for intelligent scheduling of UE actions, faults, and traffic scenarios in 5G. | Reward shaping via coverage-like bonuses and semantic waypoints; careful design of abstract state (“color”) for 5G protocols. |
| Arcuri et al., “Advanced White-Box Heuristics for Search-Based Fuzzing of REST APIs” (TOSEM 2024)[^5] | REST APIs (web, microservices) | New white-box heuristics in EvoMaster to handle under-specified API and DB schemas, improving coverage and fault finding. | **Medium–High** for fuzzing 5G SBI (HTTP/2+JSON) and management APIs; less direct for NAS/NGAP but conceptually reusable. | Handling under-specified schemas, joint reasoning about HTTP payloads and DB state, evolutionary fuzzing of API sequences. |
| Yu et al., “SandBoxFuzz” (FSE Companion 2025)[^8] | Constrained industrial microservices with fixed test framework | Grey-box fuzzing via sandbox interception in iTest, spec mining via reflection, and log-based coverage, with strong deployment experience at Ant Group. | **Medium** when integrating fuzzing into commercial or proprietary 5G test frameworks where you cannot change harness logic. | Runtime interception of inputs, auto spec mining from in-memory objects, and log-based coverage reconstruction. |
| Gulcan et al., “Model-Guided Fuzzing of Distributed Systems” (OOPSLA 2025)[^6] | Distributed protocols (Raft, 2PC) | ModelFuzz: coverage-guided schedule fuzzing using abstract TLA+ models as coverage oracles over states, mutating event schedules. | **Very High** for stateful, schedule-aware fuzzing of 5G core behavior driven by UE+gNB, especially if you write TLA+ models of 5G procedures. | Abstract protocol model as coverage, schedule mutation operators over message deliveries and crashes, and controlled scheduler architecture. |
| Dharmaadi et al., “Fuzzing frameworks for server-side web applications: a survey” (IJIS 2025)[^10] | Web/REST fuzzers | Survey of web-API fuzzing strategies: request generation, feedback, mutation, benchmarks, and open challenges (incl. microservices). | **Medium** as a design reference when choosing HTTP/REST fuzzing strategies for 5G SBI; indirect for NAS/NGAP. | Taxonomy of fuzzers (grammar vs. mutation, crash vs. vulnerability-driven), feedback strategies, and mutation operators (e.g., response dictionaries, adaptive hypermutation). |
| Wang et al., “Zero-Config Fuzzing for Microservices” (ASE 2023)[^7] | gRPC microservices | Zero-config, structure-aware fuzzing of microservices using auto-generated `FuzzSession` Protobufs for call sequences and backend fuzzing for dependent services. | **Very High**: the FuzzSession/structure-aware pattern is almost directly applicable to defining and mutating sequences of NAS/NGAP/RRC messages in your UE+gNB emulator. | Automatically generated session grammar, structure-aware mutations via libprotobuf-mutator, and backend-response fuzzing in hermetic mode. |
| Chen et al., “µFUZZ: Redesign of Parallel Fuzzing using Microservice Architecture” (USENIX Security 2023)[^9] | Fuzzer infrastructure | Parallel fuzzing architecture decomposed into microservices, with zero-copy communication and state partitioning; strong throughput improvements. | **Medium–High** if your main bottleneck is fuzzing throughput (many UEs/messages), not concept design. | Microservice-based fuzzer architecture, zero-copy shared-memory queues, and asynchronous execution pipeline for large-scale fuzzing. |

## How These Ideas Can Shape a 5G UE+gNB Fuzzer

Based on the above, a 5G-focused fuzzing design could:

- Use **Zero-Config-style session grammars** (possibly Protobuf or a custom IR) to encode sequences of NAS/NGAP/RRC messages and their parameters.[^7]
- Apply **structure-aware, search-based, and RL-guided mutations** over these sequences, guided by a mix of:
  - **Abstract protocol model coverage** (ModelFuzz-style state coverage from TLA+ models of key 5G procedures).[^6]
  - **Semantic waypoints and rewards** (RL reward augmentation ideas) that target interesting 5G scenarios.[^4]
  - **System-level uncertainty scenarios** (uncertainty-driven microservice fuzzing) such as NF restarts, delays, overload, and partial failures.[^1]
- Architect the fuzzer itself as a **microservice pipeline** (µFUZZ, MicroFuzz) so that it can scale to tens of thousands of UEs and maintain high throughput while integrating with existing CI/CD or test frameworks (SandBoxFuzz-style interception if needed).[^8][^9][^2]
- For **SBI and management APIs**, reuse EvoMaster’s REST/RPC fuzzing concepts directly to fuzz NF APIs under the same UE-driven scenarios.[^3][^5]

This combination would give you a principled, scalable framework that is aligned with the literature but tailored to the specifics of 5G core conformance via UE+gNB emulation.

---

## References

1. [2603.02551v1.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/ea0a45ba-4f3a-4394-8fc0-6db45767eb0f/2603.02551v1.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=uzCBH4eAJD4kelhndRwZqUiSOcQ%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - Fuzzing Microservices in Face of Intrinsic Uncertainties
Man Zhang1, Tao Yue∗1, and Andrea Arcuri2
1...

2. [3639477.3639723.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/c273d15d-ff66-4eee-8aa7-fdfa60c6a417/3639477.3639723.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=5IM%2FqRD6NbgMnKL2xU4AG0%2BkI34%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - .
.
Latest updates: hps://dl.acm.org/doi/10.1145/3639477.3639723
.
.
RESEARCH-ARTICLE
MicroFuzz: An...

3. [3585009.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/ae52feff-b82e-45cc-bf2a-11a2ebb469bf/3585009.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=DX%2FDm7IhtNT4N4SwdNHtbYozjbc%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - .
.
Latest updates: hps://dl.acm.org/doi/10.1145/3585009
.
.
RESEARCH-ARTICLE
White-Box Fuzzing RPC...

4. [3689779.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/a9be3152-495a-4220-ba1b-29a5f9558e0f/3689779.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=2CBkIqdjI2nphQBcT9xq6bfHxu4%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - .
.
Latest updates: hps://dl.acm.org/doi/10.1145/3689779
.
.
RESEARCH-ARTICLE
Reward Augmentation i...

5. [3652157.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/ad893595-3ecd-4ded-8fe3-daf75074d5a7/3652157.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=miH4YapHRE8FgOUqONh2GVK%2BP%2FM%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - .
.
Latest updates: hps://dl.acm.org/doi/10.1145/3652157
.
.
RESEARCH-ARTICLE
Advanced White-Box He...

6. [3763060.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/8c09eca1-6882-44ba-9913-d01534ed5b8c/3763060.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=0VBbcPUMRIZ0nNn5XtzSCX3guc4%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - .
.
Latest updates: hps://dl.acm.org/doi/10.1145/3763060
.
.
RESEARCH-ARTICLE
Model-Guided Fuzzing ...

7. [Zero-Config_Fuzzing_for_Microservices.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/61012a91-12d4-4b7e-9877-85027d0690fa/Zero-Config_Fuzzing_for_Microservices.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=qWKpQJNlo0amGiF5M7Sbq74B9MM%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - Zero-Config Fuzzing for Microservices
Wei Wang
Google Inc.
wwweiwang@google.com
Andrei Benea
Google ...

8. [3696630.3728546.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/18735e39-c369-4afd-b52a-7ab04f24e6eb/3696630.3728546.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=GQIhs1ooNr4jGx2GDF3xjf3KOMw%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - Grey-Box Fuzzing in Constrained Ultra-Large Systems: Lessons
for SE Community
Jiazhao Yu
Sun Yat-sen...

9. [usenixsecurity23-chen-yongheng.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/bdef4d04-92d1-4f71-b249-5960e280e94e/usenixsecurity23-chen-yongheng.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=BbzsAflMQFUhiFidF%2FhUl1YLxGY%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - This paper is included in the Proceedings of the 
32nd USENIX Security Symposium.
August 9–11, 2023 ...

10. [s10207-024-00979-w.pdf](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/335307993/e5689be0-d9b2-4b2a-a9bb-d5ca31424d5e/s10207-024-00979-w.pdf?AWSAccessKeyId=ASIA2F3EMEYE4PCRKIBY&Signature=1tD07AvHr79FZ93kdcKE%2BpnZ3Xc%3D&x-amz-security-token=IQoJb3JpZ2luX2VjEGMaCXVzLWVhc3QtMSJHMEUCIDlrkPRYKzietc5Ksd1rsmd%2FH6%2F%2F1Whul6ouPxBf6P3TAiEA%2FuFjjCweVpmPJ2mdT4BIhEry30P%2BSnHiBj4pPKYO%2BPMq8wQILBABGgw2OTk3NTMzMDk3MDUiDCXqme4F6a8qIP6ncSrQBHKU2QQRjxctbH52SaO71cvoGOQOaWv04D2HR%2BEmdInQr3je6qwCsKjHQHsM3OZMXfvtKFX9ZQ9of1ldn1UNjD%2F26p4nDNXr0jSzMB8lfKHnIzAnGFfbeUiW6YaYwn6ejq4XF6uVKJW1aj1D3d56B4OggJxXLx7NIv2%2B3oiVV63ltPM8HOEHKz4xVnbYGWib1A2t50JT%2BaR4%2FiO8MbEQhGeS18J52jJShkIdHe4M5W%2BVibnf8lK5pENexYIhS8VxLEyWqWUM9J3DUU2Rpqov431lXArIQoyx9iuk2rJW3%2FdWPQI466h7%2BkNSy4LvFsfeVrbzgzvqMzrPFrxNPH0Bp5TAzQ2MSlz8rk25K21hthB6XRJedXCmarEnmxeRPhut9PgnEe9XrVxcYtSH2J7peY5O0vY87DbipwrlrjtqY6kWNTkXnBl7wBmWSzUcqAHJaVTAdur826OLpC1V9SfK2CFjTltzdyBJFTs3jYCJHLG9CK4afyxTaZZTh02Rc7ZWeL%2BN4av09hsmVtrSAJTTgPUcu5d4NMg72457Hiyr%2Fe0URuwEgFYLCfuESxMjgOZXKI4UcaqqtYyRm6btC7lHOG%2F5AtHy40EviTqOIRB0YRVvRJVjVbzM%2BcfzAMqe%2BMwLDM93mjZVbV3p8QkFzNz9E83QzhrAuoMhaFq5pwasBYf5ZxUZoHDjsfjT%2B%2B3Xzk1f0L4D1lHNGIVJxcp9jYWkCvZcOs%2BKKzZsjtCie7IokVPp4nC02IwJp%2FfXBEXrx7Y62I9LKHGdXGeoGqPCqNJbCEAw1%2BbyzQY6mAE12eGzOHCeC3%2BckJgOShDDv8C58SSfClc96GEaU0UhtpRW33xS8las5fdouUv1S%2B%2BCYe2wBdyZ8uJEhXbTAQPuKnws0NvO9DtMPXLCp2px55wK2CSWPpkhssWIry%2F8%2FAZrM%2FEB7SABC6%2FMT%2BIdtScP4N%2B8uSlFG1ijFEeXQegFxSByVjl8xU4s1xcmM8xl29d%2FWnWjN%2FRP0Q%3D%3D&Expires=1773977898) - International Journal of Information Security (2025) 24:73
https://doi.org/10.1007/s10207-024-00979-...

