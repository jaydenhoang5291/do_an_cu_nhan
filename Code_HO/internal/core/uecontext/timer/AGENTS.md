<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# timer

## Purpose
NAS timer engine for UE context providing concurrent timer management for 5GMM/5GSM protocol timers. Implements retry/timeout logic with configurable durations and callbacks per 3GPP TS 24.501. Each timer supports a maximum retry count with separate callbacks for each timeout tick and final expiration.

## Key Files
| File | Description |
|------|-------------|
| `timer.go` | Timer types (T3346-T3527) and duration constants per 3GPP spec |
| `engine.go` | `TimerEngine` - concurrent timer management with context-based cancellation, retry logic, timeout/expire callbacks |

## For AI Agents

### Working In This Directory
- **Callbacks run in goroutines**: Both `TimeoutFunc` and `ExpireFunc` are executed via `go callback()` to avoid blocking the timer loop
- **Context-based cancellation**: Each timer uses `context.WithCancel` for clean shutdown
- **Manual triggers**: Use `TriggerTimeout()` to manually fire a timeout event without waiting for the duration
- **Auto-removal**: `Stop()` removes the timer from the engine; use `StopWithExpireFunc()` if you need the expire callback to run

### Common Patterns

**Creating and Starting a Timer:**
```go
// Create timer engine (typically done in UeContext initialization)
timerEngine := timer.NewTimerEngine()

// Create a timer with retry logic
timerEngine.CreateTimer(timer.TimerConfig{
    TimerType:   timer.T3510,              // Registration timer
    Duration:    timer.T3510_duration,     // 15 seconds
    CountMax:    5,                         // Retry 5 times
    TimeoutFunc: func() {                   // Called on each timeout
        ue.resendRegistrationRequest()
    },
    ExpireFunc: func() {                    // Called when CountMax reached
        ue.handleRegistrationTimeout()
    },
})

// Start the timer
timerEngine.Start(timer.T3510)

// Stop without expire callback
timerEngine.Stop(timer.T3510)

// Stop and run expire callback
timerEngine.StopWithExpireFunc(timer.T3510)
```

**Checking Timer Status:**
```go
isActive, currentCount, err := timerEngine.GetTimerStatus(timer.T3510)
if isActive && currentCount > 3 {
    // Timer has retried more than 3 times
}
```

**Manual Trigger (for testing or forced retry):**
```go
err := timerEngine.TriggerTimeout(timer.T3510)
// Non-blocking - fails if trigger already pending
```

**Timer Types and Durations (from timer.go):**
```go
T3346  // 15 minutes - Registration attempt delay
T3396  // 30 minutes - 5GMM back-off timer
T3445  // 12 hours    - Periodic registration update
T3502  // 12 minutes  - Detach timer
T3510  // 15 seconds  - Registration request retransmission
T3511  // 10 seconds  - Authentication retransmission
T3512  // 54 minutes  - Periodic registration timer
T3516  // 30 seconds  - Authentication failure timer
T3517  // 15 seconds  - Service request retransmission
T3519  // 1 minute    - 5GMM generic timer
T3520  // 15 seconds  - Security mode retransmission
T3521  // 15 seconds  - Identity request retransmission
T3525  // 1 minute    - PDU session modification
T3540  // 10 seconds  - Service request timer
T3527  // 15 seconds  - Deregistration retransmission
```

### Timer Lifecycle

```
CreateTimer() → Start() → [tick/tick/.../tick] → ExpireFunc()
                  ↓
              Stop() → removed from engine
                  ↓
         StopWithExpireFunc() → ExpireFunc() + removed
```

### Concurrency Notes
- `TimerEngine.mu` protects the `timers` map
- `Timer.mu` protects individual timer state (isActive, currentCount)
- Callbacks are executed in separate goroutines to prevent blocking
- `manualTrigger` channel is buffered (size 1) to prevent blocking on multiple triggers

## Dependencies

### Internal
- None (standalone package)

### External
- `context` - Timer cancellation
- `sync` - RWMutex for concurrent access
- `time` - Duration and ticker management
