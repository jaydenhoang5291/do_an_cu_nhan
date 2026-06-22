# StormSIM Architecture

## Table of Contents

- [System Overview](#system-overview)
- [Component Architecture](#component-architecture)
- [Data Flow](#data-flow)
- [State Machines](#state-machines)
- [Concurrency Model](#concurrency-model)
- [Module Reference](#module-reference)
- [Transport Layer](#transport-layer)
- [Observability](#observability)
- [Anti-Patterns & Technical Debt](#anti-patterns--technical-debt)

---

## System Overview

StormSIM is a high-performance 5G UE and gNodeB emulator designed for testing and benchmarking 5G Core networks. The system simulates the radio access network (RAN) side of 5G, handling both control plane (NGAP/NAS) and user plane (GTP-U) protocols.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              5G Core (AMF/SMF/UPF)                          │
└─────────────────────────────────┬───────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │    SCTP/NGAP (N2)         │
                    │    GTP-U (N3)             │
                    └─────────────┬─────────────┘
                                  │
┌─────────────────────────────────┴───────────────────────────────────────────┐
│                            StormSIM Emulator                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                        GnbContext (gNodeB)                               ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ ││
│  │  │   AMF Pool   │  │   UE Pool    │  │ NGAP Dispatch│  │  SCTP Conn  │ ││
│  │  │  (sync.Map)  │  │  (sync.Map)  │  │  (handlers)  │  │  Manager    │ ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘ ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                  │                                           │
│                    ┌─────────────┴─────────────┐                            │
│                    │   RLink (Virtual Radio)   │                            │
│                    │   UE↔gNB Channels         │                            │
│                    └─────────────┬─────────────┘                            │
│                                  │                                           │
│  ┌───────────────────────────────┴───────────────────────────────────────┐  │
│  │                    UeContext (per UE)                                 │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐ │  │
│  │  │  5GMM FSM    │  │  5GSM FSM    │  │ NAS Handler  │  │  Timers    │ │  │
│  │  │ (MmWorker)   │  │ (SmWorker)   │  │ (Encode/Dec) │  │  Engine    │ │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Observability Layer                                │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │ Ring Buffer  │  │ Delay Track  │  │   Stats      │               │   │
│  │  │   Logger     │  │   (per UE)   │  │ (per proc)   │               │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │     OAM Backend           │
                    │     (REST API)            │
                    └───────────────────────────┘
```

---

## Component Architecture

### Directory Structure

```
StormSIM/
├── cmd/                        # Application entrypoints
│   └── emulator/               # Main emulator binary
├── internal/
│   ├── core/                   # Core protocol handling
│   │   ├── uecontext/          # UE context, FSMs, NAS handlers
│   │   └── gnbcontext/         # gNB context, NGAP handlers
│   ├── common/                 # Shared infrastructure
│   │   ├── fsm/                # Async FSM framework
│   │   ├── pool/               # Worker pool management
│   │   ├── logger/             # Ring buffer logging
│   │   └── stats/              # Procedure statistics
│   ├── transport/              # Transport layer
│   │   ├── rlink/              # Virtual radio (UE↔gNB)
│   │   └── sctpngap/           # SCTP/NGAP transport
│   └── scenarios/              # UE orchestration
├── pkg/
│   ├── model/                  # 3GPP data models, events, states
│   └── config/                 # Configuration structures
├── monitoring/
│   ├── oambackend/             # REST API for control/metrics
│   └── gtp5g/                  # Kernel GTP-U tunnel management
├── config/                     # YAML configuration files
└── docs/                       # Documentation
```

### Worker Pool Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Worker Pool System                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │  MmWorkerPool   │    │  SmWorkerPool   │    │  GnbWorkerPool  │         │
│  │  (5GMM Events)  │    │  (5GSM Events)  │    │  (NGAP Events)  │         │
│  │                 │    │                 │    │                 │         │
│  │  ┌───────────┐  │    │  ┌───────────┐  │    │  ┌───────────┐  │         │
│  │  │ MmWorker  │  │    │  │ SmWorker  │  │    │  │ GnbWorker │  │         │
│  │  │ (gorount.)│  │    │  │ (gorount.)│  │    │  │ (gorount.)│  │         │
│  │  └───────────┘  │    │  └───────────┘  │    │  └───────────┘  │         │
│  │       ...       │    │       ...       │    │       ...       │         │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘         │
│                                                                              │
│  ┌─────────────────┐                                                        │
│  │  SctpWorkerPool │                                                        │
│  │  (SCTP I/O)     │                                                        │
│  │                 │                                                        │
│  │  ┌───────────┐  │                                                        │
│  │  │SctpWorker │  │                                                        │
│  │  │ (gorount.)│  │                                                        │
│  │  └───────────┘  │                                                        │
│  │       ...       │                                                        │
│  └─────────────────┘                                                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

Event Flow:
UE Event → Channel → Worker picks → FSM dispatch → State transition → Actions
```

---

## Data Flow

### Registration Procedure

```
┌──────┐     ┌───────┐     ┌─────────┐     ┌─────┐
│  UE  │     │  gNB  │     │RLink    │     │ AMF │
│Ctx   │     │Ctx    │     │Channel  │     │     │
└──┬───┘     └───┬───┘     └────┬────┘     └──┬──┘
   │             │              │             │
   │ RegRequest  │              │             │
   │─────────────┼──────────────►             │
   │             │              │             │
   │             │  NGAP:RRCSetupComplete     │
   │             │──────────────┼─────────────►
   │             │              │             │
   │             │              │  NGAP:DownlinkNasTransport
   │             │◄─────────────┼─────────────┤
   │             │              │             │
   │ AuthRequest │◄─────────────┤             │
   │◄────────────┤              │             │
   │             │              │             │
   │ AuthResponse│              │             │
   │─────────────┼──────────────►             │
   │             │              │             │
   │             │  NGAP:UplinkNasTransport   │
   │             │──────────────┼─────────────►
   │             │              │             │
   │             │              │  ... Security Mode ...
   │             │              │             │
   │ RegComplete │◄─────────────┤             │
   │◄────────────┤              │             │
   │             │              │             │
   │ [REGISTERED]│              │             │
   │             │              │             │
```

### PDU Session Establishment

```
┌──────┐     ┌───────┐     ┌─────────┐     ┌─────┐     ┌─────┐
│  UE  │     │  gNB  │     │RLink    │     │ AMF │     │ SMF │
│Ctx   │     │Ctx    │     │Channel  │     │     │     │     │
└──┬───┘     └───┬───┘     └────┬────┘     └──┬──┘     └──┬──┘
   │             │              │             │           │
   │PduSessReq   │              │             │           │
   │─────────────┼──────────────►             │           │
   │             │              │             │           │
   │             │  NGAP:UL NAS │             │           │
   │             │──────────────┼─────────────►           │
   │             │              │             │           │
   │             │              │             │ Nsmf_PDUSession
   │             │              │             │──────────►│
   │             │              │             │           │
   │             │              │             │  PDU Session Accept
   │             │              │             │◄──────────┤
   │             │              │             │           │
   │             │  NGAP:PDU Session Resource Setup
   │             │◄─────────────┼─────────────┤           │
   │             │              │             │           │
   │PduSessAccept│◄─────────────┤             │           │
   │◄────────────┤              │             │           │
   │             │              │             │           │
   │ [ACTIVE]    │              │             │           │
   │             │              │             │           │
```

### Handover (Xn/N2)

```
┌──────┐  ┌───────┐  ┌───────┐  ┌─────┐
│  UE  │  │gNB-S  │  │gNB-T  │  │ AMF │
│Ctx   │  │(Source)│  │(Target)│  │     │
└──┬───┘  └───┬───┘  └───┬───┘  └──┬──┘
   │          │          │         │
   │HO Trigger│          │         │
   │──────────►          │         │
   │          │          │         │
   │          │ Xn:HOReq │         │
   │          │─────────►│         │
   │          │          │         │
   │          │ Xn:HOAck │         │
   │          │◄─────────┤         │
   │          │          │         │
   │ HOCommand│◄─────────┤         │
   │◄─────────┤          │         │
   │          │          │         │
   │ HO to T  │          │         │
   │──────────┼──────────►         │
   │          │          │         │
   │          │          │NGAP:HandoverNotify
   │          │          │────────►│
   │          │          │         │
   │[CONNECTED to T]      │         │
   │          │          │         │
```

---

## State Machines

### 5GMM State Machine (Mobility Management)

```
                    ┌─────────────────────┐
                    │                     │
          ┌────────►│   MM_DEREGISTERED   │◄────────┐
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │         RegisterInit Event              │
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │         │
          │         │  MM_REGISTERING     │         │
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │         Registration Success            │ Deregister Event
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │─────────┘
          │         │   MM_REGISTERED     │
          │         │                     │◄────────┐
          │         └──────────┬──────────┘         │
          │                    │                    │
          │        ServiceRequest Event             │
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │         │
          │         │ MM_SERVICE_REQUEST  │─────────┘
          │         │                     │   Timeout/Fail
          │         └──────────┬──────────┘
          │                    │
          │         Paging Event
          │                    │
          │                    ▼
          │         ┌─────────────────────┐
          │         │                     │
          └─────────│    MM_PAGING        │
                    │                     │
                    └─────────────────────┘
```

**5GMM States:**
- `MM_DEREGISTERED` - Initial state, no NAS signaling connection
- `MM_REGISTERING` - Registration procedure in progress
- `MM_REGISTERED` - Successfully registered with network
- `MM_SERVICE_REQUEST` - Service request procedure active
- `MM_PAGING` - Responding to network paging
- `MM_DEREGISTERING` - Deregistration in progress

**Key Events:**
- `RegisterInitEvent` - Start registration
- `RegistrationAcceptEvent` - Registration accepted by AMF
- `RegistrationRejectEvent` - Registration rejected
- `ServiceRequestEvent` - Initiate service request
- `PagingEvent` - Network paging received
- `DeregisterEvent` - Initiate deregistration

### 5GSM State Machine (Session Management)

```
                    ┌─────────────────────┐
                    │                     │
          ┌────────►│    SM_PDU_INACTIVE   │◄────────┐
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │         PduSessionInit Event            │
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │         │
          │         │  SM_PDU_PENDING     │         │
          │         │                     │         │
          │         └──────────┬──────────┘         │
          │                    │                    │
          │         PduSessionAccept Event          │ Release Event
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │─────────┘
          │         │    SM_PDU_ACTIVE    │
          │         │                     │◄────────┐
          │         └──────────┬──────────┘         │
          │                    │                    │
          │         Modification Event              │
          │                    │                    │
          │                    ▼                    │
          │         ┌─────────────────────┐         │
          │         │                     │         │
          │         │  SM_MODIFYING       │─────────┘
          │         │                     │  Complete/Reject
          │         └─────────────────────┘
          │                    │
          │         Release Event
          │                    │
          │                    ▼
          │         ┌─────────────────────┐
          │         │                     │
          └─────────│   SM_RELEASING      │
                    │                     │
                    └─────────────────────┘
```

**5GSM States:**
- `SM_PDU_INACTIVE` - No PDU session established
- `SM_PDU_PENDING` - PDU session establishment in progress
- `SM_PDU_ACTIVE` - PDU session active, user plane available
- `SM_MODIFYING` - PDU session modification in progress
- `SM_RELEASING` - PDU session release in progress

**Key Events:**
- `PduSessionInitEvent` - Start PDU session establishment
- `PduSessionAcceptEvent` - Session accepted by network
- `PduSessionRejectEvent` - Session rejected
- `ModificationEvent` - Modify session parameters
- `ReleaseEvent` - Release PDU session

---

## Concurrency Model

### Per-UE Goroutines

Each UE context runs in its own goroutine, providing isolation and preventing head-of-line blocking:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Per-UE Processing                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  UE-1 Goroutine         UE-2 Goroutine         UE-N Goroutine   │
│  ┌──────────────┐       ┌──────────────┐       ┌──────────────┐ │
│  │ 5GMM FSM     │       │ 5GMM FSM     │       │ 5GMM FSM     │ │
│  │ 5GSM FSM     │       │ 5GSM FSM     │       │ 5GSM FSM     │ │
│  │ Timers       │       │ Timers       │       │ Timers       │ │
│  │ Logger       │       │ Logger       │       │ Logger       │ │
│  └──────────────┘       └──────────────┘       └──────────────┘ │
│        │                      │                      │          │
│        ▼                      ▼                      ▼          │
│  [MmWorker Pool]       [SmWorker Pool]       [GnbWorker Pool]  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Event Dispatch Mechanism

```go
// FSM Event Flow (from internal/common/fsm/)
type FSM struct {
    CurrentState State
    transitions  map[State]map[EventType]Transition
    eventChan    chan Event
}

// Never call SendEvent from within callback - use SetNextEvent
func (f *FSM) SendEvent(event Event) {
    f.eventChan <- event  // Async dispatch to worker pool
}

func (f *FSM) SetNextEvent(event Event) {
    f.nextEvent = event   // Chain event for current callback
}
```

### Thread Safety Patterns

| Pattern | Usage | Location |
|---------|-------|----------|
| `sync.Map` | UE Pool, AMF Pool | `gnbcontext/` |
| `sync/atomic` | ID generation, counters | `uecontext/`, `gnbcontext/` |
| Channels | Event dispatch, RLink | `fsm/`, `rlink/` |
| Per-UE mutex | Non-hot paths | `uecontext/` |

**Critical Constraint**: NO mutexes in hot paths. Use channels for coordination.

---

## Module Reference

### UE Context (`internal/core/uecontext/`)

**Purpose**: Manages individual UE state, NAS handling, and FSM dispatch.

**Key Components**:
- `UeContext` - Per-UE state container
- `statemachine_5gmm.go` - 5GMM FSM definition
- `statemachine_5gsm.go` - 5GSM FSM definition
- `handle_n1sm.go` - NAS message handlers
- `oam.go` - OAM remote control interface
- `timer_manager.go` - NAS timer management

**Critical Files**:
```
uecontext/
├── ue_context.go          # UeContext struct definition
├── statemachine_5gmm.go   # 5GMM state transitions
├── statemachine_5gsm.go   # 5GSM state transitions
├── handle_n1sm.go         # NAS-SM message handling
├── handle_n1mm.go         # NAS-MM message handling
├── timer_manager.go       # Timer T3510, T3511, etc.
├── oam.go                 # Remote control API
└── triggers.go            # Event trigger functions
```

### gNB Context (`internal/core/gnbcontext/`)

**Purpose**: Manages gNodeB state, NGAP handling, and UE/AMF pools.

**Key Components**:
- `GnbContext` - gNB state container
- `dispatch.go` - NGAP message routing
- `ngap_handler.go` - NGAP procedure handlers
- `oam.go` - OAM remote control interface

**Critical Files**:
```
gnbcontext/
├── gnb_context.go         # GnbContext struct definition
├── dispatch.go            # NGAP message dispatcher
├── ngap_handler.go        # NGAP procedure handlers
├── amf_pool.go            # AMF connection management
├── ue_pool.go             # UE context storage
└── oam.go                 # Remote control API
```

### FSM Framework (`internal/common/fsm/`)

**Purpose**: Generic async finite state machine with event-driven transitions.

**Key Features**:
- Async event processing via channels
- Configurable transitions with callbacks
- Entry/exit actions
- Event chaining via `SetNextEvent`

**Usage Pattern**:
```go
fsm := fsm.NewFSM(
    initialState,
    fsm.Config{
        "StateA": {
            EventA: {To: "StateB", Action: onEventA},
        },
        "StateB": {
            EventB: {To: "StateA", Action: onEventB},
        },
    },
)
fsm.SendEvent(event)  // Async dispatch
```

### Worker Pools (`internal/common/pool/`)

**Purpose**: Manage goroutine pools for different event types.

**Pool Types**:
| Pool | Purpose | Worker Type |
|------|---------|-------------|
| MmWorkerPool | 5GMM events | MmWorker |
| SmWorkerPool | 5GSM events | SmWorker |
| GnbWorkerPool | NGAP events | GnbWorker |
| SctpWorkerPool | SCTP I/O | SctpWorker |

**Configuration**:
```yaml
# Pool sizes configured in config.yml
workerPools:
  mmWorkers: 4
  smWorkers: 4
  gnbWorkers: 2
  sctpWorkers: 2
```

### Logger (`internal/common/logger/`)

**Purpose**: High-performance ring buffer logging for concurrent UE/gNB logging.

**Features**:
- Per-UE log buffers (configurable size)
- Delay tracking integration
- Non-blocking writes
- Memory-efficient for high UE counts

### Stats (`internal/common/stats/`)

**Purpose**: Collect and aggregate procedure statistics.

**Metrics Collected**:
- Registration success/failure rates
- PDU session establishment times
- Handover success rates
- Per-procedure latency distribution

---

## Transport Layer

### RLink - Virtual Radio (`internal/transport/rlink/`)

**Purpose**: Simulates the radio interface between UE and gNB using Go channels.

**Architecture**:
```
┌─────────────────────────────────────────────────────────────┐
│                       RLink Manager                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌─────────────┐      ┌─────────────┐                     │
│   │  UE Tx Chan │─────►│  gNB Rx Chan│                     │
│   │  (per UE)   │      │  (per gNB)  │                     │
│   └─────────────┘      └─────────────┘                     │
│                                                              │
│   ┌─────────────┐      ┌─────────────┐                     │
│   │ gNB Tx Chan │─────►│  UE Rx Chan │                     │
│   │  (per gNB)  │      │  (per UE)   │                     │
│   └─────────────┘      └─────────────┘                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Message Types** (see `pkg/model/rrc.go`):
- `RrcSetupRequest` / `RrcSetup` / `RrcSetupComplete`
- `RrcReconfiguration`
- `RrcRelease`
- `HandoverCommand` / `HandoverComplete`

### SCTP/NGAP (`internal/transport/sctpngap/`)

**Purpose**: Handle SCTP connections and NGAP message encoding/decoding.

**Key Components**:
- SCTP connection management (N2 interface)
- NGAP message serialization
- AMF endpoint handling

**NGAP Procedures**:
- NG Setup
- Initial UE Message
- Uplink/Downlink NAS Transport
- PDU Session Resource Setup/Release
- Handover Preparation/Resource Allocation

---

## Observability

### Delay Tracking

Built-in delay measurement at protocol boundaries:

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ NAS Tx  │───►│ NGAP Tx │───►│ SCTP Tx │───►│ AMF Rx  │
│ t0      │    │ t1      │    │ t2      │    │ t3      │
└─────────┘    └─────────┘    └─────────┘    └─────────┘

Delay Metrics:
- NAS→NGAP: t1 - t0
- NGAP→SCTP: t2 - t1
- SCTP→AMF: t3 - t2
- End-to-end: t3 - t0
```

### OAM Backend (`monitoring/oambackend/`)

REST API endpoints for real-time control and monitoring:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/ues` | GET | List all UEs |
| `/api/ues/{id}` | GET | Get UE details |
| `/api/ues/{id}/trigger` | POST | Trigger UE event |
| `/api/gnbs` | GET | List all gNBs |
| `/api/stats` | GET | Get procedure statistics |
| `/api/delays` | GET | Get delay measurements |

### GTP-U Management (`monitoring/gtp5g/`)

Kernel-level GTP-U tunnel management for user plane:

- `gogtp5g-link` - Manage GTP interfaces
- `gogtp5g-tunnel` - Manage GTP tunnels (PDU sessions)

---

## Anti-Patterns & Technical Debt

### Known Issues

| Issue | Location | Impact | Status |
|-------|----------|--------|--------|
| Race condition in ID generation | `getRanUeId()`, `getUeTeid()` | Potential ID collisions under load | **Known** |
| Excessive `Fatal()` calls | Protocol handlers | Process crash instead of recovery | **Known** |
| Recursive FSM events | - | Prevented by constraint | **Mitigated** |

### Critical Constraints (NEVER violate)

1. **No recursive FSM events**: Never call `SendEvent()` from within an FSM callback; use `SetNextEvent()` instead.

2. **No `sync.Pool` for SCTP reads**: Causes data corruption across worker goroutines.

3. **No mutexes in hot paths**: Use `sync/atomic` for ID generation and counters; use channels for coordination.

4. **No type error suppression**: Never use `as any`, `@ts-ignore` equivalent patterns.

### Error Handling Pattern

```go
// AVOID: Fatal on protocol errors
func handleNgapMessage(msg []byte) {
    if err := decode(msg); err != nil {
        log.Fatal("decode failed")  // BAD: Crashes process
    }
}

// PREFER: Graceful error handling
func handleNgapMessage(msg []byte) error {
    if err := decode(msg); err != nil {
        log.Errorf("decode failed: %v", err)
        return err  // GOOD: Let caller handle
    }
    return nil
}
```

---

## References

- [3GPP TS 24.501](https://www.3gpp.org/DynaReport/24501.htm) - NAS protocol
- [3GPP TS 38.413](https://www.3gpp.org/DynaReport/38413.htm) - NGAP protocol
- [AGENTS.md](../AGENTS.md) - Project knowledge base
- [CONFIGURATION.md](./CONFIGURATION.md) - Configuration guide
- [FEATURES.md](./FEATURES.md) - Supported 5G procedures