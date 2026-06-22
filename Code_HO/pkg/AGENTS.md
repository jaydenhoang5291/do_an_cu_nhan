<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# pkg - Shared Packages

## Purpose
Contains shared packages used across the StormSIM codebase. The `model` package defines 3GPP data structures, FSM states/events, and inter-component messages. The `config` package handles YAML configuration loading, UE/gNB settings, and CSV-based handover event parsing.

## Key Files
| File | Description |
|------|-------------|
| `config/config.go` | Main `Config` struct, YAML loading, scenario/remote server/testing config |
| `config/gnbconf.go` | `GNodeBConfig` struct, DNS hostname resolution for AMF/gNB interfaces |
| `config/ueconf.go` | `UeConfig` struct with IMSI credentials, security settings, DNN, slice info |
| `config/csvreader.go` | CSV-based handover event loading for mobility testing scenarios |
| `model/event-state.go` | FSM state types (`StateType`) and event types (`EventType`) for 5GMM/5GSM |
| `model/ue.go` | UE types: `TunnelMode` (TUN/VRF), `Integrity` (NIA0-3), `Ciphering` (NEA0-3) |
| `model/gnb.go` | gNB types: `AMF`, `ControlIF`, `DataIF`, `GnbInfo` with TAC/PLMN/slice support |
| `model/pdusession.go` | `GnbPDUSessionContext`, `Teid`, `UeCoreContext` for session management |
| `model/slice.go` | Network slice types: `Plmn` (MCC/MNC), `Snssai` (SST/SD) |
| `model/rrc.go` | RLink message types for virtual radio communication (NAS, paging, handover) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `model/` | 3GPP data models, FSM states/events, RLink messages (see `model/AGENTS.md`) |
| `config/` | Configuration loading, YAML parsing, CSV handover events (see `config/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- **model package**: Import as `stormsim/pkg/model`. Use for all state/event constants and shared data structures.
- **config package**: Import as `stormsim/pkg/config`. Call `config.LoadConfig(path)` to load YAML configuration.
- State/event types are string constants - use `model.StateType` and `model.EventType` for type safety.

### Testing Requirements
```bash
# Run all tests in pkg/
go test ./pkg/...

# Run specific package tests
go test ./pkg/config/...
go test ./pkg/model/...
```

### Common Patterns

**Loading configuration:**
```go
import "stormsim/pkg/config"

cfg := config.LoadConfig("./config/config.yml")
// Access: cfg.GNodeBConfig, cfg.DefaultUe, cfg.AMFs, cfg.Scenarios
```

**Using FSM states and events:**
```go
import "stormsim/pkg/model"

// 5GMM states
currentState := model.Deregistered
nextState := model.RegisteredInitiated

// Events to trigger transitions
event := model.InitRegistrationRequestEvent
```

**Accessing UE security capabilities:**
```go
ueConfig := config.UeConfig{...}
secCap := ueConfig.GetUESecurityCapability() // Returns *nas.UeSecurityCapability
```

**Loading handover events from CSV:**
```go
cfg := config.CSVHandoverConfig{
    FilePath:     "handover.csv",
    GnbIdMapping: map[int]string{37: "000008", 42: "000009"},
    StepDelayMs:  100,
}
steps, err := config.LoadHandoverEventsFromCSV(cfg)
```

## Dependencies

### Internal
- `internal/common/logger` - Logging in config package
- `internal/transport/rlink` - Connection type used in RRC messages

### External
- `github.com/reogac/nas` - NAS protocol library (UE security capabilities, GUTI)
- `github.com/lvdund/ngap/ies` - NGAP information elements (FiveGSTMSI, UESecurityCapabilities)
- `gopkg.in/yaml.v2` - YAML configuration parsing

<!-- MANUAL: -->
