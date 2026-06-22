package monitor

import (
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/logger"
	"stormsim/internal/common/pool"
	"stormsim/internal/core/gnbcontext"
	"stormsim/pkg/model"
	"time"
)

var hoLogger = logger.InitLogger("ho-fsm", nil)

type HoProcedure struct {
	fsmHo         *fsm.Fsm
	stateHo       *fsm.State
	prUeId        int64
	sourceGnbId   string
	targetGnbId   string
	success       bool
	failureReason string
	handoverType  int // 1=Xn, 2=N2

	loss   float64
	delay  time.Duration
	jitter time.Duration

	startTime time.Time
	endTime   time.Time
	duration  time.Duration
	finished  bool

	prepareStart  time.Time
	executeStart  time.Time
	completeStart time.Time
	failStart     time.Time

	ueStateCallback func(prUeId int64, state model.StateType)
}

type HoEventData struct {
	PrUeId        int64
	SourceGnbId   string
	TargetGnbId   string
	Success       bool
	FailureReason string
}

func NewHoProcedure(prUeId int64, sourceGnbId, targetGnbId string, handoverType int, loss float64, delay, jitter time.Duration) *HoProcedure {
	ho := &HoProcedure{
		prUeId:       prUeId,
		sourceGnbId:  sourceGnbId,
		targetGnbId:  targetGnbId,
		handoverType: handoverType,
		loss:         loss,
		delay:        delay,
		jitter:       jitter,
	}

	ho.stateHo = fsm.NewState(model.MonitorNull, ho)
	ho.fsmHo = fsm.NewFsm(ho.buildTransitions(), pool.GnbWorkerPool)

	hoLogger.Info("[UE-%d] HO FSM created: %s->%s, NULL", prUeId, sourceGnbId, targetGnbId)
	return ho
}

func (ho *HoProcedure) SetUeStateCallback(callback func(prUeId int64, state model.StateType)) {
	ho.ueStateCallback = callback
}

func (ho *HoProcedure) buildTransitions() fsm.Options {
	return fsm.Options{
		Transitions: fsm.Transitions{
			fsm.Tuple(model.MonitorNull, model.HoDecisionEvent):                model.MonitorPrepare,
			fsm.Tuple(model.MonitorPrepare, model.HoRlinkPrepareReqSentEvent):  model.MonitorExecute,
			fsm.Tuple(model.MonitorPrepare, model.HoFailEvent):                 model.MonitorHoFail,
			fsm.Tuple(model.MonitorExecute, model.HoPathSwitchRequestEvent):    model.MonitorComplete,
			fsm.Tuple(model.MonitorExecute, model.HoPathSwitchFailEvent):       model.MonitorHoFail,
			fsm.Tuple(model.MonitorExecute, model.HoFailEvent):                 model.MonitorHoFail,
			fsm.Tuple(model.MonitorComplete, model.HoRlinkSetupPduSessonEvent): model.MonitorHoSuccess,
			fsm.Tuple(model.MonitorComplete, model.HoFailEvent):                model.MonitorHoFail,
		},
		Callbacks: fsm.Callbacks{
			model.MonitorNull:      ho.onNull,
			model.MonitorPrepare:   ho.onPrepare,
			model.MonitorExecute:   ho.onExecute,
			model.MonitorComplete:  ho.onComplete,
			model.MonitorHoSuccess: ho.onHoSuccess,
			model.MonitorHoFail:    ho.onHoFail,
		},
		NonTransitionalEvents: []model.EventType{
			model.HoUeConnectedEvent,
			model.HoPathSwitchReqEvent,
			model.HoPathSwitchAckEvent,
			model.HoRlinkPrepareRespEvent,
		},
	}
}

func (ho *HoProcedure) StartHo() {
	ho.startTime = time.Now()
	if ho.prepareStart.IsZero() {
		ho.prepareStart = ho.startTime
	}
	event := fsm.NewEventData(model.HoDecisionEvent, &HoEventData{
		PrUeId:      ho.prUeId,
		SourceGnbId: ho.sourceGnbId,
		TargetGnbId: ho.targetGnbId,
	})
	ho.fsmHo.SendEvent(ho.stateHo, event)
}

func (ho *HoProcedure) SendEvent(eventType model.EventType, data *HoEventData) {
	event := fsm.NewEventData(eventType, data)
	ho.fsmHo.SendEvent(ho.stateHo, event)
}

func (ho *HoProcedure) GetState() model.StateType {
	return ho.stateHo.CurrentState()
}

func (ho *HoProcedure) IsSuccess() bool {
	return ho.success
}

func (ho *HoProcedure) GetFailureReason() string {
	return ho.failureReason
}

func (ho *HoProcedure) GetStartTime() time.Time {
	return ho.startTime
}

func (ho *HoProcedure) GetEndTime() time.Time {
	return ho.endTime
}

func (ho *HoProcedure) GetDuration() time.Duration {
	return ho.duration
}

func (ho *HoProcedure) GetPrepareStart() time.Time {
	return ho.prepareStart
}

func (ho *HoProcedure) GetExecuteStart() time.Time {
	return ho.executeStart
}

func (ho *HoProcedure) GetCompleteStart() time.Time {
	return ho.completeStart
}

func (ho *HoProcedure) GetFailStart() time.Time {
	return ho.failStart
}

func (ho *HoProcedure) finishTiming() {
	if ho.finished {
		return
	}
	ho.endTime = time.Now()
	ho.duration = ho.endTime.Sub(ho.startTime)
	ho.finished = true
}

func (ho *HoProcedure) updateUeState(state model.StateType) {
	hoLogger.Info("[UE-%d] HO FSM: UE state -> %s", ho.prUeId, state)
	if ho.ueStateCallback != nil {
		ho.ueStateCallback(ho.prUeId, state)
	}
}

func (ho *HoProcedure) onNull(state *fsm.State, event *fsm.EventData) {
	switch event.Type() {
	case model.EntryEvent:
		hoLogger.Info("[UE-%d] HO FSM: ENTER NULL", ho.prUeId)
	case model.ExitEvent:
		hoLogger.Info("[UE-%d] HO FSM: EXIT NULL", ho.prUeId)
	case model.HoDecisionEvent:
		hoLogger.Info("[UE-%d] HO FSM: NULL + HoDecisionEvent -> PREPARE", ho.prUeId)
		data := fsm.GetEventData[HoEventData](event)
		if data != nil {
			ho.triggerXnHandover(data)
		}
	}
}

func (ho *HoProcedure) onPrepare(state *fsm.State, event *fsm.EventData) {
	switch event.Type() {
	case model.EntryEvent:
		hoLogger.Info("[UE-%d] HO FSM: ENTER PREPARE", ho.prUeId)
	case model.ExitEvent:
		hoLogger.Info("[UE-%d] HO FSM: EXIT PREPARE", ho.prUeId)
	case model.HoRlinkPrepareReqSentEvent:
		hoLogger.Info("[UE-%d] HO FSM: PREPARE + HoRlinkPrepareReqSentEvent -> EXECUTE", ho.prUeId)
		if ho.executeStart.IsZero() {
			ho.executeStart = time.Now()
		}
		ho.updateUeState(model.HoStart)
	case model.HoFailEvent:
		hoLogger.Warn("[UE-%d] HO FSM: PREPARE + HoFailEvent", ho.prUeId)
		data := fsm.GetEventData[HoEventData](event)
		ho.handleHoFail(data)
	}
}

func (ho *HoProcedure) onExecute(state *fsm.State, event *fsm.EventData) {
	switch event.Type() {
	case model.EntryEvent:
		hoLogger.Info("[UE-%d] HO FSM: ENTER EXECUTE", ho.prUeId)
	case model.ExitEvent:
		hoLogger.Info("[UE-%d] HO FSM: EXIT EXECUTE", ho.prUeId)
	case model.HoPathSwitchRequestEvent:
		hoLogger.Info("[UE-%d] HO FSM: EXECUTE + HoPathSwitchRequestEvent -> COMPLETE", ho.prUeId)
		if ho.completeStart.IsZero() {
			ho.completeStart = time.Now()
		}
	case model.HoPathSwitchFailEvent:
		hoLogger.Warn("[UE-%d] HO FSM: EXECUTE + HoPathSwitchFailEvent -> HO_FAIL", ho.prUeId)
		data := fsm.GetEventData[HoEventData](event)
		ho.handleHoFail(data)
	case model.HoFailEvent:
		hoLogger.Warn("[UE-%d] HO FSM: EXECUTE + HoFailEvent -> HO_FAIL", ho.prUeId)
		data := fsm.GetEventData[HoEventData](event)
		ho.handleHoFail(data)
	}
}

func (ho *HoProcedure) onComplete(state *fsm.State, event *fsm.EventData) {
	switch event.Type() {
	case model.EntryEvent:
		hoLogger.Info("[UE-%d] HO FSM: ENTER COMPLETE", ho.prUeId)
	case model.ExitEvent:
		hoLogger.Info("[UE-%d] HO FSM: EXIT COMPLETE", ho.prUeId)
	case model.HoRlinkSetupPduSessonEvent:
		hoLogger.Info("[UE-%d] HO FSM: COMPLETE + HoRlinkSetupPduSessonEvent -> SUCCESS", ho.prUeId)
		ho.finishTiming()
		ho.updateUeState(model.HoSuccess)
	case model.HoFailEvent:
		hoLogger.Warn("[UE-%d] HO FSM: COMPLETE + HoFailEvent -> HO_FAIL", ho.prUeId)
		data := fsm.GetEventData[HoEventData](event)
		ho.handleHoFail(data)
	}
}

func (ho *HoProcedure) onHoSuccess(state *fsm.State, event *fsm.EventData) {
	switch event.Type() {
	case model.EntryEvent:
		hoLogger.Info("[UE-%d] HO FSM: ENTER SUCCESS (HO Complete!)", ho.prUeId)
	case model.ExitEvent:
	}
}

func (ho *HoProcedure) onHoFail(state *fsm.State, event *fsm.EventData) {
	switch event.Type() {
	case model.EntryEvent:
		hoLogger.Warn("[UE-%d] HO FSM: ENTER FAIL (HO Failed!)", ho.prUeId)
		ho.updateUeState(model.HoFail)
	case model.ExitEvent:
	}
}

func (ho *HoProcedure) triggerXnHandover(data *HoEventData) {
	sourceGnb := ho.findGnbById(ho.sourceGnbId)
	targetGnb := ho.findGnbById(ho.targetGnbId)
	if sourceGnb == nil || targetGnb == nil {
		return
	}

	if ho.handoverType == 2 {
		hoLogger.Info("[UE-%d] HO FSM: Triggering N2Handover %s->%s", ho.prUeId, ho.sourceGnbId, ho.targetGnbId)
		gnbcontext.TriggerNgapHandover(sourceGnb, targetGnb, ho.prUeId, ho.loss, ho.delay, ho.jitter)
	} else {
		hoLogger.Info("[UE-%d] HO FSM: Triggering XnHandover %s->%s", ho.prUeId, ho.sourceGnbId, ho.targetGnbId)
		gnbcontext.TriggerXnHandover(sourceGnb, targetGnb, ho.prUeId, ho.loss, ho.delay, ho.jitter)
	}
}

func (ho *HoProcedure) findGnbById(gnbId string) *gnbcontext.GnbContext {
	if gnb, ok := gnbcontext.Gnbs.Load(gnbId); ok {
		return gnb.(*gnbcontext.GnbContext)
	}
	return nil
}

func (ho *HoProcedure) triggerPathSwitchRequest(data *HoEventData) {
}

func (ho *HoProcedure) handleHoFail(data *HoEventData) {
	ho.finishTiming()
	if ho.failStart.IsZero() {
		ho.failStart = time.Now()
	}
	ho.success = false
	if data != nil {
		ho.failureReason = data.FailureReason
	}
	hoLogger.Error("[UE-%d] HO FSM: FAILURE - %s", ho.prUeId, ho.failureReason)
	ho.updateUeState(model.HoFail)
}
