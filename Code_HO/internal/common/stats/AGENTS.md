<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# stats

## Purpose
Global procedure statistics collection for tracking 5G NAS procedures (registration, deregistration, PDU session establishment). Uses lock-free atomic counters for running/completed/failed states with time-series historical snapshots for monitoring and OAM API queries. Designed for extreme concurrency with zero mutex contention in hot paths.

## Key Files
| File | Description |
|------|-------------|
| `procedure_stats.go` | `StatsCollector` with atomic counters, procedure types, JSON snapshots |
| `history.go` | `StatsHistory` with cron-based time-series snapshots (max 3600) |

## For AI Agents

### Working In This Directory
- **Use Global Singletons**: Access via `stats.GlobalStats` and `stats.GlobalHistory` - no need to create instances
- **Atomic Operations**: All counters use `atomic.Int64` - safe for concurrent access without mutexes
- **History Auto-pruning**: Snapshots automatically trimmed to `MaxSnapshots` (3600) via sliding window

### Common Patterns

**Tracking Procedure Lifecycle**:
```go
import "github.com/stormsim/internal/common/stats"

// Start tracking a procedure
stats.GlobalStats.StartProcedure(stats.ProcRegistration)

// On successful completion
stats.GlobalStats.CompleteProcedure(stats.ProcRegistration)

// On failure
stats.GlobalStats.FailProcedure(stats.ProcRegistration)
```

**Getting Current Stats Snapshot**:
```go
snapshot := stats.GlobalStats.GetSnapshot()
// Returns: map[ProcedureType]StatSnapshot
// StatSnapshot has: Running, Completed, Failed (all int64)

// Example: check registration stats
regStats := snapshot[stats.ProcRegistration]
fmt.Printf("Running: %d, Completed: %d, Failed: %d\n",
    regStats.Running, regStats.Completed, regStats.Failed)
```

**Starting Historical Collection**:
```go
// Start cron job - takes snapshot every second
stats.GlobalHistory.StartCron(time.Second)

// Stop cron job
stats.GlobalHistory.StopCron()
```

**Querying Historical Data**:
```go
// Get all snapshots (up to 3600)
all := stats.GlobalHistory.GetAllSnapshots()

// Get snapshots since a specific time
recent := stats.GlobalHistory.GetSnapshotsSince(time.Now().Add(-5 * time.Minute))

// Get count
count := stats.GlobalHistory.GetSnapshotCount()
```

**Adding Custom Procedure Types**:
```go
// Register a new procedure type at runtime
stats.GlobalStats.AddProcedureType(stats.ProcedureType("custom_procedure"))
stats.GlobalStats.StartProcedure(stats.ProcedureType("custom_procedure"))
```

### Procedure Types
```go
const (
    ProcRegistration   ProcedureType = "registration"   // 5GMM registration
    ProcDeregistration ProcedureType = "deregistration" // 5GMM deregistration
    ProcPduEstablish   ProcedureType = "pdu_establish"  // 5GSM PDU session
)
```

### Data Structures

**ProcedureStat** - Internal atomic counters:
```go
type ProcedureStat struct {
    Running   atomic.Int64  // Currently in-progress
    Completed atomic.Int64  // Successfully finished
    Failed    atomic.Int64  // Failed/errored
}
```

**StatSnapshot** - JSON-serializable snapshot:
```go
type StatSnapshot struct {
    Running   int64 `json:"running"`
    Completed int64 `json:"completed"`
    Failed    int64 `json:"failed"`
}
```

**HistoricalSnapshot** - Timestamped stats:
```go
type HistoricalSnapshot struct {
    Timestamp time.Time                      `json:"timestamp"`
    Stats     map[ProcedureType]StatSnapshot `json:"stats"`
}
```

## Dependencies

### Internal
- None (standalone package)

### External
- `sync` - RWMutex for history, sync/atomic for counters
- `time` - Ticker for cron snapshots, timestamps

## Concurrency Notes
- `ProcedureStat`: Uses `atomic.Int64` for all counters - lock-free in hot paths
- `StatsCollector`: RWMutex only on `GetSnapshot()` and `AddProcedureType()` - not in procedure tracking
- `StatsHistory`: RWMutex protected, runs cron in separate goroutine
- **Safe for 10,000+ concurrent UEs** calling Start/Complete/Fail without contention
