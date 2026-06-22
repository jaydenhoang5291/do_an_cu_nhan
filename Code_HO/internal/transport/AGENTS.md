<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Transport Layer

## Purpose
Transport layer for the StormSIM 5G emulator providing SCTP/NGAP communication with the 5G Core (N2 interface) and virtual radio links between UEs and gNBs. Implements high-performance, concurrent message passing using Go channels and worker pools.

## Key Files
| File | Description |
|------|-------------|
| `sctpngap/sctp.go` | SCTP connection management with read/write workers, mutex-protected calls, and pool integration |
| `sctpngap/ngapid.go` | NGAP PPID constant (60) with endianness detection |
| `rlink/link.go` | Virtual radio link connection (UE↔gNB bidirectional channels) |
| `rlink/pool.go` | Connection pool implementation (currently commented out) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `sctpngap/` | SCTP transport and NGAP protocol handling for N2 interface to AMF |
| `rlink/` | Virtual radio link layer simulating air interface between UE and gNB |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         5G Core (AMF)                           │
└───────────────────────────────┬─────────────────────────────────┘
                                │ SCTP/NGAP (N2 Interface)
┌───────────────────────────────▼─────────────────────────────────┐
│                      sctpngap.SctpConn                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ ReadWorkers  │  │WriteWorkers  │  │   ReadCh (chan []byte)│  │
│  │  (goroutines)│  │  (pond.Pool) │  │                      │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                        GnbContext                               │
└───────────────────────────────┬─────────────────────────────────┘
                                │ rlink.Connection
┌───────────────────────────────▼─────────────────────────────────┐
│                      rlink.Connection                           │
│  ┌──────────────────┐           ┌──────────────────┐           │
│  │   UplinkCh       │ UE → GNB  │   DownlinkCh     │ GNB → UE  │
│  │ (chan Message)   │──────────►│ (chan Message)   │◄──────────│
│  └──────────────────┘           └──────────────────┘           │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                         UeContext                               │
└─────────────────────────────────────────────────────────────────┘
```

## For AI Agents

### SCTP/NGAP Transport (`sctpngap/`)

**Creating SCTP Connection:**
```go
import "stormsim/internal/transport/sctpngap"

// Create SCTP connection to AMF
sctpConn := sctpngap.NewSctpConn(gnbID, localAddr, remoteAddr, ctx)
if err := sctpConn.Connect(); err != nil {
    log.Fatalf("SCTP connect failed: %v", err)
}

// Send NGAP message
sctpConn.Send(ngapEncodedBytes)

// Receive NGAP messages
for data := range sctpConn.Read() {
    // Process incoming NGAP message
    handleNgapMessage(data)
}

// Cleanup
sctpConn.Close()
```

**Critical SCTP Implementation Details:**

1. **Buffer Allocation**: Each read worker has its own buffer - do NOT use `sync.Pool` for read buffers to avoid data corruption across goroutines
```go
// CORRECT: Each worker has its own buffer
func (sc *SctpConn) readWorker() {
    buf := make([]byte, payloadSize) // Per-worker buffer
    // ...
}

// INCORRECT: Sharing buffers via sync.Pool causes data races
```

2. **Thread-Safe SCTP Calls**: SCTP read/write are protected by mutexes
```go
// readFromConn and writeToConn use mutexes internally
func (sc *SctpConn) readFromConn(buf []byte) (int, *sctp.SndRcvInfo, error) {
    sc.readMutex.Lock()
    defer sc.readMutex.Unlock()
    return sc.conn.SCTPRead(buf)
}
```

3. **Write Workers**: Uses subpool from `pool.SctpNgapWorkerPool`
```go
writeWorkers: pool.SctpNgapWorkerPool.NewSubpool(
    pool.GetMaxSctpNgapWorker()/2, 
    pond.WithContext(ctx),
)
```

4. **NGAP PPID**: Automatically handles endianness
```go
// ngapid.go handles little/big endian conversion
var NGAP_PPID uint32 = 60  // Standard NGAP PPID
```

### Virtual Radio Link (`rlink/`)

**Creating RLink Connection:**
```go
import "stormsim/internal/transport/rlink"

// Create bidirectional connection
conn := rlink.NewConnection(ueID, msin, gnbID, bufferSize, timeout)

// UE → GNB (Uplink)
if err := conn.SendUplink(msg); err != nil {
    log.Printf("Uplink send failed: %v", err)
}

// GNB → UE (Downlink)
if err := conn.SendDownlink(msg); err != nil {
    log.Printf("Downlink send failed: %v", err)
}

// Receive channels
uplinkCh := conn.GetUplinkChan()   // GNB listens here
downlinkCh := conn.GetDownlinkChan() // UE listens here

// Connection key for map storage
key := rlink.ConnectionKey(ueID, gnbID) // Returns "ueID:gnbID"
```

**Message Interface:**
```go
// Implement Message interface for custom message types
type Message interface {
    GetType() string
}

// Example implementation
type NasMessage struct {
    Type string
    Data []byte
}

func (m *NasMessage) GetType() string {
    return m.Type
}
```

**Connection Lifecycle:**
```go
// Check if closed
if conn.IsClosed() {
    // Connection was closed
}

// Close connection (idempotent)
conn.Close() // Closes both UplinkCh and DownlinkCh
```

### Configuration Constants

```go
// sctpngap/sctp.go
const (
    payloadSize               = 65535
    requestTimeout            = 2 * time.Second
    sctpDefaultNumOstreams    = 5
    sctpDefaultMaxInstreams   = 3
    sctpDefaultMaxAttempts    = 2
    sctpDefaultMaxInitTimeout = 2
    defaultChannelBuffer      = 5000
)

// rlink/link.go
const DefaultBufferSize int = 256
const DefaultDuration time.Duration = 3 * time.Second
```

### Common Patterns

**SCTP Message Processing Loop:**
```go
func (gnb *GnbContext) startNgapReceiver() {
    for data := range gnb.sctpConn.Read() {
        // Submit to GNB worker pool for processing
        pool.GnbWorkerPool.Submit(func() {
            gnb.handleNgapMessage(data)
        })
    }
}
```

**RLink Message Handling (GNB side):**
```go
func (gnb *GnbContext) handleUplink(conn *rlink.Connection) {
    for msg := range conn.GetUplinkChan() {
        switch msg.GetType() {
        case "nas":
            // Process NAS message from UE
            gnb.processNasFromUe(msg)
        }
    }
}
```

**RLink Message Handling (UE side):**
```go
func (ue *UeContext) handleDownlink(conn *rlink.Connection) {
    for msg := range conn.GetDownlinkChan() {
        switch msg.GetType() {
        case "nas":
            // Process NAS message from GNB
            ue.processNasFromGnb(msg)
        }
    }
}
```

### Error Handling

**SCTP Timeout Handling:**
```go
// Read workers use deadline-based reads for context checking
sc.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
    continue // Expected timeout, check context again
}
```

**RLink Timeout:**
```go
// Send methods timeout after configured duration
select {
case c.UplinkCh <- msg:
    return nil
case <-time.After(c.Timeout):
    return fmt.Errorf("timeout sending uplink message")
}
```

### Testing

```bash
# Run transport tests
go test ./internal/transport/...

# Run with verbose output
go test -v ./internal/transport/sctpngap/...
go test -v ./internal/transport/rlink/...
```

## Dependencies

### Internal
- `internal/common/logger` - Zerolog-based structured logging
- `internal/common/pool` - Worker pool management (`SctpNgapWorkerPool`)

### External
- `github.com/ishidawataru/sctp` - SCTP socket implementation
- `github.com/alitto/pond/v2` - Worker pool for async write operations

### Integration Points
- `internal/core/gnbcontext` - Consumes SCTP connection for NGAP handling
- `internal/core/uecontext` - Uses RLink connections for NAS message exchange
- `pkg/model` - Message types implementing `rlink.Message` interface

## Notes

- **RLink pool.go**: Contains commented-out `LinkPool` implementation. Currently connections are managed directly by UE/gNB contexts.
- **SCTP Stream IDs**: Round-robin across streams (0-4) for load distribution
- **Memory Safety**: Per-worker buffers in SCTP readers prevent data races
