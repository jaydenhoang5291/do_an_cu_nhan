<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# gogtp5g-link

## Purpose
CLI tool for kernel-level GTP5G network interface lifecycle management. Creates and deletes gtp5g virtual network devices via rtnetlink, binding them to UDP sockets for GTP-U packet handling (port 2152).

## Key Files
| File | Description |
|------|-------------|
| `main.go` | Network interface management: `CmdAdd`/`CmdDel` with role configuration (UPF/RAN) |

## Subdirectories
None - single-file tool.

## For AI Agents

### Build Command
```bash
go build -o bin/gogtp5g-link ./monitoring/gtp5g/gogtp5g-link
```

### Usage
```bash
# Create gtp5g interface (UPF role, default)
./bin/gogtp5g-link add gtp5g0 192.168.1.1

# Create gtp5g interface (RAN role)
./bin/gogtp5g-link add gtp5g0 192.168.1.1 --ran

# Delete gtp5g interface
./bin/gogtp5g-link del gtp5g0
```

### Roles
- **UPF (role=0)**: Default - User Plane Function role
- **RAN (role=1)**: Radio Access Network role (use `--ran` flag)

### Implementation Details

**Interface Creation (`CmdAdd`):**
1. Opens rtnetlink connection via `nl.Open(syscall.NETLINK_ROUTE)`
2. Creates UDP socket bound to `<ip>:2152` (GTP-U port)
3. Passes file descriptor to kernel via `IFLA_FD1` attribute
4. Configures hashsize to 131072 for PDR hash table
5. Sets role via `IFLA_ROLE` attribute
6. Creates link via `rtnllink.Create()` with `IFLA_INFO_KIND="gtp5g"`
7. Brings interface up via `rtnllink.Up()`
8. Blocks on `stopSignal` channel until shutdown

**Interface Deletion (`CmdDel`):**
1. Opens rtnetlink connection
2. Removes interface via `rtnllink.Remove()`

### Netlink Client Pattern
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
wg.Go(func() {
    mux.Serve()
})

conn, err := nl.Open(syscall.NETLINK_ROUTE)
if err != nil {
    return err
}
defer conn.Close()

c := nl.NewClient(conn, mux)

// For add: rtnllink.Create(c, ifname, linkinfo)
// For del: rtnllink.Remove(c, ifname)
```

### Link Info Attributes
```go
linkinfo := &nl.Attr{
    Type: syscall.IFLA_LINKINFO,
    Value: nl.AttrList{
        {Type: rtnllink.IFLA_INFO_KIND, Value: nl.AttrString("gtp5g")},
        {Type: rtnllink.IFLA_INFO_DATA, Value: nl.AttrList{
            {Type: gtp5gnl.IFLA_FD1, Value: nl.AttrU32(fd)},      // UDP socket FD
            {Type: gtp5gnl.IFLA_HASHSIZE, Value: nl.AttrU32(131072)},
            {Type: gtp5gnl.IFLA_ROLE, Value: nl.AttrU32(role)},   // 0=UPF, 1=RAN
        }},
    },
}
```

## Dependencies

### External
- `github.com/free5gc/go-gtp5gnl` - GTP5G netlink attributes (`IFLA_FD1`, `IFLA_HASHSIZE`, `IFLA_ROLE`)
- `github.com/khirono/go-nl` - Generic netlink library (`NewMux`, `Open`, `NewClient`, `Attr`)
- `github.com/khirono/go-rtnllink` - RTNL link management (`Create`, `Remove`, `Up`)

### Kernel Requirements
- `gtp5g.ko` kernel module must be loaded
- Requires `CAP_NET_ADMIN` capability
- UDP port 2152 must be available

<!-- MANUAL: -->
