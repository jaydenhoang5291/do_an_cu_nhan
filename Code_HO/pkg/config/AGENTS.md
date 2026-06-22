<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# pkg/config - Configuration Loading

## Purpose
Handles YAML configuration loading for the 5G emulator, including gNB settings, UE credentials, AMF connections, scenarios, and CSV-based handover event parsing for mobility testing.

## Key Files
| File | Description |
|------|-------------|
| `config.go` | Main `Config` struct, YAML loading via `LoadConfig()`, scenario/event definitions, remote server config, fuzz testing config |
| `gnbconf.go` | `GNodeBConfig` struct, DNS hostname resolution via `resolvHost()` for AMF/gNB interfaces |
| `ueconf.go` | `UeConfig` struct with IMSI credentials, security settings (integrity/ciphering), DNN, slice info, `GetUESecurityCapability()` |
| `csvreader.go` | `LoadHandoverEventsFromCSV()` for mobility testing, `HandoverStep` struct, `ParseGnbMapping()` for gNB ID translation |
| `csvreader_test.go` | Unit tests for CSV reader and gNB mapping parsing |

## Subdirectories
None - this is a leaf package.

## For AI Agents

### Core Types

**Config (root configuration):**
```go
type Config struct {
    GNodeBConfig GNodeBConfig  // gNB control/data interfaces, list of gNBs
    DefaultUe    UeConfig      // Default UE credentials and settings
    AMFs         []model.AMF   // AMF connection endpoints
    Scenarios    []Scenario    // UE groups with event sequences
    RemoteServer RemoteServer  // Remote API server config
    Testing      TestingConf   // Fuzz testing configuration
    LogLevel     string        // Global log level
    Logging      LoggingConfig // Log buffer sizes
}
```

**UeConfig (UE credentials and settings):**
```go
type UeConfig struct {
    UeId       int             // Runtime UE ID (not from YAML)
    Msin       string          // IMSI suffix (after MCC/MNC)
    Key, Opc   string          // USIM authentication keys (hex)
    Op, Amf    string          // Operator variant, AMF field
    Sqn        string          // Sequence number for authentication
    Dnn        string          // Data Network Name
    Hplmn      model.Plmn      // Home PLMN (MCC/MNC)
    Snssai     model.Snssai    // Network slice (SST/SD)
    Integrity  model.Integrity // NIA0-3 enabled flags
    Ciphering  model.Ciphering // NEA0-3 enabled flags
    TunnelMode model.TunnelMode // TUN/VRF (runtime only)
}
```

**GNodeBConfig (gNB interfaces):**
```go
type GNodeBConfig struct {
    DefaultControlIF model.ControlIF // N2/Control interface (IP, port)
    DefaultDataIF    model.DataIF    // N3/Data interface (IP, port)
    ListGnbs         []model.GnbInfo // Individual gNB configs
}
```

**Scenario and EventInfo (UE orchestration):**
```go
type Scenario struct {
    NUEs     int         // Number of UEs in this scenario
    Gnbs     []string    // gNB IDs for this scenario
    UeEvents []EventInfo // Events to execute for each UE
}

type EventInfo struct {
    Event                 model.EventType // Registration, PDU session, etc.
    TimeBeforeExcuteEvent uint8           // Delay before event (seconds)
    NumberPduSessions     int             // PDU sessions to establish
    RegisterType          int             // 0=Initial, 1=Emergency
    DeregisterType        int             // 0=Not switch off, 1=Switch off
    PduSessionType        int             // PDU session type
    Params                []int           // Additional parameters
}
```

**HandoverStep (CSV mobility testing):**
```go
type HandoverStep struct {
    Step         int    // Step index from CSV
    FromGnbId    string // Source gNB ID
    ToGnbId      string // Target gNB ID
    HandoverType int    // 1=Xn handover, 2=N2 handover
}
```

### Common Patterns

**Loading YAML configuration:**
```go
import "stormsim/pkg/config"

cfg := config.LoadConfig("./config/config.yml")
// Access configuration sections:
//   cfg.GNodeBConfig.DefaultControlIF.Ip
//   cfg.DefaultUe.Msin
//   cfg.AMFs[0].Ip
//   cfg.Scenarios[0].NUEs
```

**Getting UE security capabilities:**
```go
ueConfig := config.UeConfig{
    Ciphering: model.Ciphering{Nea0: true, Nea2: true},
    Integrity: model.Integrity{Nia0: true, Nia2: true},
}
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
if err != nil {
    // Handle error
}
for _, step := range steps {
    if step.IsXnHandover() {
        // Xn handover from step.FromGnbId to step.ToGnbId
    }
}
```

**Parsing gNB mapping string:**
```go
mapping, err := config.ParseGnbMapping("37:000008,42:000009,59:000010")
// Result: map[int]string{37: "000008", 42: "000009", 59: "000010"}
```

### DNS Resolution

The `resolvHost()` function (internal, called by `LoadConfig`) resolves hostnames to IPv4 addresses:
- Resolves AMF IP addresses from configuration
- Resolves gNB control and data interface IPs
- Logs warnings for skipped IPv6 addresses
- Fatals if no suitable IPv4 address found

### CSV Format for Handover Testing

Required columns in CSV:
| Column | Description |
|--------|-------------|
| `Bước` | Step index |
| `ue0_BS_ketnoi` | Connected base station ID |
| `ue0_handover` | 1 if handover occurs, 0 otherwise |
| `ue0_handover_to_type` | 1=Xn, 2=N2 |

### Testing Requirements
```bash
# Run all config tests
go test ./pkg/config/...

# Run with verbose output
go test -v ./pkg/config/...
```

### Default Values

Applied by `LoadConfig()` when not specified in YAML:
- `Logging.UeLogBufferSize`: 50
- `Logging.GnbLogBufferSize`: 100

## Dependencies

### Internal
- `stormsim/pkg/model` - Data types: `AMF`, `ControlIF`, `DataIF`, `GnbInfo`, `Plmn`, `Snssai`, `Integrity`, `Ciphering`, `TunnelMode`, `EventType`, `StateType`
- `stormsim/internal/common/logger` - Logging (`InitLogger`, `ParseLogLevel`)

### External
- `gopkg.in/yaml.v2` - YAML parsing
- `github.com/reogac/nas` - NAS protocol library (`nas.UeSecurityCapability`)
- Standard library: `os`, `net`, `encoding/csv`, `strconv`, `strings`, `fmt`

<!-- MANUAL: -->
