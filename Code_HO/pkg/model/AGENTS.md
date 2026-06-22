<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# model - 3GPP Data Models

## Purpose
Defines shared data structures, FSM states, events, and RLink messages for the 5G emulator. This package is the canonical source for all 3GPP-related types used across the codebase - no protocol-specific enums or constants should be defined elsewhere.

## Key Files
| File | Description |
|------|-------------|
| `event-state.go` | FSM state types (`StateType`) and event types (`EventType`) for 5GMM/5GSM state machines |
| `ue.go` | UE types: `TunnelMode` (TUN/VRF), `Integrity` (NIA0-3), `Ciphering` (NEA0-3) |
| `gnb.go` | gNB types: `AMF`, `ControlIF`, `DataIF`, `GnbInfo` with TAC/PLMN/slice support |
| `pdusession.go` | `GnbPDUSessionContext`, `Teid`, `PagedUE`, `UeCoreContext` for session management |
| `slice.go` | Network slice types: `Plmn` (MCC/MNC), `Snssai` (SST/SD) |
| `rrc.go` | RLink message types for virtual radio communication (NAS, paging, handover) |

## FSM States (event-state.go)

### 5GMM States
| State | Description |
|-------|-------------|
| `NULL` | Initial null state |
| `IDLE` | Idle state |
| `Deregistered` | UE not registered with network |
| `DeregistrationInitiated` | Deregistration in progress |
| `AuthenticationInitiated` | Authentication in progress |
| `RegisteredInitiated` | Registration in progress |
| `Registered` | UE successfully registered |

### 5GSM States
| State | Description |
|-------|-------------|
| `PDUSessionInactive` | No active PDU session |
| `PDUSessionActivePending` | PDU session establishment in progress |
| `PDUSessionInactivePending` | PDU session release in progress |
| `PDUSessionActive` | PDU session established |
| `PDUModificationPending` | PDU session modification in progress |

## FSM Events (event-state.go)

### Lifecycle Events
| Event | Purpose |
|-------|---------|
| `EntryEvent` | FSM state entry callback |
| `ExitEvent` | FSM state exit callback |
| `Enable` | Enable event for initialization |

### 5GMM Events
| Event | Triggers |
|-------|----------|
| `GmmMessageEvent` | Incoming NAS GMM message |
| `InitRegistrationRequestEvent` | Start registration procedure |
| `RegistrationRejectEvent` | Network rejected registration |
| `AuthFailEvent` | Authentication failure |
| `SecurityModeFailEvent` | Security mode control failure |
| `RegistrationAcceptEvent` | Network accepted registration |
| `InitDeregistrationRequestEvent` | UE-initiated deregistration |
| `DeregistrationAcceptEvent` | Deregistration accepted |
| `NetworkDeregistrationRequestEvent` | Network-initiated deregistration |
| `MissingInfo` | Missing information in procedure |

### Timer Events
| Event | Timer |
|-------|-------|
| `T3502Event` | T3502 (registration retry) |
| `T3510Event` | T3510 (registration) |
| `T3511Event` | T3511 (identity request) |

### 5GSM Events
| Event | Triggers |
|-------|----------|
| `InitPduSessionEstablishmentRequestEvent` | Start PDU session establishment |
| `EstablishmentReject` | PDU session rejected |
| `EstablishmentAccept` | PDU session accepted |
| `ReleaseRequest` | UE-initiated release |
| `ReleaseCommand` | Network-initiated release |
| `ModificationRequest` | UE-initiated modification |
| `ModificationCommand` | Network-initiated modification |
| `ModificationReject` | Modification rejected |
| `ModificationComplete` | Modification completed |

### Trigger Commands (External Events)
| Event | Purpose |
|-------|---------|
| `NullInit` | Initialize from NULL state |
| `IdleInit` | Initialize from IDLE state |
| `RegisterInit` | Trigger registration |
| `DeregistraterInit` | Trigger deregistration |
| `ServiceRequestInit` | Trigger service request |
| `PduSessionInit` | Create PDU session |
| `DestroyPduSession` | Release PDU session |
| `XnHandover` | Xn-based handover |
| `N2Handover` | N2-based handover |
| `Terminate` | Graceful UE termination |
| `Kill` | Force UE termination |

## Data Types

### UE Types (ue.go)
```go
// GTP-U tunnel mode
type TunnelMode int
const (
    TunnelDisabled TunnelMode = iota  // No tunnel
    TunnelTun                          // TUN device only
    TunnelVrf                          // TUN + VRF device
)

// 5G integrity algorithms (NAS)
type Integrity struct {
    Nia0, Nia1, Nia2, Nia3 bool
}

// 5G ciphering algorithms (NAS)
type Ciphering struct {
    Nea0, Nea1, Nea2, Nea3 bool
}
```

### gNB Types (gnb.go)
```go
// AMF endpoint
type AMF struct {
    Ip   string
    Port int
}

// Control plane interface
type ControlIF struct {
    Ip   string
    Port int
}

// User plane interface
type DataIF struct {
    Ip   string
    Port int
}

// gNB configuration
type GnbInfo struct {
    GnbId            string    // 22-bit gNB ID
    Tac              string    // Tracking Area Code
    Plmn             Plmn      // PLMN identity
    SliceSupportList []Snssai  // Supported network slices
}
```

### Slice Types (slice.go)
```go
// PLMN identity
type Plmn struct {
    Mcc string  // Mobile Country Code (3 digits)
    Mnc string  // Mobile Network Code (2-3 digits)
}

// S-NSSAI (Single Network Slice Selection Assistance Information)
type Snssai struct {
    Sst string  // Slice/Service Type
    Sd  string  // Slice Differentiator
}
```

### PDU Session Types (pdusession.go)
```go
// GTP-U tunnel endpoint identifiers
type Teid struct {
    UplinkTeid   uint32   // UE -> UPF
    DownlinkTeid uint32   // UPF -> UE
}

// PDU session context at gNB
type GnbPDUSessionContext struct {
    Teid
    Snssai
    PduSessionId int64
    UpfIp        string
    PduType      uint64    // 0=IPv4, 1=IPv6, 2=IPv4v6, 3=Ethernet
    QosId        int64
    FiveQi       int64
    PriArp       int64
}

// Paged UE information
type PagedUE struct {
    FiveGSTMSI *ies.FiveGSTMSI
    Timestamp  time.Time
}

// UE core context for handover
type UeCoreContext struct {
    MobilityInfo           Plmn
    MaskedIMEISV           string
    PduSession             [16]*GnbPDUSessionContext
    AllowedSnssai          []Snssai
    LenSlice               int
    UeSecurityCapabilities *ies.UESecurityCapabilities
}
```

## RLink Messages (rrc.go)

RLink messages are exchanged between UE and gNB over the virtual radio interface (`internal/transport/rlink`).

### Message Types
| Type | Direction | Purpose |
|------|-----------|---------|
| `RLinkCreateConnectionRequest` | UE → gNB | UE requests RRC connection |
| `RLinkCreateConnectionResponse` | gNB → UE | gNB accepts connection |
| `RlinkUeIdleInform` | UE → gNB | UE entering idle mode |
| `RlinkUeReadyPaging` | UE → gNB | UE ready to receive paging |
| `RlinkSetupPaging` | gNB → UE | Deliver paging message |
| `RLinkHandoverPrepareRequest` | Source gNB → Target gNB | Initiate handover |
| `RlinkRlinkHandoverPrepareResponse` | Target gNB → Source gNB | Handover preparation result |
| `RLinkHandoverForwardUeContext` | Source gNB → Target gNB | Transfer UE context |
| `NasMsg` | Bidirectional | NAS message transport |
| `RlinkSetupPduSessonCommand` | gNB → UE | PDU session setup |

### Message Structures
```go
// All RLink messages implement GetType() string

type RLinkCreateConnectionRequest struct {
    PrUeId int64
    Msin   string         // IMSI/subscription ID
    Tmsi   *nas.Guti      // GUTI if available
    Conn   *rlink.Connection
}

type RLinkHandoverPrepareRequest struct {
    PrUeId       int64
    Conn         *rlink.Connection
    TargetGnbId  string
    IsXnHandover bool
    IsN2Handover bool
}

type RLinkHandoverForwardUeContext struct {
    PrUeId         int64
    Msin           string
    Conn           *rlink.Connection
    AmfUeNgapId    int64
    UeCoreContext  *UeCoreContext
    SourceGnbId    string
}

type NasMsg struct {
    PrUeId int64
    Nas    []byte  // Raw NAS message
}
```

## For AI Agents

### Import Pattern
```go
import "stormsim/pkg/model"

// Use type-safe state/event constants
currentState := model.Deregistered
event := model.InitRegistrationRequestEvent
```

### Common Patterns

**Checking FSM state:**
```go
if state == model.Registered {
    // UE is registered, can establish PDU sessions
}
```

**Creating PDU session context:**
```go
pduCtx := &model.GnbPDUSessionContext{
    Teid: model.Teid{
        UplinkTeid:   uplinkTeid,
        DownlinkTeid: downlinkTeid,
    },
    Snssai:       model.Snssai{Sst: "1", Sd: "010203"},
    PduSessionId: 1,
    UpfIp:        "10.0.0.1",
    PduType:      0, // IPv4
}
```

**Creating RLink messages:**
```go
// All RLink messages must implement GetType() string
msg := &model.NasMsg{
    PrUeId: ueId,
    Nas:    nasBytes,
}
msgType := msg.GetType() // Returns "NasMsg message"
```

**Handover context transfer:**
```go
handoverCtx := &model.RLinkHandoverForwardUeContext{
    PrUeId:        ueId,
    Msin:          msin,
    AmfUeNgapId:   amfUeNgapId,
    UeCoreContext: ueCoreContext,
    SourceGnbId:   sourceGnbId,
}
```

### Adding New States/Events
1. Add state constant to `event-state.go` under appropriate section (5GMM or 5GSM)
2. Add event constant if triggering a new transition
3. Update FSM transition table in `internal/core/uecontext/`
4. Document in AGENTS.md

### Adding New RLink Messages
1. Define message struct in `rrc.go`
2. Implement `GetType() string` method returning the message type constant
3. Add message type constant to `rrc.go`
4. Update handlers in `internal/transport/rlink/`

## Dependencies

### Internal
- `internal/transport/rlink` - `Connection` type used in RLink messages

### External
- `github.com/reogac/nas` - NAS protocol library (`nas.Guti`)
- `github.com/lvdund/ngap/ies` - NGAP information elements (`ies.FiveGSTMSI`, `ies.UESecurityCapabilities`)

## Subdirectories
None - this is a leaf package.
