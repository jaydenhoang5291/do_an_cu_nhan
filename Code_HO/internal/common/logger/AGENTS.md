<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Logger

## Purpose
High-performance logging infrastructure for StormSIM supporting 10,000+ concurrent UEs. Provides zerolog-based structured logging with colored console output, thread-safe ring buffers for per-entity log retrieval via OAM API, and delay tracking for NAS/NGAP request-response timing analysis.

## Key Files
| File | Description |
|------|-------------|
| `logger.go` | Zerolog-based Logger with colored level formatting (cyan DEBUG, green INFO, yellow WARN, red ERROR, magenta FATAL) |
| `ringbuffer.go` | Generic thread-safe RingBuffer[T] with generation tracking for safe in-place updates, avoids ABA problem |
| `buffered_logger.go` | BufferedLogger combining Logger + RingBuffer + DelayTracker, used per UE/gNB context |
| `delay_types.go` | DelayTracker for NAS/NGAP request-response timing, procedure duration statistics (registration, PDU establish) |
| `nas_types.go` | NAS message type names via `github.com/reogac/nas` constants, request/response pair mappings |
| `ngap_types.go` | NGAP message type names via type assertion on `github.com/lvdund/ngap/ies` structs |

## Subdirectories
None.

## For AI Agents

### Ring Buffer Design

**Generation Tracking for Safe Updates:**
The ring buffer uses monotonic generation counters to enable safe in-place updates. When a delay is calculated on receive, the original send entry is updated with the calculated delay.

```go
rb := logger.NewRingBuffer[logger.LogEntry](1000)

// Push returns index and generation for later update
idx, gen := rb.Push(entry)

// Update fails safely if slot was overwritten (generation mismatch)
success := rb.Update(idx, gen, updatedEntry) // Returns false if slot recycled

// Retrieve entries
all := rb.GetAll()      // Chronological order (oldest first)
recent := rb.GetLast(10) // Most recent first
count := rb.Count()
```

### BufferedLogger Usage (Per UE/gNB Context)

**Creation:**
```go
bl := logger.NewBufferedLogger(
    1000,              // buffer size (entries)
    "UE",              // entity type ("UE" or "gNB")
    "imsi-123456789",  // entity ID
    nil,               // optional fields map
    func() string { return ue.Fsm.CurrentState().String() }, // state getter for log entries
)
```

**Logging Methods (dual output: console + ring buffer):**
```go
bl.Info("Registration started")
bl.Warn("Timer T3510 expired")
bl.Error("NAS decode failed")
bl.Debug("GMM state: %s", state)
bl.Fatal("Critical failure") // Exits process
bl.Panic("Unrecoverable error")
bl.Trace("Detailed packet dump")
```

**Delay-Tracking Logging (for protocol messages):**
```go
// Use LogSend/LogReceive instead of Info() for protocol messages
bl.LogSend("nas", "RegistrationRequest")     // Records send timestamp
bl.LogReceive("nas", "AuthenticationRequest") // Calculates delay, updates ring buffer

bl.LogSend("ngap", "InitialUEMessage")
bl.LogReceive("ngap", "DownlinkNASTransport")

bl.LogSend("rlink", "UplinkData")
bl.LogReceive("rlink", "DownlinkData")
```

**Retrieving Logs (for OAM API):**
```go
logs := bl.GetLogs()                    // All entries
recent := bl.GetLastLogs(50)            // Last 50 entries
errors := bl.GetLogsByLevel("ERROR")    // Filter by level
json, _ := bl.GetLogsJSON()             // JSON export
count := bl.GetLogCount()
```

**Delay Statistics:**
```go
delays := bl.GetDelayLogs(50)                    // Last 50 delay entries
nasDelays := bl.GetDelayLogsByProtocol("nas", 20) // Filter by protocol
stats := bl.GetDelayStats()                       // Aggregated statistics
// stats.Mean, stats.StdDev, stats.Min, stats.Max, stats.Count
// stats.Procedures["registration"] - procedure-specific stats
// stats.NasPairs - per request/response pair stats
```

### Delay Tracking Mechanics

**Tracked Procedures (defined in `delay_types.go`):**
| Start Message | Procedure Name | End Message | Event Type |
|---------------|----------------|-------------|------------|
| RegistrationRequest | registration | RegistrationComplete | send |
| DeregistrationRequestFromUE | deregistration | DeregistrationAcceptFromUE | receive |
| PduSessionEstablishmentRequest | pdu_establish | PduSessionEstablishmentAccept | receive |
| PduSessionReleaseRequest | pdu_release | PduSessionReleaseComplete | send |

**NAS Request/Response Pairs (defined in `nas_types.go`):**
```go
// Forward lookup: request → expected responses
logger.NasRequestToResponses["RegistrationRequest"] = {"RegistrationReject", "IdentityRequest", "AuthenticationRequest"}

// Reverse lookup: response → valid preceding requests
logger.NasResponseToRequests["AuthenticationRequest"] = {"RegistrationRequest", "IdentityResponse", "AuthenticationFailure"}
```

**Protocol Types:**
- `"nas"` - NAS 5GMM/5GSM messages
- `"ngap"` - NGAP control plane messages
- `"rlink"` - Virtual radio link messages

### NAS/NGAP Type Name Helpers

**NAS Message Names:**
```go
name := logger.NasMsgTypeName(nas.RegistrationRequestMsgType) // "RegistrationRequest"
name := logger.NasMsgTypeName(msg.GmmMessage.GetMessageType())
```

**NGAP Message Names:**
```go
name := logger.NgapMsgTypeName(ngapMsg)           // Type assertion on struct pointer
name := logger.NgapProcedureCodeName(procedureCode) // Procedure code to name
```

### Log Entry Structure

```go
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`     // "INFO", "WARN", "ERROR", etc.
    Message   string    `json:"message"`
    State     string    `json:"state,omitempty"` // Current FSM state (if getter provided)
}

type DelayEntry struct {
    Protocol     string    `json:"protocol"`     // "nas", "ngap", "rlink"
    RequestType  string    `json:"requestType"`  // Human-readable request name
    ResponseType string    `json:"responseType"` // Human-readable response name
    SendTime     time.Time `json:"sendTime"`     // When request was sent
    DelayMs      float64   `json:"delayMs"`      // Round-trip delay in milliseconds
}
```

### Statistics Structures

```go
type StatsResult struct {
    Min    float64 `json:"min"`    // Minimum delay (ms)
    Max    float64 `json:"max"`    // Maximum delay (ms)
    Mean   float64 `json:"mean"`   // Mean delay (ms)
    StdDev float64 `json:"stdDev"` // Standard deviation (ms)
    Count  int     `json:"count"`  // Number of measurements
}

type DelayStats struct {
    StatsResult
    Procedures map[string]ProcedureStats `json:"procedures"` // Per-procedure stats
    NasPairs   []NasPairStats            `json:"nasPairs"`   // Per-pair stats
}

type GroupDelayStats struct {
    Mean      float64                   `json:"mean"`      // Mean across all entities
    StdDev    float64                   `json:"stdDev"`    // StdDev across entities
    Count     int                       `json:"count"`     // Total measurements
    UeCount   int                       `json:"ueCount"`   // Entities with measurements
    Procedures map[string]ProcedureStats `json:"procedures"`
    NasPairs   []NasPairStats            `json:"nasPairs"`
}
```

## Dependencies

### Internal
- None (standalone package)

### External
- `github.com/rs/zerolog` - Structured logging with console output
- `github.com/reogac/nas` - NAS message type constants for `NasMsgTypeName()`
- `github.com/lvdund/ngap/ies` - NGAP message structs for `NgapMsgTypeName()`

## Concurrency Notes

- **RingBuffer**: Protected by `sync.RWMutex`. Write operations (Push, Update) take full lock. Read operations (GetAll, GetLast, Count) take read lock.
- **DelayTracker**: Protected by `sync.RWMutex`. Pending request map and procedure history are mutex-protected.
- **BufferedLogger**: Thread-safe via embedded RingBuffer and DelayTracker mutexes.
- **Logger**: Zerolog is goroutine-safe; no additional synchronization needed.

## Performance Considerations

- Ring buffer size is fixed at creation; no dynamic resizing overhead
- Generation tracking enables O(1) updates without locking the entire buffer during calculation
- Statistics are calculated on-demand (not maintained incrementally) to minimize hot-path overhead
- `DlNasTransport` messages are excluded from delay tracking (line 165-167 in `delay_types.go`) as they are transport wrappers

<!-- MANUAL: Add notes about specific logger implementation details here -->
