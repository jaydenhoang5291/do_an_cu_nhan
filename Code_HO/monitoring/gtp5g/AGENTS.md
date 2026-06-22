<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# GTP5G Tunnel Management

## Purpose
Kernel-level GTP5G tunnel management tools for StormSIM. Provides two standalone CLI utilities:
- **gogtp5g-link**: GTP5G network interface creation/deletion via rtnetlink
- **gogtp5g-tunnel**: 5G user plane rule management (PDR, FAR, QER, URR) via generic netlink

These tools communicate with the kernel GTP5G module (gtp5g.ko) to configure user plane packet processing rules for GTP-U tunneling.

## Key Files

### gogtp5g-link/
| File | Description |
|------|-------------|
| `main.go` | Network interface management: `add`/`del` gtp5g interfaces with role (UPF/RAN) |

### gogtp5g-tunnel/
| File | Description |
|------|-------------|
| `main.go` | CLI entry point with usage help and command routing |
| `cmdtree.go` | Command tree definition mapping operations to handlers |
| `cmd.go` | Command parser infrastructure (`CmdNode`, `CmdParser`) |
| `cmd_pdr.go` | PDR (Packet Detection Rule) CRUD operations |
| `cmd_far.go` | FAR (Forwarding Action Rule) CRUD operations |
| `cmd_qer.go` | QER (QoS Enforcement Rule) CRUD operations |
| `cmd_urr.go` | URR (Usage Reporting Rule) CRUD operations |
| `oid.go` | OID parsing (`<id>` or `<seid>:<id>` format) |
| `oid_test.go` | Unit tests for OID parsing |
| `flowdesc.go` | SDF (Service Data Flow) filter description parser |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `gogtp5g-link/` | Network interface lifecycle management (add/delete gtp5g devices) |
| `gogtp5g-tunnel/` | 5G user plane rule configuration (PDR/FAR/QER/URR) |

## For AI Agents

### Build Commands
```bash
# Build both tools
go build -o bin/gogtp5g-link ./monitoring/gtp5g/gogtp5g-link
go build -o bin/gogtp5g-tunnel ./monitoring/gtp5g/gogtp5g-tunnel

# Run tests
go test ./monitoring/gtp5g/gogtp5g-tunnel/... -v
```

### gogtp5g-link Usage
```bash
# Create gtp5g interface (UPF role, default)
./bin/gogtp5g-link add gtp5g0 192.168.1.1

# Create gtp5g interface (RAN role)
./bin/gogtp5g-link add gtp5g0 192.168.1.1 --ran

# Delete gtp5g interface
./bin/gogtp5g-link del gtp5g0
```

### gogtp5g-tunnel Usage
```bash
# OID format: <id> or <seid>:<id>

# PDR operations
./bin/gogtp5g-tunnel add pdr gtp5g0 1:100 --pcd 255 --ue-ipv4 10.60.0.1 --far-id 1
./bin/gogtp5g-tunnel mod pdr gtp5g0 1:100 --pcd 128
./bin/gogtp5g-tunnel get pdr gtp5g0 1:100
./bin/gogtp5g-tunnel delete pdr gtp5g0 1:100
./bin/gogtp5g-tunnel list pdr

# FAR operations
./bin/gogtp5g-tunnel add far gtp5g0 1 --action 2 --hdr-creation 0 0x12345678 10.0.0.1 2152
./bin/gogtp5g-tunnel list far

# QER operations
./bin/gogtp5g-tunnel add qer gtp5g0 1 --gate-status 1 --mbr-ul 1000000 --mbr-dl 2000000 --qfi 9
./bin/gogtp5g-tunnel list qer

# URR operations
./bin/gogtp5g-tunnel add urr gtp5g0 1
./bin/gogtp5g-tunnel list urr
```

### Rule Types (3GPP TS 38.415)

**PDR (Packet Detection Rule):**
- Detects incoming packets based on: UE IP, TEID, SDF filter
- Options: `--pcd`, `--hdr-rm`, `--far-id`, `--ue-ipv4`, `--f-teid`, `--sdf-*`, `--qer-id`, `--gtpu-src-ip`, `--buffer-usock-path`

**FAR (Forwarding Action Rule):**
- Defines forwarding behavior: forward, drop, buffer, notify
- Options: `--action`, `--hdr-creation`, `--fwd-policy`

**QER (QoS Enforcement Rule):**
- QoS control: gate status, bit rate limits, QFI
- Options: `--gate-status`, `--mbr-ul`, `--mbr-dl`, `--gbr-ul`, `--gbr-dl`, `--qer-corr-id`, `--rqi`, `--qfi`, `--ppi`

**URR (Usage Reporting Rule):**
- Usage measurement and reporting
- Note: `ParseURROptions` has TODO - options not yet implemented

### Netlink Client Pattern
All tunnel commands follow this pattern:
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

// Use gtp5gnl.CreateXxxOID, gtp5gnl.UpdateXxxOID, etc.
```

### SDF Flow Description Format
Parsed by `ParseFlowDesc()`:
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

### OID (Object Identifier)
Parsed by `ParseOID()`:
- Simple: `456` → `{456}`
- Session-scoped: `123:456` → `{123, 456}` where 123 is SEID, 456 is rule ID

## Dependencies

### External
- `github.com/free5gc/go-gtp5gnl` - GTP5G netlink client library (rule CRUD)
- `github.com/khirono/go-nl` - Generic netlink library
- `github.com/khirono/go-rtnllink` - RTNL link management (interface add/del)

### Kernel Requirements
- `gtp5g.ko` kernel module must be loaded
- Requires `CAP_NET_ADMIN` capability

<!-- MANUAL: -->
