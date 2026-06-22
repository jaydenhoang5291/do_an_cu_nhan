Now I have the complete picture - all the research, the existing code architecture, and the current rudimentary `FsmFuzzer`. Let me produce the grand vision.

---

# StormFuzz -- Adversarial Conformance Intelligence for 5G Core

## BMAD Creative Intelligence Suite -- Full Analysis

---

## Phase 1: Dr. Quinn -- Root Cause Analysis

### Why Current 5G Fuzzers Are Insufficient


| Fuzzer       | What It Does                                          | What It Misses                                                      |
| ------------ | ----------------------------------------------------- | ------------------------------------------------------------------- |
| **5GReplay** | Replay pcap + rule-based field mutation via DPI       | No runtime state awareness, no NAS security re-signing, no multi-UE |
| **AMFuzz**   | Proxy intercept + IE-aware mutation + NAS re-sign     | Single-UE only, no schedule fuzzing, no coverage feedback           |
| **CovFUZZ**  | Coverage-guided downlink fuzzing with gNB hooks       | Downlink only, requires gNB source modification, no uplink/multi-UE |
| **5GC-Fuzz** | Stateful black-box, learns FSM then mutates sequences | External observer only, cannot re-sign NAS, no population-scale     |


### Five Whys: Why Can't They Find the Hard Bugs?

1. They mutate one message or one session at a time
2. 5GC's hardest bugs live in **cross-UE interactions**, concurrent state machines, and resource contention
3. No tool operates at **population level** -- they all test UE-by-UE
4. Building a high-fidelity, high-scale UE/gNB emulator is the hard infrastructure problem
5. **StormSIM already solved that problem** -- it just hasn't weaponized it yet

### The TRIZ Resolution

**Contradiction**: More concurrent scenarios (thoroughness) vs. exploding state space (tractability).

**Resolution**: Don't fuzz each UE independently. Fuzz the **collective behavior** -- treat the UE swarm as one organism with coordinated mutations. Operate on **swarm strategies** (e.g., "50% register, 30% handover, 20% attack") rather than per-UE mutations.

---

## Phase 2: Victor -- Blue Ocean Strategy

### The Uncontested Space

No tool in the literature or industry combines all four:


| Dimension                        | Existing Tools                         | StormFuzz                                                |
| -------------------------------- | -------------------------------------- | -------------------------------------------------------- |
| **Conformance Oracle**           | TTCN-3 (deterministic, no adversarial) | 3GPP spec-model verdicts (PASS/FAIL/ANOMALY/UNREACHABLE) |
| **Population-scale Adversarial** | None (all single-UE)                   | 10,000+ UEs as coordinated adversary                     |
| **Schedule Fuzzing**             | None for 5G                            | Event ordering, timing, faults across UEs/gNBs           |
| **RL-guided Exploration**        | None for 5G                            | Intelligent swarm strategy with formal-model waypoints   |
| **Mutation Depth**               | Packet bytes or single IE              | Content + Schedule + Infrastructure, 3 dimensions        |
| **Instrumentation**              | One-sided or black-box                 | Dual-side white-box (emulator + core)                    |


StormSIM's existing `FsmFuzzer` in `fsm_fuzz.go` is currently a random state/event injector -- the seed of something much bigger. The FSM framework, worker pools, NAS/NGAP libraries, security context, and timer engine are the **infrastructure moat** that no competitor has.

---

## Phase 3: Carson -- The Grand Architecture

### StormFuzz: 6-Layer Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                          LAYER 6: RL CONDUCTOR                               │
│                                                                              │
│  RL Agent (BonusMaxRL + WaypointRL composite)                                │
│  ┌────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐  │
│  │ Swarm Strategy     │  │ Reward Engine        │  │ Waypoint Registry    │  │
│  │ Generator          │  │                      │  │                      │  │
│  │ • Population mix   │  │ score = a*state_nov  │  │ • UE registered      │  │
│  │ • Role assignment  │  │   + b*coverage_nov   │  │   without security   │  │
│  │ • Coordination     │  │   + c*semantic_anom  │  │ • GUTI collision     │  │
│  │   patterns         │  │   + d*cross_core_div │  │   detected           │  │
│  │ • Fault injection  │  │   - e*exec_cost      │  │ • Emergency during   │  │
│  │   schedule         │  │                      │  │   overload           │  │
│  └────────────────────┘  └──────────────────────┘  │ • Handover + Dereg   │  │
│                                                    │   race               │  │
│                                                    └──────────────────────┘  │
├──────────────────────────────────────────────────────────────────────────────┤
│                     LAYER 5: SPEC MODEL ORACLE                               │
│                                                                              │
│  3GPP FSMs encoded as formal property checkers                               │
│  ┌───────────────────┐  ┌───────────────────────┐  ┌────────────────────────┐│
│  │ State Coverage    │  │ Conformance Checker   │  │ Differential Oracle    ││
│  │ Tracker           │  │                       │  │                        ││
│  │ • Abstract state  │  │ • TS 24.501 (NAS)     │  │ • Open5GS vs free5GC   ││
│  │   visited/total   │  │ • TS 38.413 (NGAP)    │  │ • Same input,          ││
│  │ • Transition      │  │ • TS 33.501 (Security)│  │   different behavior   ││
│  │   coverage map    │  │ • Verdict: PASS/FAIL/ │  │   = implementation     ││
│  │ • Cross-entity    │  │   ANOMALY/UNREACHABLE │  │   bug                  ││
│  │   state combos    │  │                       │  │                        ││
│  └───────────────────┘  └───────────────────────┘  └────────────────────────┘│
├──────────────────────────────────────────────────────────────────────────────┤
│                  LAYER 4: 3-DIMENSIONAL MUTATION ENGINE                      │
│                                                                              │
│  DIM 1: CONTENT MUTATION       DIM 2: SCHEDULE MUTATION                      │
│  ┌───────────────────────┐    ┌──────────────────────────────┐               │
│  │ IE-Aware Mutators     │    │ Swap(M_i, M_j)               │               │
│  │ • enum invalid-but-   │    │ Drop(M_i)                    │               │
│  │   decodable           │    │ Delay(M_i, dt)               │               │
│  │ • length boundary     │    │ Duplicate(M_i, k)            │               │
│  │ • optional IE toggle  │    │ Replay(M_i, old_ctx)         │               │
│  │ • wrong identity type │    │ Reorder across UEs/gNBs      │               │
│  │ Cross-Field Mutators  │    │ InterleaveProc(MM, SM)       │               │
│  │ • stale security ctx  │    └──────────────────────────────┘               │
│  │ • mismatched NSSAI    │                                                   │
│  │ Concurrency Mutators  │     DIM 3: INFRASTRUCTURE MUTATION                │
│  │ • same SUPI N UEs     │     ┌──────────────────────────────┐              │
│  │ • parallel deregister │     │ NF restart mid-procedure     │              │
│  │ • competing gNBs      │     │ SCTP association abort       │              │
│  └───────────────────────┘     │ SBI delay/timeout injection  │              │
│                                │ Timer jitter / expiry force  │              │
│                                │ Overload / back-pressure     │              │
│                                └──────────────────────────────┘              │
├──────────────────────────────────────────────────────────────────────────────┤
│               LAYER 3: MESSAGE HOOK + SECURITY RE-SIGNING                    │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐    │
│  │ Intercept NAS/NGAP BEFORE encryption/integrity                       │    │
│  │ → Apply mutation from Layer 4                                        │    │
│  │ → Re-compute NAS MAC + re-encrypt with KNASint/KNASenc               │    │
│  │ → Update NAS sequence number                                         │    │
│  │ → Forward to core (OR drop/delay per schedule mutation)              │    │
│  └──────────────────────────────────────────────────────────────────────┘    │
├──────────────────────────────────────────────────────────────────────────────┤
│               LAYER 2: SCENARIO DRIVER (uses existing FSM)                   │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐    │
│  │ set_target_state(ue_id, target_state)                                │    │
│  │   → UE FSM autonomously executes Registration, Auth, SecMode, etc.   │    │
│  │   → Reaches desired state for mutation injection                     │    │
│  │                                                                      │    │
│  │ Swarm orchestration:                                                 │    │
│  │   assign_roles(ue_group_A: "register", ue_group_B: "handover",       │    │
│  │                 ue_group_C: "attack_identity_collision")             │    │
│  └──────────────────────────────────────────────────────────────────────┘    │
├──────────────────────────────────────────────────────────────────────────────┤
│               LAYER 1: STORMSIM CORE (existing infrastructure)               │
│                                                                              │
│  10,000+ UEs · 100+ gNBs · FSM Engine · Worker Pools (MM/SM/GNB/SCTP)        │
│  SCTP/NGAP · NAS Library · Security Context · GTP-U · Ring Buffers           │
│  RLink Virtual Radio · Timer Engine · Task Queue                             │
│                                                                              │
│                    ↕ N2/N1/N3                                                │
│            ┌─────────────────────┐                                           │
│            │   5G Core (SUT)     │ ← White-box instrumentation optional      │
│            │   Open5GS / free5GC │                                           │
│            └─────────────────────┘                                           │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 4: Maya -- What StormFuzz Actually Attacks

### 7 Attack Families (Population-Scale)

**Family 1: Identity Collision Swarm**

- 1000 UEs register with overlapping/colliding GUTIs, SUPIs, or RAN_UE_NGAP_IDs
- Valid NAS PDU under the wrong `RAN_UE_NGAP_ID`
- Cross-UE context mixup: UE-A's NAS goes through UE-B's NGAP context
- *Single-UE fuzzers cannot find this class of bugs*
- **Target NF**: AMF (UE context management)

**Family 2: Temporal Pincer Attacks**

- Group A executes Handover while Group B simultaneously triggers Deregistration for same AMF
- Group A floods Registration while Group B replays old Security Mode Complete messages
- Concurrent PDU Session Establishment + Release for same session ID
- **Target NF**: AMF, SMF (cross-procedure races)

**Family 3: Security State Desynchronization**

- Duplicate Security Mode Complete
- Replay old Authentication Response after new RAND
- Send Registration Complete before security establishment
- Mismatch NAS sequence number, KSI, or selected security algorithm
- Stale NAS security context + new NAS count
- **Target NF**: AMF, AUSF (security state machine)

**Family 4: Ghost Context Attacks**

- Release UE contexts on emulator side but continue sending valid NAS under old IDs
- 5000 ghost contexts + 5000 legitimate UEs competing for resources
- Test whether core properly garbage-collects and rejects stale references
- **Target NF**: AMF, SMF, UPF (context lifecycle)

**Family 5: Protocol State Maze (RL-driven)**

- RL drives UEs through rare state combinations:
  - Emergency Registration during Handover
  - PDU Session Modification during Re-authentication
  - Service Request during Paging
  - Multiple PDU sessions with conflicting QoS during overload
- Formal model tracks which state combinations have been visited
- Decaying reward bonus drives into deep corners
- **Target NF**: All NFs (cross-entity state space exploration)

**Family 6: Infrastructure Earthquake**

- Controlled NF restarts while thousands of procedures are in flight
- SCTP association failures mid-authentication
- SBI timeouts between AMF-SMF-UDM during active sessions
- Timer jitter: T3510/T3560 expiry exactly before delayed response arrives
- **Target NF**: All NFs (resilience and recovery)

**Family 7: Cascade Failure Probes**

- RL conductor learns exactly the load pattern that causes SMF to slow down
- Then exploits resulting timer mismatches across 500 active PDU sessions
- Tests emergent behavior under realistic stress, not synthetic overload
- **Target NF**: System-level emergent behavior

---

## Phase 5: The Concrete Plan -- How to Build This

### Step 1: Evolve `FsmFuzzer` into Message Hook Layer

The existing `fsm_fuzz.go` randomly injects states/events. Evolve this into a **targeted mutation injection point**:

```go
type FuzzHook struct {
    Enabled       bool
    MutationMode  MutationMode   // CONTENT, SCHEDULE, INFRA, COMBINED
    Target        FuzzTarget     // Which message types to intercept
    Mutators      []Mutator      // Chain of mutation operators
    SecurityResigner *NASResigner // Re-sign NAS after mutation
}

type Mutator interface {
    Mutate(msg *NASMessage, ctx *FuzzContext) (*NASMessage, MutationRecord)
    Applicable(msgType nas.MessageType) bool
}
```

The hook sits between `trigger.go` (where NAS messages are built) and the actual SCTP send, intercepting messages **before** encryption/integrity so mutations can be applied cleanly, then re-signed.

### Step 2: Build IE-Aware Mutator Library

Leverage the existing NAS library. Each mutator is message-type-aware:


| Mutator Class       | Examples                                                            | Priority |
| ------------------- | ------------------------------------------------------------------- | -------- |
| **Identity**        | Wrong SUCI type, stale GUTI, invalid GUAMI, borrowed RAN_UE_ID      | P0       |
| **Length/Boundary** | Max-length IE, zero-length, off-by-one                              | P0       |
| **Optional IE**     | Remove mandatory-like IEs, insert rare optional IEs                 | P0       |
| **Security**        | Wrong KSI, stale NAS SQN, mismatched algorithm, invalid MAC         | P0       |
| **Cross-field**     | Valid encoding but broken logic constraints (NSSAI vs DNN mismatch) | P1       |
| **Timing**          | Delayed response, premature response, response after timeout        | P1       |
| **Duplication**     | Send same message N times, replay old message in new context        | P1       |


### Step 3: Build Schedule Mutation Engine

Extend the task queue in `task.go` to support adversarial scheduling:


| Operator                | Description                                        |
| ----------------------- | -------------------------------------------------- |
| `Swap(M_i, M_j)`        | Swap delivery order of two messages                |
| `Drop(M_i)`             | Suppress a message silently                        |
| `Delay(M_i, dt)`        | Hold message for `dt` before delivery              |
| `Duplicate(M_i, k)`     | Send same message `k` times                        |
| `Replay(M_i, ctx_old)`  | Replay message from old context in current session |
| `InterleaveProc(A, B)`  | Interleave steps of two different procedures       |
| `AbortSCTP(step=n)`     | Kill SCTP association at step `n`                  |
| `RestartNF(nf, step=n)` | Restart a network function at step `n`             |


### Step 4: Build Spec Model Oracle

Encode 3GPP state machine rules as property checkers:

```go
type SpecOracle struct {
    LegalTransitions map[StateCombo][]StateCombo
    ForbiddenStates  []StatePredicate
    Waypoints        []WaypointPredicate
}

type Verdict int
const (
    PASS      Verdict = iota  // Behavior matches spec
    FAIL                       // Spec violation
    ANOMALY                    // Unexpected but not clearly forbidden
    UNREACHABLE                // Reached state spec says impossible
)
```

Key waypoint predicates for RL:


| Waypoint                                                      | Why Interesting             |
| ------------------------------------------------------------- | --------------------------- |
| `UE.state == REGISTERED && UE.security_ctx == NULL`           | Critical security violation |
| `exists UE_a, UE_b : UE_a.GUTI == UE_b.old_GUTI`              | Identity confusion          |
| `AMF.overload == true && emergency_registration > 0`          | Emergency under stress      |
| `handover_in_progress(UE) && deregistration_in_progress(UE)`  | Race condition              |
| `active_PDU_sessions > capacity && new_registration_arriving` | Resource exhaustion         |
| `SMF.session_released && UPF.forwarding_table_not_empty`      | State desync across NFs     |


### Step 5: Build RL Conductor

Based on BonusMaxRL + WaypointRL from the distributed systems literature:

```
RL Agent Action Space:
  - Which UE group registers (count, timing)
  - Which UE group starts PDU session
  - Which UE group does handover
  - Which mutation operators to apply
  - Whether to inject infrastructure fault
  - Which NF to stress/restart

RL Agent Reward:
  r = (1/V(s,a))           // Decaying exploration bonus (BonusMaxRL)
    + progR * I(waypoint)   // Waypoint progression bonus
    + finalR * I(target)    // Target state reached
```

The RL agent learns a **strategy** for breaking the 5G core. Over thousands of episodes, it develops intuition for which combinations of UE behaviors, timing, and faults produce spec violations.

### Step 6: Corpus Manager + Reproducibility

Unlike AMFuzz which saves random seeds for replay, StormFuzz saves **scenario descriptors**:

```yaml
scenario_id: "storm-2026-03-21-00042"
swarm_config:
  group_a: {count: 500, role: register, mutation: identity_collision}
  group_b: {count: 200, role: handover, mutation: temporal_pincer}
  group_c: {count: 100, role: attack, mutation: ghost_context}
schedule_mutations:
  - {type: delay, target_msg: auth_response, delay_ms: 500}
  - {type: restart_nf, target: smf, at_step: 3}
content_mutations:
  - {target_msg: registration_request, field: guti, mutator: stale_reuse}
verdict: UNREACHABLE
discovered_state: "UE registered without security context"
coverage_delta: {amf: +12 edges, smf: +3 edges}
```

---

## Phase 6: Sophia -- Why This Is "To Lon" (Grand)

### What Makes StormFuzz Fundamentally Different


| Dimension           | Every Existing 5G Fuzzer              | StormFuzz                                                                       |
| ------------------- | ------------------------------------- | ------------------------------------------------------------------------------- |
| **Scale**           | 1 UE, 1 session                       | 10,000+ UEs as coordinated adversary                                            |
| **Mutation**        | Packet bytes or single IE             | Content + Schedule + Infrastructure (3 dimensions)                              |
| **Intelligence**    | Random or coverage-guided             | RL with formal-model waypoints + semantic rewards                               |
| **Oracle**          | Crash detection only                  | 3GPP spec model verdicts + differential testing                                 |
| **Bug class**       | Parser errors, single-message rejects | Races, state confusion, cascade failures, identity attacks, resource exhaustion |
| **Instrumentation** | One-sided or black-box                | Dual-side white-box (emulator + core)                                           |
| **Existing infra**  | Must build from scratch               | Leverages StormSIM's FSM, NAS library, worker pools, security context           |


### The Core Insight

> **StormSIM's existing scale IS the security feature.** The swarm of 10,000+ UEs with FSM-level control is something no 5G fuzzer has. Making the swarm adversarial, making the adversary intelligent (RL), and making the verdicts spec-aware (formal model oracle) -- that combination doesn't exist anywhere.

### The AI/RL/ML Role Is Precise, Not Decorative


| AI Component             | Purpose                                                                   | Why Necessary                                                                   |
| ------------------------ | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **RL (BonusMaxRL)**      | Exploration bonus with decay -- drives agent to visit new abstract states | Random fuzzing statistically never reaches deep multi-entity states             |
| **RL (WaypointRL)**      | Semantic waypoints as intermediate rewards                                | Guides agent toward high-value scenarios (security violations, race conditions) |
| **Formal Model**         | 3GPP state machine as coverage oracle                                     | Defines what "interesting" means -- not code lines, but protocol states         |
| **Differential Testing** | Compare behavior across Open5GS / free5GC / OAI                           | Same input, different behavior = implementation bug -- no model needed          |


RL is NOT used for mutation generation (that's deterministic, IE-aware). RL is used for **strategic decision-making**: which swarm configuration, which fault injection, which timing, will drive the system into unexplored territory.

---

## Roadmap: 4 Phases

### Phase 1: Foundation (MVP -- publishable)

- Evolve `FsmFuzzer` into Message Hook Layer
- 20-30 IE-aware mutators for Registration + PDU Session
- 8-10 schedule mutators (delay, drop, duplicate, replay)
- Basic oracle: crash, timeout, unexpected reject cause
- Instrument Open5GS for edge coverage
- **Bug target**: Single-UE deep stateful bugs (like AMFuzz found 7 bugs on 3 cores)

### Phase 2: Swarm (the differentiator)

- Swarm strategy generator: population mix, role assignment, coordination patterns
- Concurrency mutators: same SUPI/GUTI, parallel procedures, competing gNBs
- All 7 attack families operational
- Multi-component coverage collection
- **Bug target**: Cross-UE bugs, race conditions, resource exhaustion

### Phase 3: Intelligence (the moat)

- RL Conductor (BonusMaxRL + WaypointRL)
- Formal spec model oracle with waypoint predicates
- Reward function with state novelty + coverage + semantic anomaly
- Automated campaign that learns over thousands of episodes
- **Bug target**: Deep, multi-entity, concurrent states that random exploration never reaches

### Phase 4: Differential + Production

- Differential oracle: Open5GS vs free5GC vs OAI
- CI/CD integration
- Scenario DSL for custom attack definition
- Corpus management with seed lifecycle
- **Bug target**: Cross-implementation inconsistencies, spec ambiguities

---

### Publishable Contribution (One Sentence)

> **StormFuzz demonstrates that population-scale adversarial emulation, guided by formal 3GPP protocol models and reinforcement learning, constitutes a fundamentally more powerful testing paradigm for 5G Core conformance and security than single-session packet mutation -- and provides the first system that realizes this paradigm by weaponizing a production-scale UE/gNB emulator with 3-dimensional mutation (content + schedule + infrastructure), spec-model oracles, and swarm-coordinated attack strategies.**

Target venues: **NDSS, CCS, USENIX Security** (security), **ISSTA, ICSE, ASE** (testing), **OOPSLA** (distributed systems verification).