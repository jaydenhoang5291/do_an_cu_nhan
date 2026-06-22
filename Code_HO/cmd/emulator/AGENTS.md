<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# cmd/emulator

## Purpose
Main entry point for the StormSIM 5G network simulator binary (`stormsim`). This is the orchestration layer that parses CLI arguments, loads configuration, and dispatches to the appropriate scenario runner based on selected mode (normal, replay, or CSV handover).

## Key Files
| File | Description |
|------|-------------|
| `emulator.go` | Main entry point with CLI flags, configuration loading, and mode routing |

## Subdirectories
None - this is a leaf directory containing a single `package main` file.

## For AI Agents

### Working In This Directory
- **emulator.go is `package main`** - It cannot be imported by other packages
- **Keep this file minimal** - Business logic belongs in `internal/` packages
- **The file is pure orchestration** - It parses args and delegates to `internal/scenarios`
- **Do not add protocol handling here** - NGAP, NAS, FSM logic belongs in `internal/core/`

### Execution Modes
The emulator supports three mutually exclusive modes:

| Mode | Flag | Handler Function |
|------|------|------------------|
| Normal | (default) | `scenarios.TestScenarios(&cfg)` |
| Replay | `--replay FILE` | `scenarios.TestSingleUE(&cfg, true, replayFile)` |
| CSV Handover | `--csv FILE` | `scenarios.TestCSVHandover(&cfg, csvCfg)` |

### CLI Flags Reference
```bash
-c, --config FILE      Load configuration from YAML file
--pcap FILE            Capture SCTP/NAS traffic to PCAP file
-r, --replay FILE      Replay recorded scenario (1 UE only)
--config-help, --ch    Show detailed configuration guide
--csv FILE             Load handover events from CSV for mobility simulation
--gnb-map MAP          Map CSV gNB IDs to config IDs (format: "csvId1:gnbId1,csvId2:gnbId2")
--step-delay MS        Delay between handover steps (default: 1000ms)
```

### Initialization Sequence
```
main() → cli.App setup → runScenarios()
                           ├── setConfigFile() → config.LoadConfig()
                           ├── monitoring.CaptureTraffic() [if --pcap]
                           └── Route to scenario runner:
                               ├── --csv set    → scenarios.TestCSVHandover()
                               ├── --replay set → scenarios.TestSingleUE()
                               └── default      → scenarios.TestScenarios()
```

### When Modifying emulator.go
- **Adding new CLI flags**: Add to `cli.App.Flags` slice, then handle in `runScenarios()`
- **Adding new modes**: Create new flag, parse in `runScenarios()`, call appropriate `scenarios.*` function
- **Changing config help**: Edit the heredoc in `showConfigHelp()`
- **Version bump**: Update the `version` constant at top of file

### Testing
```bash
# Build verification
go build -o bin/stormsim ./cmd/emulator

# Runtime verification (requires root for SCTP)
sudo ./bin/stormsim --config-help
sudo ./bin/stormsim -c config/config.yml
sudo ./bin/stormsim -c config/config.yml --pcap test.pcap
sudo ./bin/stormsim -r replay.log -c config/config.yml
sudo ./bin/stormsim -c config/config.yml --csv handover.csv --gnb-map "1:000008,2:000009"
```

### Common Pitfalls
- **Do not add heavy logic here** - This file should remain thin delegation
- **Do not import internal/core packages directly** - Use `internal/scenarios` as the facade
- **Logger init is in `init()`** - `log` is available before `main()` runs
- **PCAP flag must be set before scenario start** - Already handled correctly in current code

## Dependencies

### Internal
- `internal/common/logger` - Ring buffer logging, initialized in `init()`
- `internal/scenarios` - UE/gNB scenario orchestration (TestScenarios, TestSingleUE, TestCSVHandover)
- `monitoring` - Traffic capture via `CaptureTraffic()`
- `pkg/config` - YAML configuration loading and gNB mapping parsing

### External
- `github.com/urfave/cli/v2` - CLI framework for flag parsing and help generation

### Standard Library
- `fmt` - Formatted output for config help
- `os` - Argument access via `os.Args`

<!-- MANUAL: This entry point is stable. Only modify when adding new execution modes or CLI flags -->
