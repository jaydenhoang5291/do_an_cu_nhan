<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# rlink

## Purpose
Virtual radio link transport layer simulating the air interface between UE and gNB. Provides bidirectional Go channel-based message passing with timeout protection and thread-safe lifecycle management. Replaces physical radio communication with in-memory channels for high-scale emulation (10,000+ UEs).

## Key Files
| File | Description |
|------|-------------|
| `link.go` | Core `Connection` struct with bidirectional channels (UplinkCh/DownlinkCh), `Message` interface, send/receive methods with timeout, lifecycle management |
| `pool.go` | Commented-out `LinkPool` implementation for centralized connection management (currently unused - connections managed directly by UE/gNB contexts) |

## For AI Agents

### Working In This Directory
- **Do NOT uncomment pool.go** - Connection management is handled directly by `GnbContext.rlinkPool` and `UeContext.rlinkConn`
- **Channel Direction**: Uplink = UE → GNB, Downlink = GNB → UE
- **Thread Safety**: Connection uses `sync.RWMutex` for `closed` state - safe for concurrent send/receive
- **Timeout Protection**: Both `SendUplink` and `SendDownlink` timeout after `c.Timeout` to prevent goroutine blocking

### Creating Connections
```go
import "stormsim/internal/transport/rlink"

// Create bidirectional connection (typically in uecontext/service.go or gnbcontext)
conn := rlink.NewConnection(
    ueID,                      // int64 UE identifier
    msin,                      // string MSIN (subscription ID)
    gnbID,                     // string gNB identifier
    rlink.DefaultBufferSize,   // 256 - channel buffer size
    rlink.DefaultDuration,     // 3 * time.Second - send timeout
)
```

### Message Interface Implementation
```go
// Message interface (link.go:13-15)
type Message interface {
    GetType() string
}

// Implementations live in pkg/model/rrc.go:
// - RrcConnectionRequest
// - RrcConnectionSetup
// - RrcConnectionComplete
// - RrcSecurityModeCommand
// - RrcUeContextRelease
```

### Sending Messages
```go
// UE → GNB (Uplink) - used by uecontext/gnb.go:sendGnb()
if err := conn.SendUplink(msg); err != nil {
    // "timeout sending uplink message" or "connection closed"
}

// GNB → UE (Downlink) - used by gnbcontext/sender.go:sendMsgToUe()
if err := conn.SendDownlink(msg); err != nil {
    // "timeout sending downlink message" or "connection closed"
}
```

### Receiving Messages
```go
// GNB listening for UE uplink (gnbcontext/service.go:listenToUE())
for msg := range conn.GetUplinkChan() {
    switch msg.GetType() {
    case "RrcConnectionRequest":
        // Handle RRC request from UE
    }
}

// UE listening for GNB downlink (uecontext/gnb.go:handleGnbMsg())
for msg := range conn.GetDownlinkChan() {
    switch msg.GetType() {
    case "RrcConnectionSetup":
        // Handle RRC setup from GNB
    }
}
```

### Connection Storage Pattern
```go
// GNB stores connections using ConnectionKey (gnbcontext/context.go:148)
key := rlink.ConnectionKey(ueID, gnbID) // Returns "ueID:gnbID"
gnb.controlPlaneInfo.rlinkPool.Store(key, conn)

// Retrieval
if val, ok := gnb.controlPlaneInfo.rlinkPool.Load(key); ok {
    conn := val.(*rlink.Connection)
}
```

### Connection Lifecycle
```go
// Check if closed (thread-safe)
if conn.IsClosed() {
    // Connection was already closed
}

// Close connection (idempotent, closes both channels)
conn.Close()

// Cleanup pattern (gnbcontext/context.go:163-165)
gnb.controlPlaneInfo.rlinkPool.Range(func(key, value any) bool {
    value.(*rlink.Connection).Close()
    return true
})
```

### Constants
```go
const DefaultBufferSize int = 256           // Channel buffer size
const DefaultDuration time.Duration = 3 * time.Second // Send timeout
```

## Dependencies

### Internal
- `pkg/model/rrc.go` - Message implementations (`RrcConnectionRequest`, etc.)
- `internal/core/gnbcontext` - Creates/stores connections in `rlinkPool` (sync.Map)
- `internal/core/uecontext` - Stores connection per UE, handles downlink
- `internal/core/gnbcontext/sender.go` - `SendToGnb()` for inter-gNB routing

### External
- Standard library only (`fmt`, `sync`, `time`)

## Notes

- **pool.go is commented out**: `LinkPool` was designed for centralized management but connections are now managed directly by `GnbContext.rlinkPool` (sync.Map)
- **No sync.Pool**: Channels are per-connection, not pooled
- **Timeout prevents deadlock**: Without timeout, full channels would block sender indefinitely
- **Bidirectional abstraction**: Single `Connection` object provides both directions, simplifying UE/gNB coordination
