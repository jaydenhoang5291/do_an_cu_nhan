<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-31 -->

# Managers

## Purpose
Handover management system coordinating inter-gNB handover execution across ground and NTN (Non-Terrestrial Network / satellite) gNBs. Provides priority-based scheduling, NTN satellite visibility coordination, aggregate handover statistics tracking, and an HTTP-based OAM API for monitoring and control.

## Key Files

| File | Description |
|------|-------------|
| `manager.go` | `HandoverManager` — central orchestrator: gNB registration, event loop, scheduler, handover execution (Xn/N2), callback management, network condition routing |
| `gnb_group.go` | `GnbGroup` — groups gNBs by type (Ground/NTN) with shared `NetworkCondition` (packet loss, latency, jitter) |
| `handover_queue.go` | `HandoverQueue` — priority heap-based queue for scheduled handovers, with per-UE cancellation and lazy-deletion |
| `handover_tracker.go` | `ManagerHandoverTracker` — aggregate statistics: success/fail rates, duration min/max/avg, Xn vs N2 counts, failure reason breakdown |
| `ntn_coordinator.go` | `NTNCoordinator` — satellite visibility window management, ground↔satellite handover feasibility checks, handover window prediction |
| `oam.go` | `HandoverManagerOAM` — HTTP REST endpoints for stats, queue inspection, NTN visibility, group management, best-target queries |
| `errors.go` | Sentinel errors: `ErrEventChannelFull`, `ErrGnbNotFound`, `ErrInvalidHandoverType`, `ErrHandoverNotPossible`, `ErrQueueEmpty` |
| `context.go` | Package stub (empty, retained for backward compatibility) |

## Subdirectories

None.

## For AI Agents

### Architecture Overview

```
HandoverManager
├── gnbs (sync.Map)              ← registered gNB references
├── gnbGroups (map + RWMutex)    ← Ground/NTN group management
├── handoverQueue                ← priority-sorted scheduled handovers
├── ntnCoordinator               ← satellite visibility windows
├── tracker                      ← aggregate stats
├── eventChan (buffered: 1000)   ← immediate handover events
├── runEventLoop() goroutine     ← pulls from eventChan → GnbWorkerPool
└── runScheduler() goroutine     ← polls queue every 10ms for due handovers
```

### Working In This Directory

**Concurrency Model:**
- `gnbs` uses `sync.Map` for lock-free gNB lookups
- `gnbGroups` uses `sync.RWMutex` (not hot path — group changes are rare)
- `handoverQueue` uses internal `sync.Mutex` for heap operations
- `ntnCoordinator` uses `sync.RWMutex` for visibility data
- Atomic counters (`sync/atomic`) for total/success/failed handover counts
- Handover execution is dispatched via `pool.GnbWorkerPool.Submit()`

**Event Flow:**
1. `ScheduleHandover()` → queue → `runScheduler()` polls → `eventChan`
2. `TriggerImmediateHandover()` → `eventChan` directly
3. `runEventLoop()` reads `eventChan` → submits to `GnbWorkerPool`
4. `executeHandover()` validates gNBs, checks NTN visibility, calls `gnbcontext.TriggerXnHandover` or `TriggerNgapHandover`
5. `OnHandoverComplete()` updates counters, fires callbacks, records tracker stats

### Key Data Structures

```go
// Handover types
HandoverTypeXn = 1   // Xn interface handover
HandoverTypeN2 = 2   // N2/NGAP interface handover

// Group types
GroupTypeGround = 0   // Terrestrial gNB
GroupTypeNTN    = 1   // Satellite gNB

// Network simulation parameters applied per-group
type NetworkCondition struct {
    PacketLoss float64   // 0.0 - 1.0
    LatencyMs  int
    JitterMs   int
}
```

### OAM REST API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GetStats` | GET | Total/success/failed counts, success rate, queue size |
| `GetTrackerSummary` | GET | Detailed stats with duration metrics and failure breakdown |
| `GetTrackerResults` | GET | Individual handover results (currently returns nil — aggregate only) |
| `GetQueue` | GET | Pending scheduled handovers |
| `GetSatellites` | GET | All registered satellite gNBs |
| `GetGroundGnbs` | GET | All registered ground gNBs |
| `GetVisibility` | GET | Satellite visibility: current window, next window, visible ground gNBs |
| `GetGroups` | GET | All gNB groups |
| `GetGroupByName` | GET | Single group details with gNB IDs and network conditions |
| `GetBestTarget` | GET | Best handover target for a given source gNB/UE |
| `PredictHandoverWindow` | GET | Next handover window between source and target |

### Known Issues

- **`context.go` is empty**: The old `Manger`/`RlinkGroup` stubs were replaced by the `HandoverManager` system but the file was not removed.
- **`oam.go` `GetBestTarget`**: The `prUeIdStr` query parameter is parsed but hardcoded to `0` — the UE ID is not actually used for target selection.
- **Scheduler polling**: 10ms ticker in `runScheduler()` may be excessive under low load; consider adaptive scheduling.
- **`sync.Map` for callbacks**: Callbacks are stored per-UE but never cleaned up on UE deregistration.

## Dependencies

### Internal
- `internal/common/logger` — Ring buffer logger
- `internal/common/pool` — `GnbWorkerPool` for dispatching handover execution
- `internal/core/gnbcontext` — `GnbContext`, `TriggerXnHandover`, `TriggerNgapHandover`, `IsHandoverSuccess`

### External
- `container/heap` — Priority queue implementation
- `net/http` — OAM REST API
- `encoding/json` — JSON serialization for OAM responses

<!-- MANUAL: Add notes about managers implementation details here -->
