<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Worker Pool Management

## Purpose
Manages worker pools for concurrent processing of 5G protocol events. Provides hierarchical pool structure with dedicated subpools for different protocol layers (5GMM, 5GSM, gNB NGAP, SCTP/NGAP transport). Auto-sizes based on CPU count and expected UE/gNB scale.

## Key Files
| File | Description |
|------|-------------|
| `pool.go` | Pool initialization, auto-sizing logic, stats retrieval |

## Subdirectories
None

## For AI Agents

### Pool Architecture

**Root Pool with Subpools:**
```
WorkerPool (root, max capacity)
├── MmWorkerPool (40% of remaining) - 5GMM events
├── SmWorkerPool (35% of remaining) - 5GSM events
├── GnbWorkerPool (remaining) - gNB NGAP handling
└── SctpNgapWorkerPool (1/6 of total) - SCTP I/O workers
```

**Pool Variables (package-level):**
```go
var (
    WorkerPool         pond.Pool  // Root pool
    MmWorkerPool       pond.Pool  // 5GMM (mobility management)
    SmWorkerPool       pond.Pool  // 5GSM (session management)
    GnbWorkerPool      pond.Pool  // gNB context handling
    SctpNgapWorkerPool pond.Pool  // SCTP read/write
)
```

### Initialization

**Call once at startup:**
```go
// Auto-detect sizes based on CPU count
pool.InitWorkerPool(ctx, 0, 0, nUEs, nGnbs)

// Or manual configuration
pool.InitWorkerPool(ctx, maxPool, nSctpWorker, nUEs, nGnbs)
```

**Auto-sizing Rules:**
- Base: 1500 workers per CPU core
- Min total: 1000, Max total: 50000 (100000 if required by UE count)
- Required workers: `(nUEs * 3) + (nGnbs * 100)`
- SCTP workers: `totalPool/6`, minimum `max(200, nGnbs*50)`
- Remaining split: 40% MM, 35% SM, rest GNB

### Usage Pattern

**Submitting tasks to pools:**
```go
// FSM events submit to appropriate pool
fsm.w.Submit(func() {
    state.slock.Lock()
    fsm.handler(state, event)
    state.slock.Unlock()
})
```

**Getting pool statistics:**
```go
stats := pool.GetPoolStats()
// Returns: total_capacity, total_running, mm_capacity, mm_running,
//          sm_capacity, sm_running, gnb_capacity, gnb_running,
//          sctp_capacity, sctp_running
```

### Concurrency Constraints

**CRITICAL: NO mutexes in hot paths.**

- Use `sync/atomic` for ID generation and counters
- Use channels for coordination between goroutines
- Worker pools handle concurrent task execution
- Each FSM state has its own `slock` mutex - acquired by pool worker

**Why this matters:**
- Hot paths process thousands of events per second
- Mutex contention destroys throughput at scale
- Worker pools provide structured concurrency without lock contention

### Pool Sizing Guidance

**For 10,000 UEs, 100 gNBs (typical):**
```
Total: ~45000 workers
SCTP: ~7500 workers (1/6)
Remaining: ~37500 workers
  - MM: ~15000 (40%)
  - SM: ~13125 (35%)
  - GNB: ~9375 (25%)
```

**Minimum viable (small tests):**
```
Total: 1000 workers
SCTP: 200 workers (minimum enforced)
Remaining: 800 workers
  - MM: 320, SM: 280, GNB: 200
```

## Dependencies

### Internal
- None (pure infrastructure)

### External
- `github.com/alitto/pond/v2` - Worker pool with subpools and context support

## Implementation Notes

**Singleton Pattern:**
- Uses `sync.Once` to ensure single initialization
- `poolInitOnce.Do(func() { ... })` guarantees one-time setup

**Subpool Relationship:**
- Subpools share capacity from root pool
- Each subpool has its own max worker limit
- Overflow from subpool uses root pool capacity

<!-- MANUAL: Add notes about specific pool implementation details here -->
