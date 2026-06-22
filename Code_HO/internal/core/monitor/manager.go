package monitor

import (
	"context"
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/logger"
	"stormsim/internal/common/pool"
	"stormsim/internal/core/gnbcontext"
	"stormsim/pkg/model"
	"sync"
	"sync/atomic"
	"time"
)

const (
	HandoverTypeXn = 1
	HandoverTypeN2 = 2
)

type HandoverEvent struct {
	PrUeId       int64
	SourceGnbId  string
	TargetGnbId  string
	HandoverType int
	Priority     int
}

type HandoverCallback func(result HandoverResult)

type HandoverResult struct {
	PrUeId        int64
	SourceGnbId   string
	TargetGnbId   string
	HandoverType  int
	Success       bool
	FailureReason string
	Duration      time.Duration
	StartTime     time.Time
	EndTime       time.Time
}

type HandoverMonitor struct {
	logger *logger.Logger

	gnbs sync.Map

	gnbGroups     map[string]*GnbGroup
	gnbGroupsLock sync.RWMutex

	handoverQueue *HandoverQueue

	ntnCoordinator *NTNCoordinator

	tracker *MonitorHandoverTracker

	eventChan chan HandoverEvent

	callbacks sync.Map

	hoProcedures sync.Map // prUeId int64: *HoProcedure

	ctx    context.Context
	cancel context.CancelFunc

	totalHandovers   atomic.Int64
	successHandovers atomic.Int64
	failedHandovers  atomic.Int64

	activeFsms    map[int64]*fsm.State
	activeFsmLock sync.RWMutex
}

func NewHandoverMonitor(ctx context.Context) *HandoverMonitor {
	childCtx, cancel := context.WithCancel(ctx)

	hm := &HandoverMonitor{
		logger:         logger.InitLogger("handover-manager", map[string]string{"mod": "handover-manager"}),
		gnbGroups:      make(map[string]*GnbGroup),
		eventChan:      make(chan HandoverEvent, 1000),
		ctx:            childCtx,
		cancel:         cancel,
		handoverQueue:  NewHandoverQueue(),
		ntnCoordinator: NewNTNCoordinator(),
	}

	hm.tracker = NewMonitorHandoverTracker(hm)

	hm.activeFsms = make(map[int64]*fsm.State)

	InitHandoverFSM()

	gnbcontext.HandoverStateCallback = func(prUeId int64, eventType model.EventType) {
		state := hm.GetHandoverState(prUeId)
		if state != nil && HandoverFSM != nil {
			HandoverFSM.SendEvent(state, fsm.NewEmptyEventData(eventType))
		}
	}

	go hm.runEventLoop()
	go hm.runScheduler()

	hm.registerHoEventHandler()

	return hm
}

func (hm *HandoverMonitor) registerHoEventHandler() {
	gnbcontext.HoEventHandler = func(prUeId int64, eventType model.EventType, data interface{}) {
		ho, ok := hm.hoProcedures.Load(prUeId)
		if !ok {
			if eventType == model.HoDecisionEvent {
				if m, ok := data.(map[string]interface{}); ok {
					sourceGnbId := m["SourceGnbId"].(string)
					targetGnbId := m["TargetGnbId"].(string)
					loss, _ := m["Loss"].(float64)
					delay, _ := m["Delay"].(time.Duration)
					jitter, _ := m["Jitter"].(time.Duration)
					ho = hm.createHoProcedure(prUeId, sourceGnbId, targetGnbId, loss, delay, jitter)
				}
			}
		}
		if ho != nil {
			ho.(*HoProcedure).SendEvent(eventType, nil)
		}
	}
}

func (hm *HandoverMonitor) createHoProcedure(prUeId int64, sourceGnbId, targetGnbId string, loss float64, delay, jitter time.Duration) *HoProcedure {
	ho := NewHoProcedure(prUeId, sourceGnbId, targetGnbId, 1, loss, delay, jitter)
	ho.SetUeStateCallback(func(id int64, state model.StateType) {
		hm.updateUeHoState(id, state)
	})
	hm.hoProcedures.Store(prUeId, ho)
	ho.StartHo()
	return ho
}

func (hm *HandoverMonitor) updateUeHoState(prUeId int64, state model.StateType) {
	gnbcontext.Gnbs.Range(func(k, v interface{}) bool {
		gnb := v.(*gnbcontext.GnbContext)
		ue, err := gnb.GetGnbUeByPrUeId(prUeId)
		if err == nil && ue != nil {
			ue.SetHoState(state)
			return false
		}
		return true
	})
}

func (hm *HandoverMonitor) RegisterGnb(gnb *gnbcontext.GnbContext, groupType GnbGroupType, groupName string) {
	gnbId := gnb.GetId()
	hm.gnbs.Store(gnbId, gnb)

	hm.gnbGroupsLock.Lock()
	defer hm.gnbGroupsLock.Unlock()

	group, exists := hm.gnbGroups[groupName]
	if !exists {
		group = &GnbGroup{
			Name: groupName,
			Type: groupType,
			Gnbs: make([]*gnbcontext.GnbContext, 0),
			NetworkCondition: &NetworkCondition{
				PacketLoss: 0,
				LatencyMs:  0,
				JitterMs:   0,
			},
		}
		hm.gnbGroups[groupName] = group
	}

	group.Gnbs = append(group.Gnbs, gnb)

	if groupType == GroupTypeNTN {
		hm.ntnCoordinator.RegisterSatellite(gnbId)
	}

	hm.logger.Info("Registered gNB %s to group %s (type: %v)", gnbId, groupName, groupType)
}

func (hm *HandoverMonitor) UnregisterGnb(gnbId string) {
	hm.gnbs.Delete(gnbId)

	hm.gnbGroupsLock.Lock()
	defer hm.gnbGroupsLock.Unlock()

	for _, group := range hm.gnbGroups {
		for i, gnb := range group.Gnbs {
			if gnb.GetId() == gnbId {
				group.Gnbs = append(group.Gnbs[:i], group.Gnbs[i+1:]...)
				break
			}
		}
	}

	hm.ntnCoordinator.UnregisterSatellite(gnbId)
	hm.logger.Info("Unregistered gNB %s", gnbId)
}

func (hm *HandoverMonitor) GetGnb(gnbId string) (*gnbcontext.GnbContext, bool) {
	val, ok := hm.gnbs.Load(gnbId)
	if !ok {
		return nil, false
	}
	return val.(*gnbcontext.GnbContext), true
}

func (hm *HandoverMonitor) ScheduleHandover(prUeId int64, sourceGnbId, targetGnbId string, handoverType int, executeAt time.Time) error {
	sh := &ScheduledHandover{
		PrUeId:       prUeId,
		SourceGnbId:  sourceGnbId,
		TargetGnbId:  targetGnbId,
		HandoverType: handoverType,
		ExecuteAt:    executeAt,
		Priority:     0,
	}

	hm.handoverQueue.Schedule(sh)
	hm.logger.Info("Scheduled handover: UE-%d from %s to %s at %v", prUeId, sourceGnbId, targetGnbId, executeAt)
	return nil
}

func (hm *HandoverMonitor) ScheduleHandoverWithPriority(prUeId int64, sourceGnbId, targetGnbId string, handoverType int, executeAt time.Time, priority int) error {
	sh := &ScheduledHandover{
		PrUeId:       prUeId,
		SourceGnbId:  sourceGnbId,
		TargetGnbId:  targetGnbId,
		HandoverType: handoverType,
		ExecuteAt:    executeAt,
		Priority:     priority,
	}

	hm.handoverQueue.Schedule(sh)
	hm.logger.Info("Scheduled handover with priority %d: UE-%d from %s to %s", priority, prUeId, sourceGnbId, targetGnbId)
	return nil
}

func (hm *HandoverMonitor) ScheduleBatch(handovers []*ScheduledHandover) {
	for _, sh := range handovers {
		hm.handoverQueue.Schedule(sh)
	}
	hm.logger.Info("Scheduled batch of %d handovers", len(handovers))
}

func (hm *HandoverMonitor) CancelHandover(prUeId int64) bool {
	cancelled := hm.handoverQueue.Cancel(prUeId)
	if cancelled {
		hm.logger.Info("Cancelled handover for UE-%d", prUeId)
	}
	return cancelled
}

func (hm *HandoverMonitor) TriggerImmediateHandover(prUeId int64, sourceGnbId, targetGnbId string, handoverType int) error {
	event := HandoverEvent{
		PrUeId:       prUeId,
		SourceGnbId:  sourceGnbId,
		TargetGnbId:  targetGnbId,
		HandoverType: handoverType,
		Priority:     100,
	}

	select {
	case hm.eventChan <- event:
		hm.logger.Info("Triggered immediate handover: UE-%d from %s to %s", prUeId, sourceGnbId, targetGnbId)
		return nil
	default:
		hm.logger.Error("Event channel full, cannot trigger handover for UE-%d", prUeId)
		return ErrEventChannelFull
	}
}

func (hm *HandoverMonitor) RegisterCallback(prUeId int64, cb HandoverCallback) {
	hm.callbacks.Store(prUeId, cb)
}

func (hm *HandoverMonitor) UnregisterCallback(prUeId int64) {
	hm.callbacks.Delete(prUeId)
}

func (hm *HandoverMonitor) OnHandoverComplete(prUeId int64, sourceGnbId, targetGnbId string, handoverType int, success bool, failureReason string, duration time.Duration) {
	result := HandoverResult{
		PrUeId:        prUeId,
		SourceGnbId:   sourceGnbId,
		TargetGnbId:   targetGnbId,
		HandoverType:  handoverType,
		Success:       success,
		FailureReason: failureReason,
		Duration:      duration,
		EndTime:       time.Now(),
	}

	hm.totalHandovers.Add(1)
	if success {
		hm.successHandovers.Add(1)
	} else {
		hm.failedHandovers.Add(1)
	}

	if val, ok := hm.callbacks.Load(prUeId); ok {
		if cb, ok := val.(HandoverCallback); ok {
			go cb(result)
		}
	}

	hm.tracker.RecordResult(result)
	hm.logger.Info("Handover complete: UE-%d success=%v duration=%v", prUeId, success, duration)
}

func (hm *HandoverMonitor) runEventLoop() {
	for {
		select {
		case <-hm.ctx.Done():
			hm.logger.Info("Event loop stopped")
			return
		case event := <-hm.eventChan:
			pool.GnbWorkerPool.Submit(func() {
				hm.executeHandover(event)
			})
		}
	}
}

func (hm *HandoverMonitor) runScheduler() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-hm.ctx.Done():
			hm.logger.Info("Scheduler stopped")
			return
		case <-ticker.C:
			if sh := hm.handoverQueue.GetNext(); sh != nil {
				if sh.ExecuteAt.Before(time.Now()) || sh.ExecuteAt.Equal(time.Now()) {
					event := HandoverEvent{
						PrUeId:       sh.PrUeId,
						SourceGnbId:  sh.SourceGnbId,
						TargetGnbId:  sh.TargetGnbId,
						HandoverType: sh.HandoverType,
						Priority:     sh.Priority,
					}
					select {
					case hm.eventChan <- event:
					default:
						hm.logger.Warn("Event channel full, requeueing handover for UE-%d", sh.PrUeId)
						hm.handoverQueue.Schedule(sh)
					}
				} else {
					hm.handoverQueue.Schedule(sh)
				}
			}
		}
	}
}

func (hm *HandoverMonitor) executeHandover(event HandoverEvent) {
	sourceGnb, ok := hm.GetGnb(event.SourceGnbId)
	if !ok {
		hm.OnHandoverComplete(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType, false, "source gNB not found", 0)
		return
	}

	targetGnb, ok := hm.GetGnb(event.TargetGnbId)
	if !ok {
		hm.OnHandoverComplete(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType, false, "target gNB not found", 0)
		return
	}

	if !hm.ntnCoordinator.IsHandoverPossible(event.SourceGnbId, event.TargetGnbId) {
		hm.OnHandoverComplete(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType, false, "handover not possible (visibility)", 0)
		return
	}

	startTime := time.Now()

	condition := hm.getConditionForGnb(event.TargetGnbId)
	var loss float64
	var delay, jitter time.Duration
	if condition != nil {
		loss = condition.PacketLoss
		delay = time.Duration(condition.LatencyMs) * time.Millisecond
		jitter = time.Duration(condition.JitterMs) * time.Millisecond
	}

	switch event.HandoverType {
	case HandoverTypeXn:
		hm.GetOrCreateHandoverState(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType)
		if gnbcontext.HandoverStateCallback != nil {
			gnbcontext.HandoverStateCallback(event.PrUeId, model.HO_PrepareEvent)
		}
		gnbcontext.TriggerXnHandover(sourceGnb, targetGnb, event.PrUeId, loss, delay, jitter)
	case HandoverTypeN2:
		hm.GetOrCreateHandoverState(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType)
		if gnbcontext.HandoverStateCallback != nil {
			gnbcontext.HandoverStateCallback(event.PrUeId, model.HO_PrepareEvent)
		}
		gnbcontext.TriggerNgapHandover(sourceGnb, targetGnb, event.PrUeId, loss, delay, jitter)
	default:
		hm.OnHandoverComplete(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType, false, "unknown handover type", 0)
		return
	}

	success := sourceGnb.IsHandoverSuccess(event.PrUeId)
	duration := time.Since(startTime)

	var reason string
	if !success {
		reason = "handover failed at gNB"
	}

	hm.OnHandoverComplete(event.PrUeId, event.SourceGnbId, event.TargetGnbId, event.HandoverType, success, reason, duration)
}

func (hm *HandoverMonitor) GetStats() (total, success, failed int64) {
	return hm.totalHandovers.Load(), hm.successHandovers.Load(), hm.failedHandovers.Load()
}

func (hm *HandoverMonitor) GetTracker() *MonitorHandoverTracker {
	return hm.tracker
}

func (hm *HandoverMonitor) GetAllGroups() map[string]*GnbGroup {
	hm.gnbGroupsLock.RLock()
	defer hm.gnbGroupsLock.RUnlock()

	result := make(map[string]*GnbGroup)
	for _, group := range hm.gnbGroups {
		result[group.Name] = group
	}
	return result
}

func (hm *HandoverMonitor) GetGroup(name string) *GnbGroup {
	hm.gnbGroupsLock.RLock()
	defer hm.gnbGroupsLock.RUnlock()

	if group, ok := hm.gnbGroups[name]; ok {
		return group
	}
	return nil
}

func (hm *HandoverMonitor) GetNTNCoordinator() *NTNCoordinator {
	return hm.ntnCoordinator
}

func (hm *HandoverMonitor) GetQueueSize() int {
	return hm.handoverQueue.Size()
}

func (hm *HandoverMonitor) Stop() {
	hm.cancel()
	close(hm.eventChan)
	hm.logger.Info("HandoverMonitor stopped")
}

func (hm *HandoverMonitor) GetBestHandoverTarget(prUeId int64, sourceGnbId string) string {
	sourceGnb, ok := hm.GetGnb(sourceGnbId)
	if !ok {
		return ""
	}

	sourceGroup := hm.getGnbGroup(sourceGnbId)
	if sourceGroup == nil {
		return ""
	}

	if sourceGroup.Type == GroupTypeGround {
		visibleSatellites := hm.ntnCoordinator.GetVisibleSatellites(sourceGnbId)
		if len(visibleSatellites) > 0 {
			return visibleSatellites[0]
		}
	}

	hm.gnbGroupsLock.RLock()
	defer hm.gnbGroupsLock.RUnlock()

	for _, group := range hm.gnbGroups {
		if group.Type != sourceGroup.Type {
			for _, gnb := range group.Gnbs {
				if gnb.GetId() != sourceGnbId {
					if hm.ntnCoordinator.IsHandoverPossible(sourceGnbId, gnb.GetId()) {
						return gnb.GetId()
					}
				}
			}
		}
	}

	_ = sourceGnb
	return ""
}

func (hm *HandoverMonitor) getGnbGroup(gnbId string) *GnbGroup {
	hm.gnbGroupsLock.RLock()
	defer hm.gnbGroupsLock.RUnlock()

	for _, group := range hm.gnbGroups {
		for _, gnb := range group.Gnbs {
			if gnb.GetId() == gnbId {
				return group
			}
		}
	}
	return nil
}

func (hm *HandoverMonitor) SetGroupNetworkCondition(groupName string, condition *NetworkCondition) {
	hm.gnbGroupsLock.Lock()
	defer hm.gnbGroupsLock.Unlock()

	if group, ok := hm.gnbGroups[groupName]; ok {
		group.NetworkCondition = condition
		hm.logger.Info("Updated network condition for group %s: loss=%.2f latency=%dms", groupName, condition.PacketLoss, condition.LatencyMs)
	}
}

func (hm *HandoverMonitor) GetGroupNetworkCondition(groupName string) *NetworkCondition {
	hm.gnbGroupsLock.RLock()
	defer hm.gnbGroupsLock.RUnlock()

	if group, ok := hm.gnbGroups[groupName]; ok {
		return group.NetworkCondition
	}
	return nil
}

func (hm *HandoverMonitor) PredictNextHandoverWindow(sourceGnbId, targetGnbId string) time.Time {
	return hm.ntnCoordinator.PredictNextHandoverWindow(sourceGnbId, targetGnbId)
}

func (hm *HandoverMonitor) getConditionForGnb(gnbId string) *NetworkCondition {
	group := hm.getGnbGroup(gnbId)
	if group == nil {
		return nil
	}
	return group.NetworkCondition
}

func (hm *HandoverMonitor) GetOrCreateHandoverState(prUeId int64, sourceGnbId, targetGnbId string, hoType int) *fsm.State {
	hm.activeFsmLock.Lock()
	defer hm.activeFsmLock.Unlock()
	state, exists := hm.activeFsms[prUeId]
	if !exists {
		state = NewHandoverState(hm, prUeId, sourceGnbId, targetGnbId, hoType)
		hm.activeFsms[prUeId] = state
	}
	return state
}

func (hm *HandoverMonitor) GetHandoverState(prUeId int64) *fsm.State {
	hm.activeFsmLock.RLock()
	defer hm.activeFsmLock.RUnlock()
	return hm.activeFsms[prUeId]
}
