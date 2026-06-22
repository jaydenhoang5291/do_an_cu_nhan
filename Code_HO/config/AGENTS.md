<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Config Directory

## Purpose
YAML configuration files for StormSIM 5G emulator scenarios. Each file defines gNodeB settings, UE parameters, AMF connections, and test scenarios for different 5G Core networks (Open5GS, Free5GC, etrib5gc) and use cases (single UE, load testing, handover, fuzzing).

## Key Files
| File | Description |
|------|-------------|
| `config.yml` | Default development config (localhost, 1 UE, registration test) |
| `open5gs.yml` | Open5GS Core config (50 gNBs, 2470 UEs, large-scale registration) |
| `open5gs_1ue.yml` | Open5GS single-UE test with PDU session |
| `free5gc.yml` | Free5GC Core config (50 gNBs, 5000 UEs, large-scale registration) |
| `free5gc_1ue.yml` | Free5GC minimal test (2 UEs, registration + PDU session) |
| `etrib5gc.yml` | etrib5gc Core config (2 gNBs, 100 UEs) |
| `etrib5gc-loadtest.yml` | etrib5gc load testing configuration |
| `single-ue.yml` | Single UE test with registration + PDU session (localhost) |
| `load-test.yml` | Load testing template with configurable UE count |
| `handover.yml` | Handover test (N2/N2 handover between 2 gNBs) |
| `fuzz.yml` | Fuzzing test configuration with FSM state/event constraints |
| `du_lieu.csv` | CSV data file for test data injection |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| (none) | All config files are at root level |

## For AI Agents

### Working In This Directory
- **Default config path**: `./config/config.yml` (hardcoded in `pkg/config/config.go`)
- **Config loading**: Use `config.LoadConfig(path)` from `pkg/config/config.go`
- **YAML library**: `gopkg.in/yaml.v2`

### Configuration Structure
All YAML files follow this schema (defined in `pkg/config/config.go`):

```yaml
gnodeb:
  controlif:          # N2 interface (SCTP/NGAP to AMF)
    ip: "127.0.0.20"
    port: 9487
  dataif:             # N3 interface (GTP-U)
    ip: "127.0.0.20"
    port: 2152
  listGnbs:           # Multiple gNBs for handover scenarios
    - gnbid: "000008"
      tac: "000001"
      plmn: { mcc: "208", mnc: "93" }
      slicesupportlist: [{ sst: "01", sd: "010203" }]

defaultUe:            # Template for all UEs (MSIN incremented per UE)
  msin: "0000000000"  # Base MSIN (UE ID suffix)
  key: "..."          # USIM K (128-bit hex)
  opc: "..."          # USIM OPc (128-bit hex) - or use `op` for Free5GC
  amf: "8000"
  sqn: "00000000"
  dnn: "internet"
  routingindicator: "0000"
  hplmn: { mcc: "208", mnc: "93" }
  snssai: { sst: 01, sd: "010203" }
  integrity: { nia0: false, nia1: false, nia2: true, nia3: false }
  ciphering: { nea0: true, nea1: false, nea2: true, nea3: false }
  delay: 1000         # ms between events per UE group

amfif:                # AMF connection (SCTP endpoint)
  - ip: "127.0.0.8"
    port: 38412

scenarios:            # UE groups with event sequences
  - nUEs: 100
    gnbs: ["000001"]
    ueEvents:
      - event: "RegisterInit Event"
        delay: 0
        register_type: 0
      - event: "PduSessionInit Event"

remote:               # OAM backend REST API
  enable: true
  ip: "0.0.0.0"
  port: 4000

testconf:             # Fuzzing/replay constraints
  enableFuzz: false
  5gmm:
    states: ["Registered State", "Deregisterd State"]
    events: ["RegisterInit Event", "DeregistraterInit Event"]

loglevel: info        # info, trace, debug, warn, error, fatal, panic
logging:
  ueLogBufferSize: 50
  gnbLogBufferSize: 100
```

### Key Configuration Differences by Core

| Parameter | Open5GS | Free5GC | Notes |
|-----------|---------|---------|-------|
| USIM key | `opc` (OPc) | `op` (OP) | Free5GC uses OP, Open5GS uses OPc |
| PLMN | mcc: "999", mnc: "70" | mcc: "208", mnc: "93" | Must match Core config |
| SUCI params | Required (protectionScheme, homeNetworkPublicKey) | Optional | Open5GS SUCI support |

### Event Types (from `pkg/model/`)
- `"RegisterInit Event"` - Initial registration
- `"PduSessionInit Event"` - PDU session establishment
- `"DeregistraterInit Event"` - Deregistration
- `"DestroyPduSession Event"` - PDU session release
- `"XnHandover Event"` - Xn-based handover
- `"N2Handover Event"` - N2-based handover

### Testing Requirements
- **Unit tests**: See `pkg/config/csvreader_test.go` for CSV reader tests
- **Config validation**: Run emulator with `--config path/to/config.yml` and verify startup logs
- **No dedicated config tests**: Validation happens at runtime via `config.LoadConfig()`

### Common Patterns

**Creating a new scenario config:**
```yaml
# Copy from single-ue.yml or free5gc_1ue.yml as template
# Modify: gnodeb.controlif.ip, amfif.ip, scenarios[].nUEs, defaultUe.key/opc
```

**Load testing configuration:**
```yaml
scenarios:
  - nUEs: 1000
    gnbs: ["000001"]
    ueEvents:
      - event: "RegisterInit Event"
      - event: "PduSessionInit Event"
defaultUe:
  delay: 100  # ms between UE starts in group
loglevel: error  # Reduce log volume
```

**Handover test (requires 2+ gNBs):**
```yaml
gnodeb:
  listGnbs:
    - gnbid: "000008"
      # ... plmn, tac, slice
    - gnbid: "000009"
      # ... plmn, tac, slice
scenarios:
  - nUEs: 1
    gnbs: ["000008"]
    ueEvents:
      - event: "RegisterInit Event"
      - event: "PduSessionInit Event"  # Required before handover
      - event: "N2Handover Event"       # or "XnHandover Event"
```

## Dependencies

### Internal
- `pkg/config/config.go` - Main config loader and struct definitions
- `pkg/config/gnbconf.go` - GNodeB-specific config structs
- `pkg/config/ueconf.go` - UE-specific config structs
- `pkg/config/csvreader.go` - CSV data file reader
- `pkg/model/` - EventType, StateType enums used in scenarios

### External
- `gopkg.in/yaml.v2` - YAML parsing
- `github.com/reogac/nas` - NAS security capability handling

<!-- MANUAL: Add notes about site-specific config requirements here -->
