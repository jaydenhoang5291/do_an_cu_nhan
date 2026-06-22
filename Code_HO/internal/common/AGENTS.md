<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Common Infrastructure

## Purpose
Shared infrastructure for StormSIM including async FSM framework, worker pool management, high-performance ring buffer logging with delay tracking, and global procedure statistics. All modules are designed for extreme concurrency (10,000+ UEs).

## Key Files
| File | Description |
|------|-------------|
| `fsm/fsm.go` | Generic async FSM with worker pool integration, transition table, callbacks |
| `fsm/event.go` | Generic EventData with type-safe payload via unsafe.Pointer |
| `fsm/state.go` | State management with mutex locking, next event queue |
| `fsm/fsm_fuzz.go` | Fuzzer for FSM stress testing with random state/event injection |
| `pool/pool.go` | Worker pool initialization (Mm/Sm/Gnb/SctpNgap pools) with auto-sizing |
| `logger/logger.go` | Zerolog-based Logger with colored console output |
| `logger/ringbuffer.go` | Generic thread-safe ring buffer with generation tracking for safe updates |
| `logger/buffered_logger.go` | BufferedLogger combining Logger + RingBuffer + DelayTracker |
| `logger/delay_types.go` | DelayTracker for request-response timing, procedure duration stats |
| `logger/nas_types.go` | NAS message type names and request/response pair mappings |
| `logger/ngap_types.go` | NGAP message type names via type assertion |
| `stats/procedure_stats.go` | Global atomic counters for running/completed/failed procedures |
| `stats/history.go` | Time-series snapshots of procedure stats (3600 max) |
| `ds/queue.go` | Generic thread-safe FIFO queue |
| `ds/task.go` | Tasks wrapper with event type validation |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `fsm/` | Async state machine framework for 5GMM/5GSM FSMs |
| `pool/` | Worker pool management using github.com/alitto/pond/v2 |
| `logger/` | Logging, ring buffers, delay tracking, NAS/NGAP type names |
| `stats/` | Global procedure statistics collector with history |
| `ds/` | Generic data structures (Queue, Tasks) |

## For AI Agents

### FSM Framework

**Critical Rule: NEVER call `SendEvent` from within an FSM callback.** Use `SetNextEvent` instead to avoid recursive events and race conditions.

```go
// WRONG - causes race condition
func callback(state *fsm.State, event *fsm.EventData) {
    fsm.SendEvent(state, nextEvent) // NEVER DO THIS
}

// CORRECT - queue next event
func callback(state *fsm.State, event *fsm.EventData) {
    state.SetNextEvent(nextEvent) // Safe chaining
}
```

**Creating an FSM:**
```go
fsm := fsm.NewFsm(fsm.Options{
    Transitions: fsm.Transitions{
        fsm.Tuple(model.StateDeregistered, model.RegistrationEvent): model.StateRegistered,
        // ... more transitions
    },
    Callbacks: fsm.Callbacks{
        model.StateDeregistered: someCallback,
        // ... callbacks for each state
    },
    NonTransitionalEvents: []model.EventType{
        model.TimeoutEvent, // handled without state change
    },
}, pool.MmWorkerPool) // Submit to worker pool
```

**Sending Events:**
```go
// Async - returns error channel
errCh := fsm.SendEvent(state, event)

// Sync - blocks until complete
err := fsm.SyncSendEvent(state, event)
```

### Worker Pools

**Pool Types (defined in `pool/pool.go`):**
- `pool.WorkerPool` - Root pool with max capacity
- `pool.MmWorkerPool` - 5GMM (mobility management) events
- `pool.SmWorkerPool` - 5GSM (session management) events
- `pool.GnbWorkerPool` - gNB NGAP handling
- `pool.SctpNgapWorkerPool` - SCTP read/write workers

**Initialization:**
```go
pool.InitWorkerPool(ctx, maxPool, nSctpWorker, nUEs, nGnbs)
// Auto-detects sizes if maxPool <= 0 based on CPU count and UE/gNB count
```

**Pool Stats:**
```go
stats := pool.GetPoolStats()
// Returns: total_capacity, total_running, mm_capacity, mm_running, etc.
```

### Ring Buffer Logging

**RingBuffer with Generation Tracking:**
```go
rb := logger.NewRingBuffer[LogEntry](1000)
idx, gen := rb.Push(entry)        // Returns index and generation
rb.Update(idx, gen, updatedEntry) // Safe update (fails if slot overwritten)
entries := rb.GetAll()             // Chronological order
recent := rb.GetLast(10)           // Most recent first
```

**BufferedLogger for UE/gNB Contexts:**
```go
bl := logger.NewBufferedLogger(
    1000,              // buffer size
    "UE",              // entity type
    "imsi-12345",      // entity ID
    nil,               // fields (optional)
    func() string { return state.CurrentState().String() }, // state getter
)

// Logs to both console and ring buffer
bl.Info("Registration started")
bl.LogSend("nas", "RegistrationRequest")    // Records send timestamp
bl.LogReceive("nas", "AuthenticationRequest") // Calculates delay

// Retrieve logs via OAM API
logs := bl.GetLogs()
delays := bl.GetDelayLogs(50)
stats := bl.GetDelayStats()
```

### Delay Tracking

**Tracked Procedures:**
- `registration` - RegistrationRequest → RegistrationComplete
- `deregistration` - DeregistrationRequestFromUE → DeregistrationAcceptFromUE
- `pdu_establish` - PduSessionEstablishmentRequest → PduSessionEstablishmentAccept
- `pdu_release` - PduSessionReleaseRequest → PduSessionReleaseComplete

**NAS Message Pairing:**
Defined in `logger/nas_types.go`:
- `NasResponseToRequests` - Maps response → valid preceding requests
- `NasRequestToResponses` - Maps request → expected responses

### Statistics

**Global Procedure Stats:**
```go
stats.GlobalStats.StartProcedure(stats.ProcRegistration)
// ... procedure runs ...
stats.GlobalStats.CompleteProcedure(stats.ProcRegistration)
// or
stats.GlobalStats.FailProcedure(stats.ProcRegistration)

snapshot := stats.GlobalStats.GetSnapshot()
// Returns: {running, completed, failed} per procedure type
```

**Historical Snapshots:**
```go
stats.GlobalHistory.StartCron(time.Second) // Take snapshot every second
snapshots := stats.GlobalHistory.GetSnapshotsSince(someTime)
stats.GlobalHistory.StopCron()
```

### Data Structures

**Generic Queue:**
```go
q := &ds.Queue[int]{}
q.Enqueue(1)
val, ok := q.Dequeue()
val, ok := q.Peek()
isEmpty := q.IsEmpty()
len := q.Len()
```

**Tasks with Validation:**
```go
tasks := ds.NewTasks[SomeType]([]model.EventType{
    model.RegistrationEvent,
    model.AuthEvent,
})
tasks.AssignTask(task)
task, ok := tasks.PopTask()
valid := tasks.CheckValidTask(&eventType)
```

## Dependencies

### Internal
- `pkg/model` - StateType, EventType definitions

### External
- `github.com/alitto/pond/v2` - Worker pool with subpools
- `github.com/rs/zerolog` - Structured logging
- `github.com/reogac/nas` - NAS message type constants
- `github.com/lvdund/ngap/ies` - NGAP message types for naming

## Concurrency Notes

- **FSM**: Each State has `slock` mutex for event handling. Events submitted to worker pools.
- **RingBuffer**: RWMutex protected. Generation tracking prevents ABA problem on updates.
- **Stats**: Uses `atomic.Int64` for counters. No mutex in hot paths.
- **Queue**: RWMutex protected. Simple FIFO semantics.

<!-- MANUAL: Add notes about specific common module implementation details here -->
