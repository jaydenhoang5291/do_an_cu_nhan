<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# fsm

## Purpose
Generic async Finite State Machine framework for StormSIM's 5GMM and 5GSM state machines. Provides transition table-based state management with worker pool integration, type-safe event payloads via generics, and automatic entry/exit event generation. Designed for high-concurrency scenarios where thousands of UEs process state transitions in parallel.

## Key Files
| File | Description |
|------|-------------|
| `fsm.go` | Core FSM implementation with transition table, callbacks, worker pool submission |
| `state.go` | State struct with mutex-protected current state, next event queue, type-safe info pointer |
| `event.go` | EventData with generic payload via unsafe.Pointer, type-safe accessors |
| `fsm_fuzz.go` | Fuzzer wrapper for stress testing FSMs with random state/event injection |

## For AI Agents

### Working In This Directory

**CRITICAL: NEVER call `SendEvent` or `SyncSendEvent` from within an FSM callback.** This causes recursive event processing and race conditions. Always use `SetNextEvent` to chain events.

```go
// WRONG - causes deadlock/race
func callback(state *fsm.State, event *fsm.EventData) {
    fsm.SendEvent(state, nextEvent) // NEVER DO THIS
}

// CORRECT - queue next event for processing after callback completes
func callback(state *fsm.State, event *fsm.EventData) {
    state.SetNextEvent(nextEvent) // Safe chaining
}
```

### Common Patterns

**Creating an FSM:**
```go
fsm := fsm.NewFsm(fsm.Options{
    Transitions: fsm.Transitions{
        // (current state, event) -> next state
        fsm.Tuple(model.StateDeregistered, model.RegistrationEvent): model.StateRegistered,
        fsm.Tuple(model.StateRegistered, model.DeregistrationEvent): model.StateDeregistered,
    },
    Callbacks: fsm.Callbacks{
        model.StateDeregistered: handleDeregistered,
        model.StateRegistered:   handleRegistered,
    },
    GenericCallback: nil, // Optional handler for non-transitional events
    NonTransitionalEvents: []model.EventType{
        model.TimeoutEvent, // Events that don't cause state change
    },
}, pool.MmWorkerPool) // Worker pool for async event processing
```

**State with Type-Safe Context:**
```go
// Create state with attached info
type UeInfo struct {
    Imsi string
    // ...
}
state := fsm.NewState(model.StateDeregistered, &UeInfo{Imsi: "12345"})

// Retrieve typed info in callback
func callback(state *fsm.State, event *fsm.EventData) {
    info := fsm.GetStateInfo[UeInfo](state)
    fmt.Println(info.Imsi)
}
```

**Creating Events with Payloads:**
```go
// Event with typed payload
authData := &AuthData{Rand: rand, Autn: autn}
event := fsm.NewEventData(model.AuthEvent, authData)

// Event without payload
event := fsm.NewEmptyEventData(model.TimeoutEvent)

// Access payload in callback
func callback(state *fsm.State, event *fsm.EventData) {
    data := fsm.GetEventData[AuthData](event)
    if data != nil {
        // use data.Rand, data.Autn
    }
}
```

**Sending Events:**
```go
// Async - returns error channel (non-blocking)
errCh := fsm.SendEvent(state, event)
// Optionally check for invalid transition errors
select {
case err := <-errCh:
    if err != nil {
        log.Error("Invalid transition:", err)
    }
default:
    // Event submitted to worker pool
}

// Sync - blocks until event processed
err := fsm.SyncSendEvent(state, event)
if err != nil {
    log.Error("Invalid transition:", err)
}
```

**Entry/Exit Events:**
During state transitions, the FSM automatically generates:
1. Current state callback with original event
2. Current state callback with `model.ExitEvent` (cloned from original)
3. State change
4. New state callback with `model.EntryEvent` (cloned from original)

```go
func handleRegistered(state *fsm.State, event *fsm.EventData) {
    switch event.Type() {
    case model.EntryEvent:
        // State just entered - start timers, send initial messages
    case model.ExitEvent:
        // State about to exit - cleanup resources
    case model.DeregistrationEvent:
        // Normal event handling
    }
}
```

**Fuzzing for Stress Testing:**
```go
fuzzer := fsm.NewFsmFuzzer(baseFSM, &fsm.FuzzerOptions{
    FuzzMode:       true,
    PossibleStates: []model.StateType{model.StateDeregistered, model.StateRegistered},
    PossibleEvents: []model.EventType{model.RegistrationEvent, model.TimeoutEvent},
}, ctx)

// Random event injection at interval
go fuzzer.AutoRandomEvent(state, event, 100*time.Millisecond)

// Fuzz mode overrides SendEvent to inject random states/events
fuzzer.SendEvent(state, event)
```

### Internal Details

**Locking Strategy:**
- `State.slock` - Serializes event handling for a single state (locked during entire callback chain)
- `State.mutex` - Protects read/write of current state value
- Worker pool provides parallelism across different UE states

**Event Processing Flow:**
1. `SendEvent`/`SyncSendEvent` submits to worker pool (or runs synchronously)
2. `handleEvent` acquires `slock`, checks if transitional or non-transitional
3. `transit` looks up transition, executes callbacks, handles entry/exit
4. `processNextEvent` processes any queued `next` event
5. `slock` released

## Dependencies

### Internal
- `pkg/model` - `StateType`, `EventType` definitions (`EntryEvent`, `ExitEvent`, etc.)

### External
- `github.com/alitto/pond/v2` - Worker pool (`pond.Pool` interface)

<!-- MANUAL: Add notes about FSM implementation details, edge cases, or debugging tips here -->
