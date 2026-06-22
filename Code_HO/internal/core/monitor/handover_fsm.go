package monitor

import (
	"github.com/alitto/pond/v2"
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/pool"
	"stormsim/pkg/model"
	"sync"
)

type HandoverContext struct {
	PrUeId       int64
	SourceGnbId  string
	TargetGnbId  string
	HandoverType int
	Manager      *HandoverMonitor
	State        *fsm.State
}

var HandoverFSM *fsm.Fsm
var handoverFsmOnce sync.Once

func InitHandoverFSM() {
	handoverFsmOnce.Do(func() {
		HandoverFSM = createHandoverFSM(pool.GnbWorkerPool)
	})
}

func createHandoverFSM(w pond.Pool) *fsm.Fsm {
	transitions := fsm.Transitions{
		fsm.Tuple(model.HO_NULL, model.HO_PrepareEvent):        model.HO_PREPARATION,
		fsm.Tuple(model.HO_PREPARATION, model.HO_ExecuteEvent): model.HO_EXECUTION,
		fsm.Tuple(model.HO_EXECUTION, model.HO_CompleteEvent):  model.HO_COMPLETION,
		fsm.Tuple(model.HO_COMPLETION, model.HandoverFinished): model.HO_NULL,
	}

	callbacks := fsm.Callbacks{
		model.HO_PREPARATION: ho_Preparation,
		model.HO_EXECUTION:   ho_Execution,
		model.HO_COMPLETION:  ho_Completion,
		model.HO_NULL:        ho_Null,
	}

	return fsm.NewFsm(fsm.Options{
		Transitions: transitions,
		Callbacks:   callbacks,
	}, w)
}

func ho_Preparation(state *fsm.State, event *fsm.EventData) {
	hoCtx := fsm.GetStateInfo[HandoverContext](state)
	switch event.Type() {
	case model.EntryEvent:
		if hoCtx.Manager != nil {
			hoCtx.Manager.logger.Info("HO_PREPARATION[UE-%d]: Handover preparation started from %s to %s", hoCtx.PrUeId, hoCtx.SourceGnbId, hoCtx.TargetGnbId)
		}
	}
}

func ho_Execution(state *fsm.State, event *fsm.EventData) {
	hoCtx := fsm.GetStateInfo[HandoverContext](state)
	switch event.Type() {
	case model.EntryEvent:
		if hoCtx.Manager != nil {
			hoCtx.Manager.logger.Info("HO_EXECUTION[UE-%d]: UE executing connection swap to target %s", hoCtx.PrUeId, hoCtx.TargetGnbId)
		}
	}
}

func ho_Completion(state *fsm.State, event *fsm.EventData) {
	hoCtx := fsm.GetStateInfo[HandoverContext](state)
	switch event.Type() {
	case model.EntryEvent:
		if hoCtx.Manager != nil {
			hoCtx.Manager.logger.Info("HO_COMPLETION[UE-%d]: Path switch/context release completing for target %s", hoCtx.PrUeId, hoCtx.TargetGnbId)
		}
		// Auto-revert to NULL after completion logs
		state.SetNextEvent(fsm.NewEmptyEventData(model.HandoverFinished))
	}
}

func ho_Null(state *fsm.State, event *fsm.EventData) {
	hoCtx := fsm.GetStateInfo[HandoverContext](state)
	switch event.Type() {
	case model.EntryEvent:
		if hoCtx.Manager != nil {
			hoCtx.Manager.logger.Info("HO_NULL[UE-%d]: Handover finished and FSM reverted.", hoCtx.PrUeId)
			// Cleanup state from HandoverMonitor
			hoCtx.Manager.activeFsmLock.Lock()
			delete(hoCtx.Manager.activeFsms, hoCtx.PrUeId)
			hoCtx.Manager.activeFsmLock.Unlock()
		}
	}
}

func NewHandoverState(manager *HandoverMonitor, prUeId int64, sourceGnbId, targetGnbId string, handoverType int) *fsm.State {
	ctx := &HandoverContext{
		PrUeId:       prUeId,
		SourceGnbId:  sourceGnbId,
		TargetGnbId:  targetGnbId,
		HandoverType: handoverType,
		Manager:      manager,
	}
	state := fsm.NewState(model.HO_NULL, ctx)
	ctx.State = state
	return state
}
