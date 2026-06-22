<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Internal

## Purpose
Core implementation of the StormSIM 5G emulator containing transport layer (SCTP/NGAP, virtual radio), UE/gNB contexts with 5GMM/5GSM state machines, shared infrastructure (worker pools, FSM framework, logging, statistics), and test scenario orchestration.

## Key Files
| File | Description |
|------|-------------|
| `core/gnbcontext/context.go` | GnbContext struct with AMF pool, UE pools, slice config |
| `core/gnbcontext/ngap_handler.go` | NGAP message handlers for downlink messages |
| `core/gnbcontext/ngap_send.go` | NGAP message construction and sending |
| `core/uecontext/ue.go` | UeContext struct, initialization, PDU session management |
| `core/uecontext/statemachine_5gmm.go` | 5GMM FSM transitions and callbacks |
| `core/uecontext/statemachine_5gsm.go` | 5GSM FSM for PDU session management |
| `core/uecontext/handle_n1mm.go` | NAS 5GMM message handlers (registration, auth) |
| `core/uecontext/handle_n1sm.go` | NAS 5GSM message handlers (PDU session) |
| `common/fsm/fsm.go` | Generic async FSM framework with worker pool integration |
| `common/pool/pool.go` | Worker pool initialization (Mm/Sm/Gnb/Sctp pools) |
| `common/logger/logger.go` | Zerolog-based logging with field support |
| `common/logger/ringbuffer.go` | Ring buffer for concurrent UE/gNB logging |
| `common/stats/procedure_stats.go` | Global procedure statistics collector |
| `transport/sctpngap/sctp.go` | SCTP connection management with read/write workers |
| `transport/rlink/link.go` | Virtual radio link (UE↔gNB channels) |
| `scenarios/groupUE.go` | UE grouping and event distribution |
| `scenarios/remote-api.go` | Remote API for OAM integration |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `core/` | UE and gNB context implementations (see `core/AGENTS.md`) |
| `core/gnbcontext/` | gNB context, NGAP handlers, AMF pool management |
| `core/uecontext/` | UE context, 5GMM/5GSM FSMs, NAS handlers, timers |
| `core/uecontext/sec/` | Security context, milenage authentication, SQN management |
| `core/uecontext/timer/` | Timer engine for NAS timers (T3510, T3540, etc.) |
| `core/managers/` | Management structures (incomplete) |
| `common/` | Shared infrastructure (see `common/AGENTS.md`) |
| `common/fsm/` | Async FSM framework for state machines |
| `common/pool/` | Worker pool management using github.com/alitto/pond/v2 |
| `common/logger/` | Ring buffer logging, delay tracking, buffered loggers |
| `common/stats/` | Procedure statistics (registration, PDU establish) |
| `common/ds/` | Data structures (generic queue, task queue) |
| `transport/` | Transport layer (see `transport/AGENTS.md`) |
| `transport/rlink/` | Virtual radio link between UE and gNB |
| `transport/sctpngap/` | SCTP/NGAP transport for N2 interface |
| `scenarios/` | UE orchestration and test scenarios (see `scenarios/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- **Concurrency**: Use `sync/atomic` for ID generation and counters. Use channels for coordination. NO mutexes in hot paths.
- **Memory Safety**: Do NOT use `sync.Pool` for SCTP message reads to avoid data corruption across worker goroutines.
- **FSM Flow**: NEVER call `SendEvent` from within an FSM callback. Use `SetNextEvent` instead to avoid recursive events.
- **Worker Pools**: All FSM events are processed via worker pools (MmWorkerPool, SmWorkerPool, GnbWorkerPool, SctpNgapWorkerPool).
- **Logging**: Use `BufferedLogger` for UE/gNB contexts to enable ring buffer log retrieval via OAM API.

### Testing Requirements
```bash
# Run all tests in internal/
go test ./internal/...

# Run specific package tests
go test ./internal/common/fsm/...
go test ./internal/core/uecontext/sec/...
```

### Common Patterns

**FSM Event Handling** (never call SendEvent from callback):
```go
// In callback - use SetNextEvent for chaining
func someCallback(state *fsm.State, event *fsm.EventData) {
    // ... process event ...
    state.SetNextEvent(nextEvent) // Queue next event, don't call SendEvent
}
```

**Worker Pool Usage**:
```go
// Events are submitted to pools
fsm.w.Submit(func() {
    state.slock.Lock()
    fsm.handler(state, event)
    state.slock.Unlock()
})
```

**Atomic ID Generation**:
```go
func (gnb *GnbContext) getRanUeId() int64 {
    return atomic.AddInt64(&gnb.ranUeIdGenerator, 1) - 1
}
```

**RLink Connection (UE↔gNB)**:
```go
conn := rlink.NewConnection(ueID, msin, gnbID, bufferSize, timeout)
conn.SendUplink(msg)    // UE → GNB
conn.SendDownlink(msg)  // GNB → UE
```

## Dependencies

### Internal
- `pkg/model` - 3GPP data models, events, states
- `pkg/config` - Configuration structures

### External
- `github.com/alitto/pond/v2` - Worker pool management
- `github.com/rs/zerolog` - Structured logging
- `github.com/ishidawataru/sctp` - SCTP transport
- `github.com/lvdund/ngap` - NGAP encoding/decoding
- `github.com/reogac/nas` - NAS message handling
- `github.com/vishvananda/netlink` - Network interface management
- `github.com/wmnsk/go-gtp/gtpv1` - GTP-U user plane

<!-- MANUAL: Add notes about specific internal implementation details here -->
