<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# OAM Backend

## Purpose
REST API and CLI handlers for Operations, Administration, and Maintenance (OAM) interface. Implements a context-based command hierarchy (StormSim root → UE → gNB contexts) using `urfave/cli/v3` for command parsing. Provides query capabilities for UE/gNB state, sessions, statistics, delay measurements, and procedure triggering.

## Key Files
| File | Description |
|------|-------------|
| `backend.go` | Core API interface definition and context routing. Defines `Api` interface and context ID constants (`stormsim`, `ue:`, `gnb:` prefixes) |
| `emulator.go` | Emulator-level commands: `list-ue`, `list-gnb`, `select-ue`, `select-gnb`, `stats`, `group-delay-stats`. Root context handler |
| `ue.go` | UE context commands: `info`, `ssinfo`, `stats`, `ps-create`, `logs`, `delay-logs`, `delay-stats`. Implements `UeApi` interface |
| `gnb.go` | gNB context commands: `info`, `list-amf`, `count-ue`, `list-ue`, `release-ue`, `logs`, `delay-logs`, `delay-stats`. Implements `GnbApi` interface |
| `stats.go` | Procedure statistics command with JSON/CSV output, historical snapshots, and watch mode support |
| `models.go` | Data models for OAM responses: `UeContextInfo`, `GnbInfo`, `AmfInfo`, `SessionInfo`, `WorkerInfo`, `DelayStats`, `GroupDelayStats`, `NasDelayEntry` |
| `formatter.go` | Table rendering (`RenderTable`) and CSV file output (`WriteCSV`) utilities. Auto-filename generation |
| `watch.go` | Real-time watch functionality stub (server-side disabled, client handles polling) |

## Subdirectories
None. This is a leaf package.

## For AI Agents

### Context Hierarchy
The OAM system uses a 3-level context hierarchy:
```
stormsim (root)
├── ue:<msin>     # UE context (e.g., ue:000001)
└── gnb:<gnbid>   # gNB context (e.g., gnb:gnb001)
```

Each context has its own handler and command map:
- Root: `EmuHandler` + `EmuCmds`
- UE: `UeHandler` + `UeCmds`
- gNB: `GnbHandler` + `GnbCmds`

### API Interfaces

**Root API (`Api` in `backend.go`):**
```go
type Api interface {
    RemoteGetListUes(level, state, notState string, last int) []string
    RemoteGetListGnbs() []string
    RemoteGetUeCtx(supi string) UeContextInfo
    RemoteGetSessions(supi string) SessionInfo
    RemoteMmWorkerStats() WorkerInfo
    RemoteSmWorkerStats() WorkerInfo
    RemoteGetGnbApi(gnbId string) GnbApi
    RemoteGetUeApi(msin string) UeApi
    RemoteGetAllUeDelayStats() GroupDelayStats
}
```

**UE API (`UeApi` in `ue.go`):**
```go
type UeApi interface {
    RemoteUeInfo() UeContextInfo
    RemoteUeStats() []string
    RemoteUeSessionInfo() []SessionInfo
    RemoteCreateSession(dnn string, slice models.Snssai) bool
    RemoteUeLogs(last int, level string) []logger.LogEntry
    RemoteUeDelayLogs(last int, protocol string) []NasDelayEntry
    RemoteUeDelayStats() DelayStats
}
```

**gNB API (`GnbApi` in `gnb.go`):**
```go
type GnbApi interface {
    RemoteGnbInfo() GnbInfo
    RemoteListAmf() []AmfInfo
    RemoteCountUes() int
    RemoteListUeCtxs() []string
    RemoteReleaseUe(msin string) bool
    RemoteGnbLogs(last int, level string) []logger.LogEntry
    RemoteGnbDelayLogs(last int) []NasDelayEntry
    RemoteGnbDelayStats() DelayStats
}
```

### Command Structure Pattern

Each handler follows this pattern:
```go
type XxxHandler struct {
    api         XxxApi      // Remote API interface
    emuApi      Api         // Parent emulator API (for context switching)
    nextContext *oam.HandlerContext
}

var XxxCmds map[string]cli.Command = map[string]cli.Command{
    "command-name": {
        Name:                  "command-name",
        Usage:                 "Short description",
        Description:           "Long description",
        EnableShellCompletion: true,
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "file", Usage: "Output to file"},
            &cli.BoolFlag{Name: "watch", Aliases: []string{"w"}},
        },
        Action: func(ctx context.Context, cmd *cli.Command) error {
            h := ctx.Value("handler").(*XxxHandler)
            // Use h.api to call remote methods
            return nil
        },
    },
}

func init() {
    XxxCmds["exit"] = cli.Command{
        Name: "exit",
        Usage: "Return to main menu",
        Action: func(ctx context.Context, cmd *cli.Command) error {
            h := ctx.Value("handler").(*XxxHandler)
            h.nextContext = oam.NewHandlerContext(STORMSIM_CTX_ID, 
                &EmuHandler{api: h.emuApi}, EmuCmds, nil)
            return nil
        },
    }
}
```

### Output Formatting

**Table Output:**
```go
f := NewFormatter(cmd.Writer)
headers := []string{"Key", "Value"}
rows := [][]string{{"Name", "value"}}
f.RenderTable("=== Title ===", headers, rows)
```

**CSV Output:**
```go
filePath := ResolveFilename(cmd.String("file"), "context", "command")
f.WriteCSV(filePath, headers, rows)
```

**Auto-filename format:** `20060102-150405-context-command.csv`

### Common Flags

| Flag | Type | Purpose |
|------|------|---------|
| `--file` | string | Output to file (use `auto` for timestamp-based filename) |
| `--json` | bool | Output in JSON format |
| `--csv` | bool | Output in CSV format (stats command) |
| `--watch, -w` | bool | Real-time watch mode |
| `--interval, -n` | string | Watch interval (e.g., `500ms`, `2s`) |
| `--last` | int | Show last N entries |
| `--level` | string | Filter by log level (INFO, WARN, ERROR, DEBUG) |
| `--history` | bool | Show historical statistics |
| `--since` | string | Show snapshots since duration (e.g., `5m`, `1h`) |

### Available Commands by Context

**Root (stormsim):**
| Command | Description |
|---------|-------------|
| `list-ue` | List UE contexts with state filtering |
| `list-gnb` | List all gNBs |
| `select-ue --msin <msin>` | Enter UE context |
| `select-gnb --gnbId <id>` | Enter gNB context |
| `stats` | Show procedure statistics |
| `group-delay-stats` | Show aggregated NAS delay across all UEs |

**UE Context (ue:<msin>):**
| Command | Description |
|---------|-------------|
| `info` | Show UE context info (name, gnb, PLMN, NGAP IDs, state) |
| `ssinfo` | Show PDU session info |
| `stats` | Show event/message processing stats |
| `ps-create --dnn <dnn> --sst <sst> --sd <sd>` | Trigger PDU session establishment |
| `logs` | Show buffered logs |
| `delay-logs` | Show NAS delay measurements |
| `delay-stats` | Show aggregated delay statistics |
| `exit` | Return to root context |

**gNB Context (gnb:<gnbid>):**
| Command | Description |
|---------|-------------|
| `info` | Show gNB info (name, PLMN, slice, addresses) |
| `list-amf` | List connected AMFs |
| `count-ue` | Count UEs attached to gNB |
| `list-ue` | List UE contexts at gNB |
| `release-ue --msin <msin>` | Release UE connection |
| `logs` | Show buffered logs |
| `delay-logs` | Show NGAP delay measurements |
| `delay-stats` | Show aggregated NGAP delay statistics |
| `exit` | Return to root context |

### Data Models

Key models defined in `models.go`:

```go
type UeContextInfo struct {
    Name, Gnb, Hplmn, Snssai string
    RanNgapId, AmfNgapId     int64
    MMstate                  string
    ActiveSessions           int8
}

type GnbInfo struct {
    Name, Plmn, Snssai       string
    NgapAddr, GtpAddr        string
}

type AmfInfo struct {
    Name         string
    Address      model.AMF
    State        string
    PlmnSupport  []string
    SliceSupport []string
}

type DelayStats struct {
    Min, Max, Mean, StdDev float64
    Count                  int
    Procedures             map[string]ProcedureStats
    NasPairs               []NasPairStats
}

type GroupDelayStats struct {
    Mean, StdDev float64
    Count, UeCount int
    Procedures   map[string]ProcedureStats
    NasPairs     []NasPairStats
}
```

### Adding New Commands

1. Add method to appropriate API interface
2. Implement command in `XxxCmds` map
3. Use `ctx.Value("handler").(*XxxHandler)` to access handler
4. Follow output formatting patterns (table/CSV/JSON)
5. Add `--file` flag for export capability
6. Use `NewFormatter(cmd.Writer)` for consistent output

## Dependencies

### Internal
- `internal/common/logger` - Ring buffer log entries (`logger.LogEntry`)
- `internal/common/stats` - Procedure statistics and historical snapshots (`stats.GlobalStats`, `stats.GlobalHistory`)
- `pkg/model` - 3GPP data models (`model.AMF`, `model.Snssai`)

### External
- `github.com/reogac/utils/oam` - OAM context management framework (`oam.HandlerContext`, `oam.NewHandlerContext`)
- `github.com/urfave/cli/v3` - CLI command parsing (`cli.Command`, `cli.Flag`)
- `github.com/reogac/sbi/models` - SBI models (`models.Snssai`)
