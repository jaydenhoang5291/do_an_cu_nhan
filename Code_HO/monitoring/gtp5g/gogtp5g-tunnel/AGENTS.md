<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# gogtp5g-tunnel

## Purpose
CLI tool for managing 5G user plane rules (PDR, FAR, QER, URR) via generic netlink. Communicates with the kernel GTP5G module (`gtp5g.ko`) to configure GTP-U tunnel packet processing rules according to 3GPP TS 38.415 specifications.

## Key Files
| File | Description |
|------|-------------|
| `main.go` | CLI entry point with usage help and command tree routing |
| `cmd.go` | Command parser infrastructure (`CmdNode`, `CmdToken`, `CmdParser`) |
| `cmdtree.go` | Command tree mapping operations (add/mod/delete/get/list) to handlers |
| `cmd_pdr.go` | PDR (Packet Detection Rule) CRUD operations |
| `cmd_far.go` | FAR (Forwarding Action Rule) CRUD operations |
| `cmd_qer.go` | QER (QoS Enforcement Rule) CRUD operations |
| `cmd_urr.go` | URR (Usage Reporting Rule) CRUD operations |
| `oid.go` | OID parsing (`<id>` or `<seid>:<id>` format) |
| `oid_test.go` | Unit tests for OID parsing |
| `flowdesc.go` | SDF (Service Data Flow) filter description parser |

## For AI Agents

### Working In This Directory
- This is a **standalone CLI tool**, not a library package
- Build output: `bin/gogtp5g-tunnel`
- Requires `gtp5g.ko` kernel module loaded and `CAP_NET_ADMIN` capability
- All commands follow the same netlink client pattern (see below)

### Rule Types (3GPP TS 38.415)

**PDR (Packet Detection Rule):**
- Detects incoming packets based on UE IP, TEID, SDF filter
- Options: `--pcd`, `--hdr-rm`, `--far-id`, `--ue-ipv4`, `--f-teid`, `--sdf-*`, `--qer-id`, `--gtpu-src-ip`, `--buffer-usock-path`

**FAR (Forwarding Action Rule):**
- Defines forwarding behavior: forward, drop, buffer, notify
- Options: `--action`, `--hdr-creation`, `--fwd-policy`

**QER (QoS Enforcement Rule):**
- QoS control: gate status, bit rate limits (MBR/GBR), QFI
- Options: `--gate-status`, `--mbr-ul`, `--mbr-dl`, `--gbr-ul`, `--gbr-dl`, `--qer-corr-id`, `--rqi`, `--qfi`, `--ppi`
- Note: Deprecated options (`--mbr-uhigh`, `--mbr-ulow`, etc.) return errors

**URR (Usage Reporting Rule):**
- Usage measurement and reporting
- Note: `ParseURROptions` has TODO - options not yet implemented

### Common Patterns

**Netlink Client Pattern (used by all commands):**
```go
var wg sync.WaitGroup
mux, err := nl.NewMux()
if err != nil {
    return err
}
defer func() {
    mux.Close()
    wg.Wait()
}()
wg.Add(1)
go func() {
    mux.Serve()
    wg.Done()
}()

conn, err := nl.Open(syscall.NETLINK_GENERIC)
if err != nil {
    return err
}
defer conn.Close()

c, err := gtp5gnl.NewClient(conn, mux)
if err != nil {
    return err
}

link, err := gtp5gnl.GetLink(ifname)
if err != nil {
    return err
}

// Use gtp5gnl.CreateXxxOID, gtp5gnl.UpdateXxxOID, gtp5gnl.RemoveXxxOID, gtp5gnl.GetXxxOID
```

**Option Parsing Pattern:**
```go
func ParseXxxOptions(args []string) ([]nl.Attr, error) {
    var attrs []nl.Attr
    p := NewCmdParser(args)
    for {
        opt, ok := p.GetToken()
        if !ok {
            break
        }
        switch opt {
        case "--option-name":
            arg, ok := p.GetToken()
            if !ok {
                return attrs, fmt.Errorf("option requires argument %q", opt)
            }
            v, err := strconv.ParseUint(arg, 0, 32)
            if err != nil {
                return attrs, err
            }
            attrs = append(attrs, nl.Attr{
                Type:  gtp5gnl.CONSTANT,
                Value: nl.AttrU32(v),
            })
        default:
            return attrs, fmt.Errorf("unknown option %q", opt)
        }
    }
    return attrs, nil
}
```

**OID Parsing:**
```go
// Format: <id> or <seid>:<id>
oid, err := ParseOID("123:456")  // Returns gtp5gnl.OID{123, 456}
oid, err := ParseOID("456")      // Returns gtp5gnl.OID{456}
```

**SDF Flow Description Format:**
```
<action> <dir> <proto> from <src> <ports> to <dst> <ports>

action: permit | deny
dir:    in | out
proto:  ip | <number>
src:    any | assigned | <cidr>
dst:    any | assigned | <cidr>
ports:  <port>[,<port>]*  where port = <num> | <start>-<end>

Example: "permit out ip from any to 10.60.0.1 80,443"
```

### CLI Usage
```bash
# OID format: <id> or <seid>:<id>

# PDR operations
gogtp5g-tunnel add pdr gtp5g0 1:100 --pcd 255 --ue-ipv4 10.60.0.1 --far-id 1
gogtp5g-tunnel mod pdr gtp5g0 1:100 --pcd 128
gogtp5g-tunnel get pdr gtp5g0 1:100
gogtp5g-tunnel delete pdr gtp5g0 1:100
gogtp5g-tunnel list pdr

# FAR operations
gogtp5g-tunnel add far gtp5g0 1 --action 2 --hdr-creation 0 0x12345678 10.0.0.1 2152
gogtp5g-tunnel list far

# QER operations
gogtp5g-tunnel add qer gtp5g0 1 --gate-status 1 --mbr-ul 1000000 --mbr-dl 2000000 --qfi 9
gogtp5g-tunnel list qer

# URR operations
gogtp5g-tunnel add urr gtp5g0 1
gogtp5g-tunnel list urr
```

### Build & Test
```bash
# Build
go build -o bin/gogtp5g-tunnel ./monitoring/gtp5g/gogtp5g-tunnel

# Run tests
go test ./monitoring/gtp5g/gogtp5g-tunnel/... -v
```

## Dependencies

### Internal
- None (standalone tool)

### External
- `github.com/free5gc/go-gtp5gnl` - GTP5G netlink client library (rule CRUD operations)
- `github.com/khirono/go-nl` - Generic netlink library (mux, connection, attribute handling)

<!-- MANUAL: -->
