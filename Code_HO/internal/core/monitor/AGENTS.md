<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-07 | Updated: 2026-04-07 -->

# Monitor

## Purpose
Handover monitoring and coordination for multi-gNB scenarios, particularly NTN (Non-Terrestrial Network / satellite) handover management. Monitors UE measurements, coordinates Xn/N2 handovers, and manages network conditions across ground and NTN gNBs.

## Key Files

| File | Description |
|------|-------------|
| `manager.go` | `HandoverMonitor` struct - main handover management |
| `fsm_ho.go` | `HoProcedure` FSM - Xn Handover procedure state machine |
| `gnb_group.go` | `GnbGroup` - groups gNBs by type (Ground/NTN) with network conditions |
| `handover_queue.go` | `HandoverQueue` - priority queue for scheduled handovers |
| `handover_tracker.go` | `MonitorHandoverTracker` - aggregate handover statistics |
| `ntn_coordinator.go` | `NTNCoordinator` - satellite visibility and HO window prediction |
| `oam.go` | REST API endpoints for stats/queue/visibility |

## Subdirectories

None.

## For AI Agents

### Working In This Package
- **FSM-based HO**: `HoProcedure` implements Xn Handover FSM
- **NTN Support**: Coordinates handovers between ground and NTN (satellite) gNBs
- **Network Conditions**: Applies packet loss, latency, jitter per gNB group

### Xn Handover Flow (Actual Implementation)

```
1. TriggerXnHandover()
   ↓
2. sgnb → tgnb: RLinkHandoverForwardUeContext (UE context transfer)
3. sgnb → UE:    RLinkHandoverPrepareRequest (HO command)
   ↓
4. UE connects to tgnb via new RLink connection
   ↓
5. tgnb: handler RLinkHandoverForwardUeContext
          → sendPathSwitchRequest() → AMF (NGAP PathSwitchRequest)
   ↓
6. AMF → tgnb: PathSwitchRequestAcknowledge
   ↓
7. tgnb: ueHoStatusPool[prUeId] = true (HO success flag)
   ↓
8. UE → sgnb: RlinkRlinkHandoverPrepareResponse
   ↓
9. sgnb: delete UE context (UE state = UE_DOWN)
```

### HO FSM States

| State | Description |
|-------|-------------|
| `MonitorNull` | Monitoring mode - waiting for HO decision |
| `MonitorPrepare` | Sending RLinkHandoverForwardUeContext + RLinkHandoverPrepareRequest |
| `MonitorExecute` | PathSwitchRequest sent, waiting for Ack |
| `MonitorComplete` | PathSwitch completed, waiting for final confirmation |
| `MonitorHoSuccess` | HO completed successfully |
| `MonitorHoFail` | HO failed |

### HO FSM Events (Mapped to Actual Messages)

| Event | Mapped To | Description |
|-------|-----------|-------------|
| `HoDecisionEvent` | `TriggerXnHandover()` | Start handover |
| `HoXnForwardUeContextEvent` | `RLinkHandoverForwardUeContext` | UE context transfer to tgnb |
| `HoRlinkPrepareReqEvent` | `RLinkHandoverPrepareRequest` | HO command to UE |
| `HoUeConnectedEvent` | UE connects to tgnb | UE successfully connected |
| `HoPathSwitchReqEvent` | `PathSwitchRequest` (NGAP) | Path switch request to AMF |
| `HoPathSwitchAckEvent` | `PathSwitchRequestAcknowledge` | Path switch acknowledged |
| `HoPathSwitchFailEvent` | `PathSwitchRequestFailure` | Path switch failed |
| `HoRlinkPrepareRespEvent` | `RlinkRlinkHandoverPrepareResponse` | UE response to sgnb |
| `HoSuccessEvent` | UE context deleted | HO fully complete |
| `HoFailEvent` | Error occurred | HO failed |

### Data Structures

**`HandoverMonitor`** (main coordinator):
```go
type HandoverMonitor struct {
    gnbs            sync.Map              // All registered gNBs
    gnbGroups       map[string]*GnbGroup // Grouped by name
    handoverQueue   *HandoverQueue        // Scheduled handovers
    ntnCoordinator  *NTNCoordinator       // Satellite visibility
    tracker         *MonitorHandoverTracker
    eventChan       chan HandoverEvent    // Immediate handover events
}
```

**`HoProcedure`** (per-UE HO FSM):
```go
type HoProcedure struct {
    fsmHo          *fsm.Fsm
    stateHo        *fsm.State
    prUeId         int64
    sourceGnbId    string
    targetGnbId    string
    success        bool
    failureReason  string
}
```

**`GnbGroup`** (gNB grouping):
```go
type GnbGroup struct {
    Name             string
    Type             GnbGroupType  // Ground or NTN
    Gnbs             []*gnbcontext.GnbContext
    NetworkCondition *NetworkCondition // Packet loss, latency, jitter
}
```

### Key Methods

```go
// HandoverMonitor
func (hm *HandoverMonitor) RegisterGnb(gnb *gnbcontext.GnbContext, groupType GnbGroupType, groupName string)
func (hm *HandoverMonitor) TriggerImmediateHandover(prUeId int64, sourceGnbId, targetGnbId string, handoverType int) error
func (hm *HandoverMonitor) ScheduleHandover(prUeId int64, sourceGnbId, targetGnbId string, handoverType int, executeAt time.Time) error

// HoProcedure
func (ho *HoProcedure) StartHo()
func (ho *HoProcedure) SendEvent(eventType model.EventType, data *HoEventData)
func (ho *HoProcedure) GetState() model.StateType
func (ho *HoProcedure) IsSuccess() bool
func (ho *HoProcedure) GetFailureReason() string
```

### UE HO States

UE has a simple HO state managed via `state_ho` field:
- `HoIdle` - Not in handover
- `HoStart` - Handover in progress
- `HoSuccess` - Handover completed successfully
- `HoFail` - Handover failed

```go
// UeContext methods
func (ue *UeContext) SetHoState(state model.StateType)
func (ue *UeContext) GetHoState() model.StateType
```

## Dependencies

### Internal
- `stormsim/internal/core/gnbcontext` - GnbContext, GnbUeContext for gNB and UE references
- `stormsim/internal/common/fsm` - FSM framework for HO procedure
- `stormsim/internal/common/pool` - Worker pools for async processing
- `stormsim/pkg/model` - State/Event types for HO FSM

### External
None currently.

## TODO
- [ ] Integrate HoProcedure with actual gNB message handlers
- [ ] Connect UE HO state updates from FSM callbacks
- [ ] Add timer handling for HO phases (T1, T2, T3)
- [ ] Add N2 Handover support
