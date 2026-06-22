<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# sctpngap

## Purpose
SCTP/NGAP transport layer for the N2 interface between the gNB and AMF. Manages SCTP connections with concurrent read/write workers, handles NGAP PPID encoding with endianness detection, and provides thread-safe message passing via Go channels. Designed for high-throughput NGAP message exchange with the 5G Core.

## Key Files
| File | Description |
|------|-------------|
| `sctp.go` | SCTP connection management (`SctpConn`), read/write workers, mutex-protected I/O, channel-based message passing |
| `ngapid.go` | NGAP PPID constant (60) with runtime endianness detection and conversion |

## For AI Agents

### Working In This Directory
- **Buffer Allocation**: NEVER use `sync.Pool` for read buffers. Each read worker must have its own buffer to prevent data races across goroutines.
- **Thread Safety**: SCTP read/write calls are protected by mutexes (`readMutex`, `writeMutex`). Always use `readFromConn()` and `writeToConn()` methods.
- **Worker Pool Integration**: Write operations use a subpool from `pool.SctpNgapWorkerPool`. Read workers are standalone goroutines.
- **Channel Buffering**: `ReadCh` has a 5000-message buffer. Messages are dropped (with warning) if the channel is full.

### Common Patterns

**Creating an SCTP Connection:**
```go
import "stormsim/internal/transport/sctpngap"

sctpConn := sctpngap.NewSctpConn(gnbID, localAddr, remoteAddr, ctx)
if err := sctpConn.Connect(); err != nil {
    log.Fatalf("SCTP connect failed: %v", err)
}
defer sctpConn.Close()
```

**Sending NGAP Messages:**
```go
// Non-blocking send via worker pool
sctpConn.Send(ngapEncodedBytes)

// Send internally uses pond pool for async writes
// Stream IDs are round-robined (0-4) across SCTP streams
```

**Receiving NGAP Messages:**
```go
// Read returns the receive channel
for data := range sctpConn.Read() {
    // Process incoming NGAP message
    handleNgapMessage(data)
}
// Channel closes when connection closes
```

**Per-Worker Buffer Pattern (Critical):**
```go
// CORRECT: Each worker has its own buffer
func (sc *SctpConn) readWorker() {
    buf := make([]byte, payloadSize) // Per-worker allocation
    for {
        n, _, err := sc.readFromConn(buf)
        // Copy data before sending to channel
        data := make([]byte, n)
        copy(data, buf[:n])
        sc.ReadCh <- data
    }
}

// INCORRECT: Sharing buffers via sync.Pool causes data corruption
```

**NGAP PPID Usage:**
```go
// ngapid.go automatically handles endianness at init
// Use directly in SCTP writes:
info := &sctp.SndRcvInfo{
    Stream: streamID,
    PPID:   sctpngap.NGAP_PPID, // 60 with correct endianness
}
```

### Configuration Constants
```go
const (
    payloadSize               = 65535      // Max SCTP payload
    requestTimeout            = 2 * time.Second
    sctpDefaultNumOstreams    = 5          // Number of SCTP streams
    sctpDefaultMaxInstreams   = 3
    sctpDefaultMaxAttempts    = 2
    sctpDefaultMaxInitTimeout = 2
    defaultChannelBuffer      = 5000       // ReadCh buffer size
)
```

### Error Handling
```go
// Read workers use deadline-based reads for context checking
sc.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
    continue // Expected timeout, check context again
}
```

## Dependencies

### Internal
- `internal/common/logger` - Structured logging (`*logger.Logger` embedded in `SctpConn`)
- `internal/common/pool` - Worker pool access (`GetMaxSctpNgapWorker()`, `SctpNgapWorkerPool`)

### External
- `github.com/ishidawataru/sctp` - SCTP socket implementation (`SCTPConn`, `SCTPAddr`, `SndRcvInfo`)
- `github.com/alitto/pond/v2` - Worker pool for async write operations

### Integration Points
- `internal/core/gnbcontext` - Primary consumer; creates SCTP connections for NGAP handling with AMF
- `pkg/config` - Provides gNB configuration (local/remote addresses)

## Notes

- **Stream Round-Robin**: Writes distribute across SCTP streams 0-4 for load balancing
- **Graceful Shutdown**: `Close()` waits for all read workers via `sync.WaitGroup`
- **Memory Safety**: The commented-out `bufferPool` at package level is NOT used for reads (intentionally)
- **Write Pool**: Uses `NewSubpool()` from the global `SctpNgapWorkerPool` with context cancellation support
