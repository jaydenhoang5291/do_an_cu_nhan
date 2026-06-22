<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# UE Context

## Purpose
UE (User Equipment) context implementation containing 5GMM/5GSM state machines, NAS message handlers (N1MM/N1SM), 5G AKA authentication, security context management, timer engine, PDU session management, and GTP tunnel interface setup for the data plane.

This directory implements the UE-side of the 5G NAS protocol stack per 3GPP TS 24.501.

## Key Files

| File | Description |
|------|-------------|
| `ue.go` | `UeContext` struct, initialization, PDU session management, GUTI/SUCI handling |
| `session.go` | `PduSession` struct with 5GSM state machine, GTP tunnel info, netlink interfaces |
| `statemachine_5gmm.go` | 5GMM FSM transitions (Deregistered ↔ RegisteredInitiated ↔ Registered), callbacks |
| `statemachine_5gsm.go` | 5GSM FSM for PDU session lifecycle (Inactive ↔ ActivePending ↔ Active) |
| `handle_n1mm.go` | NAS 5GMM message handlers (Auth, SecurityMode, Registration, Deregistration, etc.) |
| `handle_n1sm.go` | NAS 5GSM message handlers (PDU session establish/release accept/reject) |
| `trigger.go` | Event trigger functions (registration, deregistration, PDU session, service request) |
| `auth.go` | `AuthContext` struct, 5G AKA authentication processing (RES*, KAMF derivation) |
| `gtp.go` | GTP tunnel interface setup using netlink, gogtp5g-link, gogtp5g-tunnel |
| `gnb.go` | GNB message handling (NAS send, N1SM send, GTP setup command, handover) |
| `service.go` | UE service routines (connect to gNB, listen to gNB, DRX paging) |
| `pool.go` | UE pool initialization, FSM setup with worker pools (MmWorkerPool, SmWorkerPool) |
| `task.go` | Task tracking with timing, state management, and timeout handling |
| `oam.go` | OAM API endpoints (RemoteUeInfo, RemoteUeStats, RemoteUeSessionInfo, delay logs) |
| `utils.go` | Utility functions (SNN derivation, 5GMM/5GSM cause code to string) |
| `replay.go` | Message capture/replay for fuzzing and testing |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `sec/` | Security package - milenage algorithm, SQN management, 5G AKA key derivation |
| `timer/` | Timer engine and NAS timer types (T3346-T3540) |

### sec/ Package

| File | Description |
|------|-------------|
| `secctx.go` | `SecurityContext` struct - NAS key derivation, AS key derivation (KGNB, NH) |
| `ueauth.go` | 5G AKA key derivation functions (KDF, KAUSF, KSEAF, KAMF, RES*) |
| `milenage.go` | Milenage algorithm implementation (F1-F5*, MAC validation) |
| `sqn.go` | Sequence number management for authentication sync |

### timer/ Package

| File | Description |
|------|-------------|
| `engine.go` | `TimerEngine` - concurrent timer management with timeout/expire callbacks |
| `timer.go` | Timer types (T3346-T3540) and durations per 3GPP spec |

## For AI Agents

### Critical Constraints

**NO MUTEXES IN HOT PATHS:**
- Use `sync/atomic` for counters and ID generation
- Use channels for coordination between goroutines
- The `ue.mutex` in `UeContext` should only be used for non-hot-path operations (rlink connection management)

**NO sync.Pool FOR SCTP READS:**
- Do NOT use `sync.Pool` for SCTP message buffers
- Data corruption can occur across worker goroutines

**FSM EVENT FLOW:**
- NEVER call `SendEvent()` from within an FSM callback
- Use `state.SetNextEvent()` to queue the next event instead
- This prevents recursive event processing and potential deadlocks

### State Machine Patterns

**5GMM FSM Transitions:**
```
Deregistered + InitRegistrationRequestEvent → RegisteredInitiated
RegisteredInitiated + RegistrationAcceptEvent → Registered
RegisteredInitiated + RegistrationRejectEvent → Deregistered
Registered + InitDeregistrationRequestEvent → Deregistered
```

**5GSM FSM Transitions:**
```
PDUSessionInactive + InitPduSessionEstablishmentRequestEvent → PDUSessionActivePending
PDUSessionActivePending + EstablishmentAccept → PDUSessionActive
PDUSessionActive + ReleaseRequest → PDUSessionInactivePending
PDUSessionInactivePending + ReleaseCommand → PDUSessionInactive
```

### Event Handling Pattern

```go
// In FSM callback - use SetNextEvent for chaining
func mm_Deregistered(state *fsm.State, event *fsm.EventData) {
    ueCtx := fsm.GetStateInfo[UeContext](state)
    switch event.Type() {
    case model.InitRegistrationRequestEvent:
        if err := ueCtx.triggerInitRegistration(); err != nil {
            ueCtx.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.MissingInfo))
        }
    }
}

// Never do this:
// ueCtx.sendEventMm(event) // WRONG - causes recursive event processing
```

### Sending NAS Messages

```go
// Send 5GMM NAS message to GNB
func (ue *UeContext) sendNas(nasPdu []byte) {
    ue.sendGnb(&model.NasMsg{PrUeId: int64(ue.id), Nas: nasPdu})
}

// Send 5GSM NAS message via UL NAS Transport
func (ue *UeContext) sendN1Sm(n1Sm []byte, sessionId uint8, requestType *uint8, params *map[string]any)
```

### PDU Session Management

```go
// Create new PDU session (max 15, index 0 reserved per 3GPP)
pduSession, err := ue.createPDUSession()

// Get existing session
pduSession, err := ue.getPduSession(pduSessionId)

// Delete session (closes channels, stops GTP)
ue.deletePduSession(pduSessionId)
```

### Authentication Flow

```go
// In handleAuthenticationRequest:
// 1. Extract RAND, AUTN, ngKSI from message
// 2. Process via auth.processAuthenticationInfo()
// 3. Handle AUTH_SUCCESS, AUTH_MAC_FAILURE, AUTH_SYNC_FAILURE
// 4. On success: create SecurityContext with derived KAMF
ue.secCtx = sec.NewSecurityContext(&ue.auth.ngKsi, ue.auth.kamf, false)
```

### Timer Usage

```go
// Create timer with timeout and expire callbacks
ue.timerEngine.CreateTimer(timer.TimerConfig{
    TimerType:   timer.T3510,
    Duration:    timer.T3510_duration,
    CountMax:    1,
    TimeoutFunc: func() { /* on each timeout */ },
    ExpireFunc:  func() { /* when max count reached */ },
})
ue.timerEngine.Start(timer.T3510)
```

### Worker Pool Integration

All FSM events are processed via worker pools defined in `internal/common/pool`:
- `pool.MmWorkerPool` - 5GMM state machine events
- `pool.SmWorkerPool` - 5GSM state machine events

```go
// FSM initialization in pool.go
func InitUeContextPool(fuzzOptions *config.TestingConf, ctx context.Context) {
    fsm_mm := initFSM(pool.MmWorkerPool)
    fsm_sm := initPduFSM(pool.SmWorkerPool)
}
```

### Known Issues

- **Fatal() Calls**: Many handlers use `ue.Fatal()` on errors instead of graceful recovery. Consider refactoring to return errors and handle gracefully.
- **Timer Race Conditions**: Timer management under saturated load may have race conditions.
- **GTP Interface**: Only one tunnel per UE is currently supported (session ID 1).

## Dependencies

### Internal
- `internal/common/fsm` - Async FSM framework
- `internal/common/pool` - Worker pools (MmWorkerPool, SmWorkerPool)
- `internal/common/logger` - Buffered logger with ring buffer
- `internal/common/stats` - Procedure statistics
- `internal/common/ds` - Data structures (task queue)
- `internal/core/gnbcontext` - GNB context for message routing
- `internal/transport/rlink` - Virtual radio link (UE↔gNB channels)
- `monitoring/gtp5g` - Kernel GTP tunnel management (gogtp5g-link, gogtp5g-tunnel)
- `monitoring/oambackend` - OAM API types
- `pkg/config` - UE configuration
- `pkg/model` - 3GPP data models, events, states

### External
- `github.com/reogac/nas` - NAS message encoding/decoding
- `github.com/reogac/utils/sec5g` - 5G security key derivation
- `github.com/reogac/sbi/models` - 3GPP service-based interface models
- `github.com/vishvananda/netlink` - Network interface management (TUN, VRF, routes)
- `github.com/alitto/pond/v2` - Worker pool management
- `gopkg.in/yaml.v2` - YAML parsing for replay sessions

## Testing

```bash
# Run security tests
go test ./internal/core/uecontext/sec/...

# Run all uecontext tests
go test ./internal/core/uecontext/...
```

## Message Flow Examples

### UE Registration
1. `triggerInitRegistration()` → encodes RegistrationRequest
2. `sendNas()` → sends via RLink to GNB
3. GNB → AMF via NGAP InitialUEMessage
4. AMF → AuthenticationRequest → `handleAuthenticationRequest()`
5. UE computes RES* → AuthenticationResponse
6. AMF → SecurityModeCommand → `handleSecurityModeCommand()`
7. UE derives NAS keys → SecurityModeComplete
8. AMF → RegistrationAccept → `handleRegistrationAccept()`
9. UE stores GUTI → RegistrationComplete
10. State: `Deregistered` → `RegisteredInitiated` → `Registered`

### PDU Session Establishment
1. `triggerInitPduSessionRequest()` → creates PduSession
2. FSM event: `InitPduSessionEstablishmentRequestEvent`
3. `triggerInitPduSessionRequestInner()` → encodes PduSessionEstablishmentRequest
4. `sendN1Sm()` → UL NAS Transport to GNB
5. AMF → PduSessionEstablishmentAccept → `handlePduSessionEstablishmentAccept()`
6. GNB sends `RlinkSetupPduSessonCommand` → `setupGtpInterface()`
7. Creates kernel GTP interface, FAR, PDR rules
8. State: `PDUSessionInactive` → `PDUSessionActivePending` → `PDUSessionActive`

<!-- MANUAL: Add notes about specific UE context implementation details here -->
