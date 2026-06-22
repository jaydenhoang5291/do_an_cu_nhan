<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Scenarios

## Purpose
UE orchestration and test scenario management for StormSIM. Provides grouping mechanisms for multi-UE tests, event distribution, handover tracking, and remote API integration with the OAM backend. Supports single-UE testing, multi-group load testing, CSV-driven handover scenarios, and fuzz/replay testing modes.

## Key Files
| File | Description |
|------|-------------|
| `groupUE.go` | `UeGroup` struct for managing groups of UEs with event distribution, MSIN auto-increment, and round-robin gNB selection |
| `handover_tracker.go` | `HandoverTracker` and `HandoverResult` for monitoring Xn/N2 handover outcomes with timing statistics |
| `remote-api.go` | `RemoteApi` singleton for OAM integration - UE/gNB registry, worker stats, delay statistics aggregation |
| `test-single-ue.go` | Single UE test entry point with fuzz testing and replay support |
| `test-with-custom-scenarios.go` | Multi-UE/multi-group scenario execution with per-group gNB assignment |
| `test-csv-handover.go` | CSV-driven handover scenario runner with detailed result tracking |

## Subdirectories
None - this is a leaf package.

## Core Types

### UeGroup (`groupUE.go`)
Manages a group of UEs with shared event distribution:
```go
type UeGroup struct {
    id           int
    nUes         int
    remoteUEs    map[int]*UeRemote        // UE ID -> remote reference
    gnbs         func() (*gnbcontext.GnbContext, bool)  // Round-robin gNB selector
    curGnb       *gnbcontext.GnbContext
    defaultUECfg config.UeConfig
    GroupEvents  chan uecontext.EventUeData  // Event distribution channel
    Delay        uint16                       // Delay between UE operations (ms)
    wg           *sync.WaitGroup
    ctx          context.Context
}
```

### UeRemote (`groupUE.go`)
Lightweight reference to a UE within a group:
```go
type UeRemote struct {
    id        int
    msin      string
    fsm_state *fsm.State
    tasks     *ds.Tasks[*uecontext.EventUeData]  // Task queue for this UE
}
```

### HandoverTracker (`handover_tracker.go`)
Thread-safe handover outcome tracking:
```go
type HandoverTracker struct {
    Results      []HandoverResult
    totalCount   int
    successCount int
    failCount    int
}

type HandoverResult struct {
    Step          int
    FromGnb       string
    ToGnb         string
    HandoverType  int  // 1=Xn, 2=N2
    Success       bool
    StartTime     time.Time
    EndTime       time.Time
    Duration      time.Duration
    FailureReason string
}
```

### RemoteApi (`remote-api.go`)
Singleton registry for OAM integration:
```go
var oamApi = &RemoteApi{}

type RemoteApi struct {
    UEs  sync.Map  // msin string -> *uecontext.UeContext
    Gnbs sync.Map  // gnbId string -> *gnbcontext.GnbContext
}
```

## Test Scenario Functions

### TestSingleUE (`test-single-ue.go`)
Entry point for single UE testing:
```go
func TestSingleUE(cfg *config.Config, isReplay bool, replayFile string)
```
- Supports fuzz testing mode (`cfg.Testing.EnableFuzz`)
- Supports replay mode from saved state file
- Sends custom events from `cfg.Scenarios[0].UeEvents`

### TestScenarios (`test-with-custom-scenarios.go`)
Entry point for multi-group load testing:
```go
func TestScenarios(cfg *config.Config)
```
- Creates `UeGroup` per scenario in `cfg.Scenarios`
- Each group can have different gNB assignments
- Events distributed via `groupEvents` channels

### TestCSVHandover (`test-csv-handover.go`)
Entry point for CSV-driven handover testing:
```go
func TestCSVHandover(cfg *config.Config, csvCfg config.CSVHandoverConfig)
```
- Loads handover steps from CSV file
- Creates single UE with PDU session
- Executes Xn/N2 handovers based on CSV data
- Prints detailed timing results

## Event Distribution Pattern

### Single UE (direct)
```go
ueTasks := ueCtx.GetEventQueue()
ueTasks.AssignTask(&uecontext.EventUeData{EventType: model.RegisterInit})
```

### Multi-UE Group (via channel)
```go
// Send to group channel
group.GroupEvents <- uecontext.EventUeData{EventType: model.PduSessionInit}

// group.listenEvent() distributes to all UEs in remoteUEs map
func (g *UeGroup) distributeEvent(event *uecontext.EventUeData) {
    for _, ueRemote := range g.remoteUEs {
        if g.Delay > 0 {
            time.Sleep(time.Duration(g.Delay) * time.Millisecond)
        }
        ueRemote.tasks.AssignTask(event)
    }
}
```

### Handover Events (special handling)
```go
// Handovers are triggered on gNB side, not sent as tasks
if event.EventType == model.XnHandover || event.EventType == model.N2Handover {
    gnbcontext.TriggerXnHandover(g.curGnb, newGnb, int64(ueid))
    // or
    gnbcontext.TriggerNgapHandover(g.curGnb, newGnb, int64(ueid))
}
```

## Remote API Integration

### UE/gNB Registration
```go
// Called during UE/gNB creation
oamApi.addUes(ueCtx)
oamApi.addGnbs(gnbs)
```

### Key API Methods
| Method | Purpose |
|--------|---------|
| `RemoteGetListUes()` | List UEs filtered by state/log level |
| `RemoteGetListGnbs()` | List all registered gNBs |
| `RemoteGetUeCtx(msin)` | Get UE context info |
| `RemoteGetUeApi(ueId)` | Get `UeApi` interface for UE operations |
| `RemoteGetGnbApi(gnbId)` | Get `GnbApi` interface for gNB operations |
| `RemoteMmWorkerStats()` | Get 5GMM worker pool statistics |
| `RemoteSmWorkerStats()` | Get 5GSM worker pool statistics |
| `RemoteGetAllUeDelayStats()` | Aggregate delay stats across all UEs |

### Delay Statistics Aggregation
`RemoteGetAllUeDelayStats()` computes:
- Group mean/stddev across all UEs
- Per-procedure statistics (registration, PDU session, etc.)
- NAS request-response pair timing

## Initialization Sequence

### Worker Pool Setup
```go
func InitScenarioLogger(cfg *config.Config, maxPool, nSctpWorker int, httpSrv oam.OamServer, ctx context.Context) {
    pool.InitWorkerPool(ctx, maxPool, nSctpWorker, nUe, nGnb)
    uecontext.InitUeContextPool(&cfg.Testing, ctx)
    
    // Start OAM server if enabled
    if cfg.RemoteServer.Enable {
        oam.StartOamServer(addr, name, rootId, getter)
    }
}
```

### gNB Creation
```go
func createGnbs(nGnbs int, cfg config.GNodeBConfig, amfs []model.AMF, ...) map[string]*gnbcontext.GnbContext {
    for i := range nGnbs {
        // Increment port for each gNB
        currentControlIF.Port += i
        currentDataIF.Port += i
        
        gnb := gnbcontext.InitGnb(currentControlIF, currentDataIF, cfg.ListGnbs[i], amfs, ...)
        gnbs[cfg.ListGnbs[i].GnbId] = gnb
    }
}
```

## For AI Agents

### Adding New Scenario Types
1. Create new `test-*.go` file in this directory
2. Call `InitScenarioLogger()` for worker pool initialization
3. Use `createGnbs()` to create gNB instances
4. Register UEs/gNBs with `oamApi.addUes()` / `oamApi.addGnbs()`
5. Use `sendTask()` helper for event scheduling

### Working with UeGroup
- Use `newUeGroup()` to create a group with N UEs
- Events sent to `GroupEvents` channel are distributed to all UEs
- Handover events trigger via `gnbcontext.TriggerXnHandover/TriggerNgapHandover`
- Use `incrementMsin()` for sequential MSIN generation

### Handover Testing
- Use `HandoverTracker` for outcome tracking
- Call `StartHandover()` before triggering
- Call `CompleteHandover()` on success/failure
- Use `PrintSummary()` for results output

### Fuzz/Replay Mode
- Fuzz: Set `cfg.Testing.EnableFuzz = true`, UE starts with `RegisterInit`
- Replay: Load state with `uecontext.LoadFromFile()`, disable fuzz
- Capture saves to `logger_HHMMSS.yaml` on Ctrl+C

### Concurrency Notes
- `RemoteApi` uses `sync.Map` for thread-safe UE/gNB storage
- `HandoverTracker` uses `sync.RWMutex` for result access
- `UeGroup` uses `sync.RWMutex` for event distribution
- Do NOT hold locks during blocking operations

## Dependencies

### Internal
- `internal/core/uecontext` - UE context, event types, FSM
- `internal/core/gnbcontext` - gNB context, handover triggers
- `internal/common/pool` - Worker pool initialization
- `internal/common/ds` - Task queue (`ds.Tasks`)
- `internal/common/logger` - Scenario logging
- `internal/common/stats` - Procedure statistics
- `pkg/config` - Configuration structures, CSV loading
- `pkg/model` - Event types, states
- `monitoring/oambackend` - OAM API interfaces

### External
- `github.com/reogac/utils/oam` - OAM HTTP server
- `context`, `sync`, `time` - Standard library

<!-- MANUAL: Add notes about specific scenario implementation details here -->
