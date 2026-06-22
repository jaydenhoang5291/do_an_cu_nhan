<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Monitoring

## Purpose
Observability, REST API, and GTP tunnel management for StormSIM. Provides OAM (Operations, Administration, Maintenance) interface for querying UE/gNB state, statistics, delay measurements, and kernel-level GTP5G tunnel rule management via netlink.

## Key Files
| File | Description |
|------|-------------|
| `pcap.go` | PCAP traffic capture on N2 interface using gopacket/pcapgo |
| `oambackend/backend.go` | OAM API interface and context routing (StormSim → UE → gNB) |
| `oambackend/emulator.go` | Emulator-level commands: `list-ue`, `list-gnb`, `select-ue`, `select-gnb`, `stats`, `group-delay-stats` |
| `oambackend/ue.go` | UE context commands: `info`, `ssinfo`, `stats`, `ps-create`, `logs`, `delay-logs`, `delay-stats` |
| `oambackend/gnb.go` | gNB context commands: `info`, `list-amf`, `count-ue`, `list-ue`, `release-ue`, `logs`, `delay-logs`, `delay-stats` |
| `oambackend/stats.go` | Procedure statistics with JSON/CSV output and historical snapshots |
| `oambackend/models.go` | Data models: `UeContextInfo`, `GnbInfo`, `AmfInfo`, `SessionInfo`, `DelayStats`, `GroupDelayStats` |
| `oambackend/formatter.go` | Table rendering and CSV file output utilities |
| `oambackend/watch.go` | Real-time watch functionality (server-side disabled) |
| `gtp5g/gogtp5g-link/main.go` | GTP5G network interface creation/deletion via rtnllink |
| `gtp5g/gogtp5g-tunnel/main.go` | CLI entry point for tunnel rule management |
| `gtp5g/gogtp5g-tunnel/cmdtree.go` | Command tree parser for add/mod/delete/get/list operations |
| `gtp5g/gogtp5g-tunnel/cmd_pdr.go` | PDR (Packet Detection Rule) CRUD via go-gtp5gnl |
| `gtp5g/gogtp5g-tunnel/cmd_far.go` | FAR (Forwarding Action Rule) CRUD via go-gtp5gnl |
| `gtp5g/gogtp5g-tunnel/cmd_qer.go` | QER (QoS Enforcement Rule) CRUD via go-gtp5gnl |
| `gtp5g/gogtp5g-tunnel/cmd_urr.go` | URR (Usage Reporting Rule) CRUD via go-gtp5gnl |
| `gtp5g/gogtp5g-tunnel/oid.go` | OID parsing (`<id>` or `<seid>:<id>` format) |
| `gtp5g/gogtp5g-tunnel/flowdesc.go` | SDF filter flow description parsing |
| `resource/cpu_ram.py` | CPU/RAM usage logger using psutil |
| `resource/chart.ipynb` | Jupyter notebook for data visualization |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `oambackend/` | REST API and CLI handlers for OAM operations (see `oambackend/AGENTS.md`) |
| `gtp5g/` | Kernel GTP-U tunnel management tools (see `gtp5g/AGENTS.md`) |
| `resource/` | Python scripts for resource monitoring and charting |

## For AI Agents

### Working In This Directory
- **OAM commands** use `urfave/cli/v3` for CLI parsing with context-based handler routing
- **Context hierarchy**: StormSim root → `ue:<msin>` or `gnb:<gnbid>` contexts
- **GTP5G tools** are standalone CLIs built to `bin/` - not imported as libraries
- **Delay statistics** track NAS/NGAP/RLINK request-response latencies per UE/gNB

### Testing Requirements
```bash
# Build GTP5G tools
go build -o bin/gogtp5g-link ./monitoring/gtp5g/gogtp5g-link
go build -o bin/gogtp5g-tunnel ./monitoring/gtp5g/gogtp5g-tunnel

# Run unit tests
go test ./monitoring/...

# Test OID parsing
go test ./monitoring/gtp5g/gogtp5g-tunnel/... -v
```

### Common Patterns

**OAM Handler Pattern:**
```go
type XxxHandler struct {
    api         XxxApi      // Remote API interface
    emuApi      Api         // Parent emulator API
    nextContext *oam.HandlerContext
}

var XxxCmds map[string]cli.Command = map[string]cli.Command{
    "command-name": {
        Name:  "command-name",
        Flags: []cli.Flag{...},
        Action: func(ctx context.Context, cmd *cli.Command) error {
            h := ctx.Value("handler").(*XxxHandler)
            // Use h.api to call remote methods
            return nil
        },
    },
}
```

**GTP5G Netlink Pattern:**
```go
mux, _ := nl.NewMux()
defer mux.Close()
conn, _ := nl.Open(syscall.NETLINK_GENERIC)
defer conn.Close()
c, _ := gtp5gnl.NewClient(conn, mux)
link, _ := gtp5gnl.GetLink(ifname)
gtp5gnl.CreatePDROID(c, link, oid, attrs)
```

**Formatter Output:**
```go
f := NewFormatter(cmd.Writer)
f.RenderTable("=== Title ===", []string{"Col1", "Col2"}, rows)
f.WriteCSV(filePath, headers, rows)
```

## Dependencies

### Internal
- `internal/common/logger` - Ring buffer log entries for UE/gNB
- `internal/common/stats` - Procedure statistics and historical snapshots
- `pkg/model` - 3GPP data models (Snssai, AMF address)
- `pkg/config` - Configuration loading

### External
- `github.com/reogac/utils/oam` - OAM context management framework
- `github.com/urfave/cli/v3` - CLI command parsing
- `github.com/free5gc/go-gtp5gnl` - GTP5G netlink client library
- `github.com/khirono/go-nl` - Generic netlink library
- `github.com/khirono/go-rtnllink` - RTNL link management
- `github.com/google/gopacket` - Packet capture and parsing
- `github.com/google/gopacket/pcapgo` - PCAP file writing
- `github.com/vishvananda/netlink` - Network interface management

<!-- MANUAL: -->
