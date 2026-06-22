## The Intelligence Spectrum: From Random to RL with Formal-Model Waypoints

Think of it as three generations of "brain" behind the fuzzer, each dramatically smarter than the last.

---

### Generation 1: Random

This is what most simple fuzzers do. Flip random bits. Pick a random field. Insert a random value.

In a 5G context: take a Registration Request, randomly change some bytes, send it. Maybe something crashes. Repeat a billion times.

**Problem**: The 5G Core has an astronomically large state space. Random exploration is like a drunk person wandering a dark city hoping to accidentally walk into the bank vault. You'll cover the streets near the bar (parser errors, malformed PDU rejects), but you'll never reach the deep neighborhoods (race conditions during handover, state confusion after re-authentication).

---

### Generation 2: Coverage-Guided (e.g., AFL, AFLNet)

Now the fuzzer has *eyes*. It instruments the code and measures which code branches were executed. If a mutated input triggers a **new branch** that was never seen before, it's saved as an interesting seed and mutated further.

In a 5G context: you send a Registration Request with a weird GUTI. The AMF hits a code path it never hit before (say, a GUTI-reallocation branch). The fuzzer says "interesting!" and focuses future mutations around that GUTI value.

**Problem**: Code coverage is *syntactic* — it tells you which `if` statements you hit, not which *protocol state* you reached. You might cover 90% of the AMF's code lines and still never reach the state where:

> "UE-A is mid-handover, UE-B is re-authenticating with UE-A's old GUTI, and the SMF is restarting"

That state involves a *combination* of protocol conditions across multiple entities — code coverage is blind to it.

---

### Generation 3: RL with Formal-Model Waypoints (the StormFuzz idea)

This is two innovations stacked together. Let me separate them.

#### Part A: The Formal Model as a Map

You take the 3GPP specification's state machines (Registration, Authentication, Security Mode, PDU Session, Handover, etc.) and encode them as a **formal model** — think of it as a precise mathematical map of every legal and illegal state the 5G Core can be in.

For example, a simplified Registration model might have states like:

```
DEREGISTERED → REGISTRATION_INITIATED → AUTHENTICATION_ONGOING →
SECURITY_MODE_PENDING → REGISTERED → CM_CONNECTED
```

But the real model also captures **cross-entity relationships**:

```
(UE.state = SECURITY_MODE_PENDING) ∧
(AMF.ue_context[42].security_ctx = ESTABLISHED) ∧
(SMF.pdu_sessions = EMPTY)
```

This formal model defines a **coverage space over abstract protocol states**, not code lines. The fuzzer now tracks: "which *specification states* have I driven the system into?" This is fundamentally more meaningful than "which `if` branch did I hit."

A state the model says is **unreachable** but the implementation **actually reaches** = you found a spec violation.

A state the model says is **reachable** but the implementation **never enters** = you found dead behavior or a missing feature.

#### Part B: The RL Agent as a Strategic Commander

Now, instead of random mutation or even greedy coverage-chasing, you have a **reinforcement learning agent** that *learns a strategy* for driving the system into interesting states.

The RL agent controls high-level decisions:

- "Which UEs should register now?"
- "Should I inject a fault on the SCTP link?"
- "Should I delay this Authentication Response by 500ms?"
- "Should I trigger a handover for UE group B while group A is mid-registration?"

The **reward function** has two components:

**1. Exploration bonus (decaying):** Every time the RL agent drives the system into a formal-model state that has **never been visited**, it gets a reward. But the reward **decays** with repeated visits — so it can't farm easy states. It must keep pushing deeper into unexplored territory. (This idea comes directly from BonusMaxRL, OOPSLA 2024.)

**2. Semantic waypoints:** You define predicates that represent **high-value situations** the RL agent should try to reach. These are the "waypoints" — like checkpoints in a game. Examples:


| Waypoint Predicate                                                      | Why It's Interesting                                                             |
| ----------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `∃ UE_a, UE_b : UE_a.GUTI == UE_b.old_GUTI`                             | Tests identity confusion under GUTI reuse                                        |
| `AMF.overload == true ∧ emergency_registration_count > 0`               | Tests emergency call handling under overload                                     |
| `UE.state == REGISTERED ∧ UE.security_ctx == NULL`                      | Tests if a UE can reach registered state without security — a critical violation |
| `count(active_PDU_sessions) > AMF.capacity ∧ new_registration_arriving` | Tests resource exhaustion behavior                                               |
| `handover_in_progress(UE_a) ∧ deregistration_in_progress(UE_a)`         | Tests race between handover and deregistration                                   |


When the RL agent reaches a waypoint, it gets a **bonus reward**, and then the system encourages **further exploration around that state** — like saying "you found the bank vault door, now try to open it."

#### How It Works Together

```
RL Agent
   │
   │  "I'll register 2000 UEs, then restart the SMF,
   │   then trigger handover for 500 of them"
   │
   ▼
StormSIM Swarm ──── executes strategy ────► 5G Core
                                                │
   ┌────────────────────────────────────────────┘
   │
   ▼
Formal Model Oracle:  "System entered state S_1742.
                       This state was NEVER visited before.
                       Reward +1.0 (exploration bonus).
                       
                       Also, this state matches waypoint W_7:
                       'UE registered without security context.'
                       Reward +5.0 (waypoint bonus).
                       
                       VERDICT: SPEC VIOLATION DETECTED."
   │
   ▼
RL Agent updates its policy:
   "Restarting SMF during registration is productive.
    I'll try variations: restart during authentication,
    restart during security mode, restart during PDU setup..."
```

Over thousands of episodes, the RL agent **learns a strategy** for breaking the 5G Core. It doesn't randomly stumble into bugs — it develops an *intuition* for which combinations of UE behaviors, timing, and faults produce interesting results.

---

### The Key Difference, in One Analogy


| Approach                        | Analogy                                                                                                                                                             |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Random**                      | Throwing darts blindfolded at a wall, hoping to hit a bullseye                                                                                                      |
| **Coverage-guided**             | You can see the wall and aim for areas you haven't hit yet, but you can only see the surface                                                                        |
| **RL + formal-model waypoints** | You have an X-ray of the wall showing hidden rooms behind it, a GPS that rewards you for finding new rooms, and you *learn* which throwing angles open secret doors |


The formal model gives the fuzzer **semantic understanding** of what states matter. The RL agent gives it **strategic planning** to reach those states efficiently. Together, they turn a brute-force tool into an *intelligent adversary* that systematically maps and exploits the 5G Core's state space — especially the deep, multi-entity, concurrent states that random and coverage-guided approaches will statistically never reach.