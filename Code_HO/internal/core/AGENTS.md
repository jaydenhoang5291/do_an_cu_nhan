<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Core

<!-- Parent: ../AGENTS.md -->
<!-- Date: 2026-03-26 -->

## Purpose
Core UE and gNB context implementations implementing 5GMM/5GSM state machines per 5G NAS/NAS/n1 ( N1/N N1/GNB ↔ UE, messages), and 3GPP authentication, security context management, and route GTP tunnel interfaces setup for data plane traffic.

 Contains **`gnbcontext/`**, **`uecontext/`**, and **`managers/` (incomplete).

 Contains **Uecontext**** (UE context, 5GMM/5GSM FSMs, NAS message handlers, timers, security
 and GTP tunnel management.

 Contains gnbcontext-related files that define the gNB context, NGAP handlers, and AMF pool management.
 Contains **`gnbcontext/context.go`** GnbContext struct and main context definition
- **`ue.go`** UeContext struct, PDU session management
- **`gnb.go`** GnbUeContext struct for gNB-side UE context (PDU sessions, AMF connections, NGAP message handling)
- **`trigger.go`** Event triggers for registration, deregistration, PDU sessions, idle/service transitions
- **`ue.go`** UeContext - Main UE context struct with buffered logging
- **`pool.go`** UE pool initialization and FSM setup
- **`sec/`** Security package (milenage, SQN management, security context)
- **`timer/`** Timer engine and timer types (T3510, T3540, etc.)
- **`gtp.go`** GTP tunnel interface setup using gogtp5g and
- **`replay.go`** Message capture/replay for fuzzing
- **`managers/context.go`** Incomplete manager stub

- **`utils.go`** Helper functions for UE context

- **`oam.go`** OAM API endpoints for UE/gNB stats
- **`service.go`** UE/gNB service routines (connect, listen, handle messages)
- **`trigger.go`** Event trigger functions
- **`auth.go`** Authentication context with milenage processing
- **`session.go`** PDU session management
- **`handle_n1mm.go`** 5GMM NAS message handlers
- **`handle_n1sm.go`** 5GSM NAS message handlers
- **`statemachine_5gmm.go`** 5GMM FSM definition
- **`statemachine_5gsm.go`** 5GSM FSM definition

- **`timer/engine.go`** Timer engine implementation
- **`timer/timer.go`** Timer type and durations
- **`sec/secctx.go`** Security context management
- **`sec/ueauth.go`** 5G A key derivation functions (KDF)
- **`sec/milenage.go`** Milenage algorithm for authentication
- **`sec/sqn.go`** Sequence number management
- **`gtp.go`** GTP tunnel interface setup
- **`oam.go`** OAM API endpoints
- **`service.go`** Service routines (connect to gNB)
- **`gnb.go`** gNB connection management

- **`sender.go`** Message sending (UE←GNB, AMF↡5G)

- **`ngap_handler.go`** NGAP message handlers (downlink NAS, Initial Context Setup, PDU Session Resource Setup, Paging, Handover, AMF Configuration Update, etc.)
- **`ngap_send.go`** NGAP message construction and sending
- **`ngap_transfer.go`** NGAP transfer helper functions
- **`ngap_cause.go`** NGAP cause code mapping
- **`trigger.go`** Event triggers for registration/d deregistration/PDU session
- **`dispatch.go`** NGAP message dispatcher and delay tracking
- **`service.go`** gNB service routines (SCTP connection,, NGAP handling)
 rlink handling)
- **`ue.go`** UE connection management
- **`init.go`** gNB initialization and global storage
- **`amf.go`** AMF context and pool management
- **`ue.go`** GnbUeContext struct for gNB-side UE state

- **`context.go`** GnbUeContext with core context data
- **`managers/context.go`** Incomplete manager stub

- **`oam.go`** OAM API endpoints for statistics retrieval
- **`replay.go`** Message capture/replay for testing
- **`task.go`** Task tracking
- **`utils.go`** Utility functions (cause codes, SNN derivation)
- **`gnb.go`** GNB connection handling
- **`pool.go`** Pool initialization
- **`task.go`** Task tracking
- **`oam.go`** OAM API endpoints
- **`oam.go`** OAM API endpoints
- **`ngap_cause.go`** NGAP cause code mapping
- **`ngap_transfer.go`** NGAP transfer helpers (PDU session QoS, slice info)
- **`trigger.go`** Event triggers (Xn/N2 handover)
- **`dispatch.go`** NGAP message dispatcher with delay tracking
- **`sender.go`** Message sending to UE, GNB, AMF
- **`service.go`** Service routines (UE←GNB, GNB→AMF)
- **`init.go`** Initialization and global GNB storage

- **`amf.go`** AMF context and pool management
- **`ue.go`** GnbUeContext struct
- **`context.go`** GnbContext struct
- **`ngap_handler.go`** NGAP message handlers
- **`ngap_send.go`** NGAP message construction
- **`ngap_transfer.go`** NGAP transfer helpers
- **`ngap_cause.go`** NGAP cause code mapping
- **`dispatch.go`** NGAP message dispatcher
- **`sender.go`** Message sending
- **`service.go`** Service routines (UE←GNB, GNB→AMF)
- **`trigger.go`** Event triggers (Xn/N2 handover)
- **`oam.go`** OAM API endpoints
- **`oam.go`** OAM API endpoints
- **`pool.go`** Pool initialization
- **`replay.go`** Message capture/replay
- **`task.go`** Task tracking
- **`utils.go`** Utility functions
- **`gnb.go`** GNB connection handling

- **`sec/`** Security package (milenage, SQN, security context)
- **`timer/`** Timer package (timer engine, timer types)
- **`managers/`** Management structures (incomplete)

- **`gnbcontext/`**: gNB context implementation with NGAP handlers, AMF pool, and PDU session management
- **`uecontext/`**: UE context implementation with 5GMM/5GSM FSMs, NAS handlers, timer engine, security, and GTP tunnel setup
- **`managers/`**: Incomplete management stubs
- **`internal/common/fsm``**: FSM framework
- **`internal/common/pool``**: Worker pools (- **`internal/common/logger``**: Ring buffer logging
- **`internal/common/stats```: Procedure statistics
- **`internal/transport/rlink``**: Virtual radio link (UE↔gNB)
- **`internal/transport/sctpngap``**: SCTP/NGAP transport
- **`pkg/config``**: Configuration structures
- **`pkg/model``**: 3GPP data models

- **`github.com/alitto/pond/v2``**: Worker pool management
- **`github.com/lvdund/ngap``**: NGAP encoding/decoding
- **`github.com/reogac/nas``**: NAS message handling
- **`github.com/reogac/utils/sec5g`**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U user plane
- **`monitoring/gtp5g``**: Kernel GTP tunnel management

- **`stormsim/pkg/config``**: UE/gNB configurations

- **`stormsim/pkg/model``**: Data models,- **`github.com/reogac/nas``**: NAS encoding/decoding
- **`github.com/lvdund/ngap``**: NGAP encoding/decoding
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security key derivation

- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding

- **`github.com/reogac/nas``**: NAS library for message encoding/decoding
- **`github.com/reogac/utils/sec5g``**: 5G security utilities
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding

- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-ggt/gtpv1`**: GTP-U
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp```: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas```: NAS library
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp```: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1`**: GTP-U
- **`github.com/reogac/utils/sec5g```: Security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas```: NAS library
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U
- **`github.com/reogac/utils/sec5g```: Security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas```: NAS library
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2```: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U
- **`github.com/reogac/utils/sec5g```: Security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas/`**: NAS library
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp```: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U
- **`github.com/reogac/utils/sec5g``**: Security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink```: Network package

- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp```: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management

- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp```: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/lvdund/ngap/aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS library
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink```: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com//control/go fish`**: SCTP transport library
- **`github.com/ishidawataru/sctp`**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U user plane
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/lvdund/ngap`aper``**: ASN.1 PER encoding
- **`github.com/reogac/nas``**: NAS message handling
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U user plane
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas```: NAS message handling
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvanada/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog```: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U user plane
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas/``: NAS message handling
- **`github.com/reogac/utils/sec5g``**: 5G security functions
- **`github.com/vishvananda/netlink``**: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``**: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``**: GTP-U user plane
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas```: NAS message handling
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink```: Network interface management
- **`github.com/alitto/pond/v2``**: Worker pools
- **`github.com/rs/zerolog``**: Structured logging
- **`github.com/ishidawataru/sctp``: SCTP transport
- **`github.com/wmnsk/go-gtp/gtpv1``: GTP-U user plane
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/lvdund/ngap/aper```: ASN.1 PER encoding
- **`github.com/reogac/nas```: NAS message handling
- **`github.com/reogac/utils/sec5g```: 5G security functions
- **`github.com/vishvananda/netlink```: Network interface management
- **`github.com/alitto/```

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# Core

<!-- Parent: ../AGENTS.md -->
<!-- Date: 2026-03-26 -->

## Purpose
Core UE and gNB context implementations implementing 5GMM/5GSM state machines, 5G NAS message handlers, timers, security, 5G AKA authentication, and GTP tunnel interface setup for data plane traffic.

 Contains **`gnbcontext/`** (gNB context, NGAP handlers, AMF pool management), PDU session management) and **`uecontext/`** (UE context, 5GMM/5GSM FSMs, NAS handlers, timers, security, GTP tunnel setup).

 Contains **`managers/`** (incomplete management stubs).

 Contains **`uecontext/sec/`** (security context, milenage, SQN management, 5G AKA authentication) and **`uecontext/timer/`** (Timer engine for NAS timers T3510, T3540, etc.).
 Contains `monitoring/gtp5g/` (Kernel GTP tunnel management)
 Contains `internal/transport/rlink/` (Virtual radio link UE↔gNB)
 Contains `internal/transport/sctpngap/` (SCTP/NGAP transport)
 Contains `internal/common/fsm/` (Async FSM framework)
 Contains `internal/common/pool/` (Worker pools)
 Contains `internal/common/logger/` (Ring buffer logging)
 Contains `internal/common/stats/` (Procedure statistics)
 Contains `pkg/config/` (UE/gNB configurations)
 Contains `pkg/model/` (3GPP data models)
 Contains `github.com/alitto/pond/v2` (Worker pool management)
 Contains `github.com/reogac/nas`` (NAS message handling)
 Contains `github.com/reogac/utils/sec5g` (5G security functions)
 Contains `github.com/lvdund/ngap`aper``, `github.com/lvdund/ngap`ies` (NGAP encoding/decoding)
 Contains `github.com/vishvananda/netlink` (Network interface management)
 Contains `github.com/wmnsk/go-gtp/gtpv1` (GTP-U user plane)

## Key Files

| File | Description |
|------|-------------|
| `gnbcontext/context.go` | GnbContext struct, AMF/UE pools, slice config, ID generators |
| `gnbcontext/init.go` | gNB initialization, global GNB storage |
| `gnbcontext/amf.go` | GnbAmfContext struct, PLMN/slice support lists, AMF state |
| `gnbcontext/ue.go` | GnbUeContext struct, PDU session management, core context |
| `gnbcontext/ngap_handler.go` | NGAP message handlers (downlink, context setup, PDU session, handover, paging) |
| `gnbcontext/ngap_send.go` | NGAP message construction (InitialUE, UplinkNAS, responses) |
| `gnbcontext/ngap_transfer.go` | NGAP transfer helper functions (PDU session resource IE) |
| `gnbcontext/ngap_cause.go` | NGAP cause code mapping (causeToString) |
| `gnbcontext/dispatch.go` | NGAP message dispatcher with delay tracking |
| `gnbcontext/sender.go` | Message sending (UE↔gNB, AMF, SendToGnb) |
| `gnbcontext/service.go` | gNB service routines (SCTP connection, rlink handling, listenToUE) |
| `gnbcontext/trigger.go` | gNB event triggers (Xn/N2 handover, path switch) |
| `gnbcontext/oam.go` | OAM API endpoints (GetGnbById, GetGnbUes, GetStats) |
| `uecontext/ue.go` | UeContext struct, PDU session management, security, timers |
| `uecontext/session.go` | PduSession struct, state machine, GTP tunnel info |
| `uecontext/pool.go` | UE pool initialization, FSM setup |
| `uecontext/task.go` | Task tracking with timing and state management |
| `uecontext/statemachine_5gmm.go` | 5GMM FSM transitions, callbacks, statistics |
| `uecontext/statemachine_5gsm.go` | 5GSM FSM transitions, callbacks |
| `uecontext/handle_n1mm.go` | NAS 5GMM message handlers (registration, auth, security mode, deregistration) |
| `uecontext/handle_n1sm.go` | NAS 5GSM message handlers (PDU session establish/release) |
| `uecontext/trigger.go` | UE event triggers (registration, deregistration, PDU session, service request) |
| `uecontext/auth.go` | AuthContext struct, authentication processing (RES*, KAMF derivation) |
| `uecontext/sec/secctx.go` | SecurityContext struct, NAS key AS key derivation |
| `uecontext/sec/ueauth.go` | 5G AKA key derivation functions (KDF, KAUSF, KSEAF, KAMF) |
| `uecontext/sec/milenage.go` | Milenage algorithm implementation (F1-F5) |
| `uecontext/sec/sqn.go` | Sequence number management for authentication |
| `uecontext/timer/timer.go` | Timer types and durations (T3346-T3540) |
| `uecontext/timer/engine.go` | TimerEngine with concurrent timer management |
| `uecontext/service.go` | UE service routines (connect to gNB, listen to gNB, verifyPaging) |
| `uecontext/gnb.go` | GNB message handling (GTP setup, NAS send, N1SM send) |
| `uecontext/gtp.go` | GTP tunnel interface setup (netlink, gogtp5g-link, gogtp5g-tunnel) |
| `uecontext/oam.go` | OAM API endpoints (GetUeById, GetStatistics) |
| `uecontext/utils.go` | Utility functions (cause codes, SNN derivation) |
| `uecontext/replay.go` | Message capture/replay for testing |
| `managers/context.go` | Incomplete manager stubs (RlinkGroup, Manger) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `gnbcontext/` | gNB context, NGAP handlers, AMF pool management |
| `uecontext/` | UE context, 5GMM/5GSM FSMs, NAS handlers, timers, security, GTP tunnel setup |
| `uecontext/sec/` | Security context, milenage authentication, SQN management |
| `uecontext/timer/` | Timer engine for NAS timers (T3510, T3540, etc.) |
| `managers/` | Management structures (incomplete) |

## For AI Agents

### Working In This Directory
- **Concurrency**: Use `sync/atomic` for ID generation and counters. Use channels for coordination. NO mutexes in hot paths.
- **Memory Safety**: Do NOT use `sync.Pool` for SCTP message reads to avoid data corruption across worker goroutines.
- **FSM Flow**: NEVER call `SendEvent` from within an FSM callback. Use `SetNextEvent` instead to avoid recursive events.
- **Worker Pools**: All FSM events are processed via worker pools (MmWorkerPool, SmWorkerPool, GnbWorkerPool).
- **Logging**: Use `BufferedLogger` for UE/gNB contexts to enable ring buffer log retrieval via OAM API.
- **Global Storage**: `gnbcontext.Gnbs` (`sync.Map`) stores all gNB contexts by gNB ID.

### Testing Requirements
```bash
# Run all tests in core/
go test ./internal/core/...

# Run security tests
go test ./internal/core/uecontext/sec/...
```

### Common Patterns
**FSM Event Handling** (never call SendEvent from callback):
```go
// In callback - use SetNextEvent for chaining
func someCallback(state *fsm.State, event *fsm.EventData) {
    // ... process event ...
    state.SetNextEvent(nextEvent) // Queue next event, don't call SendEvent
}
```

**Atomic ID Generation**:
```go
func (gnb *GnbContext) getRanUeId() int64 {
    return atomic.AddInt64(&gnb.ranUeIdGenerator, 1) - 1
}
```

**RLink Connection (UE↔gNB)**:
```go
conn := rlink.NewConnection(ueID, msin, gnbID, bufferSize, timeout)
conn.SendUplink(msg)    // UE → GNB
conn.SendDownlink(msg)  // GNB → UE
```

**5GMM State Machine**:
```go
// Transitions defined in statemachine_5gmm.go
Deregistered + InitRegistrationRequestEvent → RegisteredInitiated
RegisteredInitiated + RegistrationAcceptEvent → Registered
Registered + InitDeregistrationRequestEvent → Deregistered
```

**5GSM State Machine**:
```go
// Transitions defined in statemachine_5gsm.go
PDUSessionInactive + InitPduSessionEstablishmentRequestEvent → PDUSessionActivePending
PDUSessionActivePending + EstablishmentAccept → PDUSessionActive
PDUSessionActive + ReleaseRequest → PDUSessionInactivePending
```

### Key Data Structures
- **`gnbcontext.GnbContext`**: Main gNB context with pools for AMF/UE management
  - `amfPool sync.Map` - AMF contexts by AMF ID
  - `ranUePool sync.Map` - UE contexts by RAN UE NGAP ID
  - `msinPool sync.Map` - UE contexts by MSIN
  - `prUeIdPool sync.Map` - UE contexts by internal UE ID
  - `downlinkTeidPool sync.Map` - UE contexts by downlink TEID

- **`uecontext.UeContext`**: Main UE context with 5GMM/5GSM state machines
  - `state_mm *fsm.State` - 5GMM state machine
  - `sessions [16]*PduSession` - PDU sessions (max 15, index 0 reserved)
  - `auth AuthContext` - Authentication context
  - `secCtx *sec.SecurityContext` - Security context

- **`uecontext.PduSession`**: PDU session with 5GSM state machine
  - `state_sm *fsm.State` - 5GSM state machine
  - `gnbPduSession *model.GnbPDUSessionContext` - GNB-side session info

- **`uecontext.AuthContext`**: Authentication context
  - `milenage *sec.Milenage` - Milenage algorithm
  - `sqn sec.Sqn` - Sequence number
  - `kamf []byte` - KAMF key

- **`uecontext.sec.SecurityContext`**: NAS security context
  - `gppNas *nas.NasContext` - 3GPP NAS context
  - `kamf []byte` - KAMF key
  - `kgnb []uint8` - gNB key

- **`uecontext.timer.TimerEngine`**: Timer management
  - `timers map[TimerType]*Timer` - Timer storage
  - `BlockTimerEvent bool` - Flag to block timer events

- **`gnbcontext.GnbAmfContext`**: AMF context
  - `tlnaAssoc TNLAssociation` - SCTP association
  - `supportedPlmnList *PlmnSupported` - Supported PLMNs
  - `supportedSliceList *SliceSupported` - Supported slices

- **`gnbcontext.GnbUeContext`**: GNB-side UE context
  - `context model.UeCoreContext` - Core context with PDU sessions
  - `rlinkConn *rlink.Connection` - RLink connection to UE

### Message Flow
1. **UE Registration**: `triggerInitRegistration()` → `sendNas()` → RLink → GNB → `buildInitialUeMessage()` → SCTP → AMF
2. **NAS Downlink**: AMF → SCTP → GNB `dispatch()` → `handlerDownlinkNasTransport()` → RLink → UE `handleNas_n1mm()`
3. **PDU Session**: `triggerInitPduSessionRequest()` → `sendN1Sm()` → RLink → GNB → `buildUplinkNasTransport()` → SCTP → AMF
4. **GTP Setup**: `setupGtpInterface()` creates kernel GTP interface using netlink and gogtp5g tools

### Important Notes
- **Fatal() Calls**: Many protocol handlers use `Fatal()` instead of graceful recovery. Consider refactoring to return errors.
- **Handover**: Xn and N2 handover are supported via `trigger.go` and `ngap_handler.go`
- **Fuzzing**: Test framework for random event injection via `enableFuzz` flag and `Capture` in `replay.go`

<!-- MANUAL: Add notes about specific core implementation details here -->
