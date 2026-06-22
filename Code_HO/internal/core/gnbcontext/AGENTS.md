<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-03-26 | Updated: 2026-03-26 -->

# gnbcontext

## Purpose
gNB (gNodeB) context implementation managing SCTP/NGAP connections to AMF(s), UE pool management, PDU session handling, and handover procedures. Acts as the control plane bridge between UEs and the 5G Core, implementing NGAP protocol handlers for downlink messages and constructing uplink NGAP messages. Supports multiple AMF connections with capacity-based selection and both Xn and N2 handover scenarios.

## Key Files
| File | Description |
|------|-------------|
| `context.go` | `GnbContext` struct with AMF/UE pools, slice config, atomic ID generators, delay tracking |
| `init.go` | `InitGnb()` initialization, global `Gnbs` sync.Map storage |
| `amf.go` | `GnbAmfContext` struct, AMF states, PLMN/slice support lists, TNLAssociation |
| `ue.go` | `GnbUeContext` struct, UE states, PDU session CRUD, context operations |
| `ngap_handler.go` | NGAP message handlers (downlink NAS, context setup, PDU session, handover, paging, AMF config update) |
| `ngap_send.go` | NGAP message construction (InitialUE, UplinkNAS, responses, handover acknowledge) |
| `ngap_transfer.go` | NGAP transfer IE builders (PDU session resource, handover containers) |
| `ngap_cause.go` | NGAP cause code to string mapping for all cause types |
| `dispatch.go` | NGAP message dispatcher with delay tracking via `DelayTracker` |
| `service.go` | Service routines (SCTP connection, rlink handling, UE message processing) |
| `trigger.go` | Event triggers (Xn/N2 handover, path switch, UE release, NG setup) |
| `sender.go` | Message sending functions (`SendToGnb`, `sendNasToUe`, `sendNgap`) |
| `oam.go` | `RemoteGnbApi` for OAM endpoints (logs, info, UE list, delay stats) |

## For AI Agents

### Working In This Directory
- **Global Storage**: `Gnbs` sync.Map stores all gNB contexts by gNB ID. Use `Gnbs.Load(gnbId)` to retrieve.
- **Atomic IDs**: Use `getRanUeId()`, `getRanAmfId()`, `getUeTeid()` for atomic ID generation. Never use mutexes for counters.
- **UE Pools**: Four concurrent-safe pools for UE lookup:
  - `ranUePool` (RAN UE NGAP ID) - primary for NGAP messages
  - `msinPool` (MSIN string) - for UE identification
  - `prUeIdPool` (internal PrUeId) - for internal routing
  - `downlinkTeidPool` (downlink TEID) - for GTP-U correlation
- **Worker Pool**: All NGAP handlers are submitted to `pool.GnbWorkerPool` for async processing.
- **Delay Tracking**: Use `LogNgapReceive()`/`LogNgapSend()` for NGAP delay measurement.

### Common Patterns

**Get UE by different keys:**
```go
// By RAN UE NGAP ID (from NGAP messages)
ue, err := gnb.getGnbUe(ranUeId)

// By internal PrUeId (from rlink)
ue, err := gnb.getGnbUeByPrUeId(prUeId)

// By MSIN (from OAM/triggers)
ue, err := gnb.getGnbUeByMsin(msin)
```

**Send NGAP message to AMF:**
```go
// From UE context
ngapPdu, _ := gnb.buildUplinkNasTransport(nasPdu, ue)
ue.sendNgap(ngapPdu)

// From AMF context
ngapPdu, _ := ngap.NgapEncode(&msg)
amf.sendNgap(ngapPdu)
```

**Send message to UE via RLink:**
```go
ue.sendNasToUe(nasPdu)  // NAS message
ue.sendMsgToUe(&model.RlinkSetupPduSessonCommand{...})  // Control message
```

**Dispatch NGAP message from SCTP:**
```go
// In sctpListen goroutine
for rawMsg := range amf.tlnaAssoc.sctpConn.Read() {
    pool.GnbWorkerPool.Submit(func() { gnb.dispatch(amf, rawMsg) })
}
```

**Trigger handover:**
```go
// Xn handover (direct gNB-to-gNB)
TriggerXnHandover(sourceGnb, targetGnb, prUeId)

// N2 handover (via AMF)
TriggerNgapHandover(sourceGnb, targetGnb, prUeId)
```

### Key Data Structures

**GnbContext** - Main gNB context:
- `dataPlaneInfo` / `controlPlaneInfo` - Network configuration
- `amfPool sync.Map` - AMF contexts by AMF ID
- `ranUePool sync.Map` - UE contexts by RAN UE NGAP ID
- `msinPool sync.Map` - UE contexts by MSIN
- `prUeIdPool sync.Map` - UE contexts by PrUeId
- `downlinkTeidPool sync.Map` - UE contexts by downlink TEID
- `delayTracker *logger.DelayTracker` - NGAP delay tracking

**GnbUeContext** - gNB-side UE context:
- `ranUeNgapId` / `amfUeNgapId` - NGAP identifiers
- `amfId` - Selected AMF
- `state` - UE state (UE_INITIALIZED, UE_ONGOING, UE_READY, UE_DOWN)
- `sctpConnection` - SCTP connection to AMF
- `rlinkConn` - RLink connection to UE
- `context model.UeCoreContext` - Core context with PDU sessions

**GnbAmfContext** - AMF context:
- `tlnaAssoc TNLAssociation` - SCTP association with weight factor
- `state` - AMF state (AMF_INACTIVE, AMF_ACTIVE, AMF_OVERLOADED)
- `supportedPlmnList` / `supportedSliceList` - Linked list of supported PLMNs/slices

### Message Flow
1. **NG Setup**: `sendNgSetupRequest()` → AMF → `handlerNgSetupResponse()` → AMF marked ACTIVE
2. **UE Registration**: UE → RLink → `gnbListen()` → `newGnBUe()` → `buildInitialUeMessage()` → AMF
3. **Downlink NAS**: AMF → SCTP → `dispatch()` → `handlerDownlinkNasTransport()` → `sendNasToUe()` → UE
4. **PDU Session**: AMF → `handlerPduSessionResourceSetupRequest()` → Create PDU session → `sendPduSessionResourceSetupResponse()`
5. **Xn Handover**: `TriggerXnHandover()` → `SendToGnb()` → `RLinkHandoverForwardUeContext` → `sendPathSwitchRequest()`

## Dependencies

### Internal
- `internal/common/logger` - BufferedLogger, DelayTracker
- `internal/common/pool` - GnbWorkerPool
- `internal/transport/rlink` - Virtual radio connection
- `internal/transport/sctpngap` - SCTP/NGAP transport
- `pkg/model` - Data models (RLink messages, PDU session context)

### External
- `github.com/lvdund/ngap` - NGAP encoding/decoding, IEs
- `github.com/lvdund/ngap/aper` - ASN.1 PER encoding
- `github.com/lvdund/ngap/ies` - NGAP Information Elements
- `github.com/lvdund/ngap/utils` - NGAP utilities (IP conversion)
- `github.com/reogac/nas` - NAS library (GUTI type)
- `github.com/wmnsk/go-gtp/gtpv1` - GTP-U (UPlaneConn type)
