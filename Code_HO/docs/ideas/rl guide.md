This paper proposes a **Reinforcement Learning-based testing framework** for distributed systems, built around two reward augmentation algorithms: **BonusMaxRL** and **WaypointRL**. Here is a deep breakdown of every step, its target, and its inputs/outputs.

---

## System Overview

The paper models a distributed system under test (e.g., RedisRaft, Etcd, RSL) as a **Markov Decision Process (MDP)** . The RL agent controls the *network* — message delivery order, crashes, partitions — to drive the system into buggy states . The core problem is **reward sparsity**: bugs are rare, so the agent gets almost no signal without reward augmentation .

---

## Step 0 — MDP Modeling of the Distributed System

**Target:** Convert the distributed system into an RL-compatible environment.


| Component                  | Definition                                                                                                       |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **State \mathcal{S}**      | Multi-set of process "colors" (abstract local states stripped of process IDs) + current network partition config |
| **Action \mathcal{A}**     | A new partition configuration — which processes can talk to each other                                           |
| **Transition \mathcal{T}** | Deliver messages allowed by the chosen partition; new messages change process colors                             |
| **Reward \mathcal{R}**     | Defined by BonusMaxRL / WaypointRL (not by bug detection alone)                                                  |


**Input:** Live distributed system processes
**Output:** An MDP (\mathcal{S}, \mathcal{A}, \mathcal{T}, \mathcal{R}) ready for RL 

**Key modeling guidelines applied** :

- Fix time intervals between steps to reduce non-determinism from clocks
- Use color abstraction that includes *just enough* state (e.g., role: Leader/Follower/Candidate in Raft) — too little → unpredictable transitions; too much → state space explosion
- Strip process identities from colors to exploit symmetry and avoid counting duplicate states

---

## Step 1 — Generic RL Loop (Algorithm 1)

**Target:** Provide the outer training scaffold used by both BonusMaxRL and WaypointRL.

**Input:**

- K: number of episodes
- H: horizon (steps per episode)
- E: MDP environment
- A: RL agent (either BonusMaxRL or WaypointRL)

**Output:** A learned policy \pi: \mathcal{S} \rightarrow \Delta(\mathcal{A})

**Steps per episode** :

1. `reset(E)` — restart the system to initial state s_0
2. `newEpisode(A)` — initialize agent data structures for this episode
3. For each step h = 1 \ldots H:
  - `pick(A, state, actions(E, state))` — agent selects action a
  - `step(E, state, a)` → returns (s', r)
  - `recordStep(A, s, a, s', r)` — append to trace
4. `processEpisode(A)` — **backwards sweep** to update Q-values using the full trace

The **backwards update** (step 4) is critical: it backpropagates rewards from the end of the episode all the way to the start *in a single sweep*, rather than waiting many episodes for signals to propagate .

---

## Step 2 — BonusMaxRL (Algorithm 2)

**Target:** Maximize *coverage of unique states* using a decaying exploration bonus — no external reward signal needed.

### 2a. Visit Tracking

**Input:** Every transition (s, a, s') observed
**Output:** Visit count table V(s, a) — incremented at each visit 

### 2b. Exploration Bonus Computation

For each transition (s, a, s'), the reward is:

 r(s, a, s') = \frac{1}{V(s,a)} 

- First visit → reward = 1 (maximum)
- k-th visit → reward = \frac{1}{k} (decays to 0)

This mimics **coverage-guided fuzzing**: prioritize new states, but let reward decay prevent getting stuck locally.

### 2c. Q-Value Update — The Key Innovation

Standard Q-learning uses *addition*:
 Q(s,a) = (1-\alpha) \cdot Q(s,a) + \alpha \cdot (r + \gamma \cdot \max_{a'} Q(s', a')) 

BonusMaxRL replaces addition with **max**:
 Q(s,a) = (1-\alpha) \cdot Q(s,a) + \alpha \cdot \max\left(r,\ \gamma \cdot \max_{a'} Q(s', a')\right) 

**Input:** Bonus reward r, next-state Q-values
**Output:** Updated Q-table

**Why max instead of sum?**  The max rule makes the Q-value an estimate of the *best (least-visited) reachable state* from (s, a), ignoring intermediate visit counts. This focuses the agent on paths toward new states, making it behave like a greedy frontier explorer rather than balancing all rewards along a trajectory.

### 2d. Action Selection

\epsilon-greedy: pick \arg\max_a Q(s, a) with probability 1-\epsilon; random action with probability \epsilon .

**Input:** Current state s, available actions, \epsilon
**Output:** Chosen action a

---

## Step 3 — Predicate / Waypoint Specification (Pre-processing for WaypointRL)

**Target:** Encode developer knowledge about "interesting" protocol states as a sequence of logical predicates.

**Input:** Protocol documentation + domain expert insight
**Output:** Ordered predicate sequence pred_1, pred_2, \ldots, pred_n where :

- pred_1 = \top (always true — initial state)
- pred_n = **target predicate** (e.g., "a consensus entry has been committed")
- pred_2 \ldots pred_{n-1} = intermediate milestones (e.g., "a leader has been elected")

**Example for Raft:** `[always_true → leader_elected → entry_committed]`

The paper notes that even *partial, generic hints* (agnostic to implementation details) are effective .

---

## Step 4 — WaypointRL (Algorithms 3 & 4)

**Target:** Guide exploration toward the target predicate pred_n by maintaining a **separate Q-table per predicate** and rewarding progression through the waypoint sequence.

### 4a. Initialization

**Input:** Predicate list, \alpha, \gamma, \epsilon, boolean `oneTime`
**Output:** One Q-table Q_p per predicate p 

### 4b. Active Predicate Tracking (per step)

At each timestep, identify the **active predicate** p = highest-indexed predicate that is currently true in state s .

**Input:** Current state s, predicate list
**Output:** Active predicate index p

### 4c. Action Selection

Use \epsilon-greedy on Q_p — the Q-table for the *active* predicate.

**Input:** Q_p, state s, \epsilon
**Output:** Action a

### 4d. Trace Recording (`recordStep`)

Store (s, a, s', p, p') — capturing which predicate was active before and after the transition .

**Input:** Transition (s, a, s')
**Output:** Trace entry with predicate pair (p, p')

### 4e. Episode Processing (`processEpisode`) — Backwards Sweep

For each transition (s, a, s', p, p') in reverse order :


| Case                                            | Reward Added                                 | Rationale                                              |
| ----------------------------------------------- | -------------------------------------------- | ------------------------------------------------------ |
| p = p' (predicate unchanged)                    | BonusMaxRL bonus \frac{1}{V(s,a)} only       | Pure exploration within this waypoint zone             |
| p \neq p', p' > p (progressed to next waypoint) | `progR = 2`                                  | Immediate reward for moving to a higher predicate      |
| Episode reached pred_n (target hit)             | `finalR = 2` (discounted by steps to target) | Reward the *entire path* that eventually led to target |


**Input:** Full episode trace with predicate annotations
**Output:** Updated Q-tables for all predicates involved

### 4f. `oneTime` Flag

If `oneTime = True`, once pred_n is satisfied, *all subsequent states* in the episode are treated as part of the target space . This handles predicates about *events* (e.g., "a crash occurred") rather than persistent state properties.

---

## Step 5 — Bug Detection

**Target:** Identify protocol violations during exploration.

**Input:** System state s' after each `step`
**Output:** Bug report if a safety/liveness violation is detected (e.g., two leaders elected in same term, committed entries lost after crash recovery) 

The RL agent is not rewarded for bugs — bug detection is a **passive observer** layered on top of the exploration. The augmented rewards (BonusMaxRL + WaypointRL) simply drive the agent to visit states where bugs are more likely to manifest .

---

## How the Two Algorithms Combine

BonusMaxRL and WaypointRL are **composited**, not alternatives :

1. **Within a waypoint zone** (predicate unchanged): BonusMaxRL's \frac{1}{V} bonus drives broad local coverage
2. **Between waypoint zones** (predicate changes): WaypointRL's `progR`/`finalR` rewards drive directed progress toward the target
3. The combined effect: the agent *reliably reaches deep, semantically interesting states* in new episodes **without caching execution prefixes**, because the Q-tables encode a reusable navigation policy to each waypoint

