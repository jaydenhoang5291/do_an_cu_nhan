<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# cmd

## Purpose
Contains main entry points for StormSIM executables. This directory holds the primary binaries: the 5G emulator (`stormsim`) and the remote control client for interacting with a running emulator instance.

## Key Files
| File | Description |
|------|-------------|
| `emulator/emulator.go` | Main 5G network simulator entry point with CLI flags for config, PCAP, replay, and CSV handover modes |
| `client/client.go` | Remote control client with interactive shell and single-command modes for emulator interaction |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `emulator/` | 5G emulator binary - handles UE/gNB simulation, scenario execution, and protocol stacks |
| `client/` | Remote CLI client - connects to emulator's REST API for runtime control and monitoring |

## For AI Agents

### Working In This Directory
- **Do not add new files here** - Entry points are minimal and delegate to `internal/` packages
- **emulator.go** is the orchestration layer - it parses CLI args and calls `scenarios.TestScenarios()`, `scenarios.TestSingleUE()`, or `scenarios.TestCSVHandover()`
- **client.go** is a thin HTTP wrapper around `monitoring/oambackend` commands
- Both files are `package main` - they cannot be imported by other packages

### Testing Requirements
- Build verification: `go build -o bin/stormsim ./cmd/emulator` and `go build -o bin/client ./cmd/client`
- Integration tests only - these entry points require full system to run
- No unit tests expected in this directory

### Common Patterns

**Emulator CLI Flags:**
```bash
sudo ./stormsim -c config/config.yml                    # Run with config
sudo ./stormsim -c config.yml --pcap capture.pcap       # Capture traffic
sudo ./stormsim -r replay.log -c config.yml             # Replay mode (1 UE)
sudo ./stormsim -c config.yml --csv handover.csv        # CSV handover mode
sudo ./stormsim --config-help                           # Show config guide
```

**Client Usage:**
```bash
./client stats                    # Single command mode
./client stats -w                 # Watch mode (refresh every second)
./client stats -w -n 2s           # Watch with custom interval
./client                          # Interactive shell mode
```

**Emulator Initialization Sequence:**
1. `main()` → `cli.App` parses flags
2. `runScenarios()` → `config.LoadConfig()` loads YAML
3. Enables PCAP if `--pcap` flag set
4. Routes to scenario runner based on flags:
   - `--csv` → `scenarios.TestCSVHandover()`
   - `--replay` → `scenarios.TestSingleUE()`
   - default → `scenarios.TestScenarios()`

**Client Request Flow:**
1. Builds `CmdRequest{ContextId, Name, Args}` JSON
2. POSTs to `http://localhost:4000/cmd`
3. Parses `CmdResponse{Message, Error, Context}`
4. Updates shell prompt and available commands from response

## Dependencies

### Internal
- `internal/common/logger` - Ring buffer logging (emulator)
- `internal/scenarios` - UE/gNB scenario orchestration (emulator)
- `monitoring` - Traffic capture and OAM backend (emulator)
- `monitoring/oambackend` - Command definitions shared with server (client)
- `pkg/config` - YAML configuration loading and parsing (emulator)

### External
- `github.com/urfave/cli/v2` - CLI framework for emulator
- `github.com/urfave/cli/v3` - CLI types for command definitions (client)
- `github.com/abiosoft/ishell` - Interactive shell for client

<!-- MANUAL: Add new entry points only if creating a separate binary (e.g., debugging tool) -->
