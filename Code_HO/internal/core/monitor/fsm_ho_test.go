package monitor

import (
	"context"
	"testing"
	"time"

	"stormsim/internal/common/fsm"
	"stormsim/internal/common/pool"
	"stormsim/pkg/model"
)

func initTestPool() {
	pool.InitWorkerPool(context.Background(), 100, 10, 1, 2)
}

func TestHoProcedure_PhaseTimestamps_Success(t *testing.T) {
	initTestPool()

	ho := NewHoProcedure(1, "000008", "000009", 1, 0, 0, 0)

	ho.StartHo()
	if err := ho.fsmHo.SyncSendEvent(ho.stateHo, fsm.NewEventData(model.HoDecisionEvent, &HoEventData{
		PrUeId:      1,
		SourceGnbId: "000008",
		TargetGnbId: "000009",
	})); err != nil {
		t.Fatalf("HoDecisionEvent: %v", err)
	}

	prepareTs := ho.GetPrepareStart()
	if prepareTs.IsZero() {
		t.Fatal("prepareStart should be non-zero after StartHo")
	}

	if err := ho.fsmHo.SyncSendEvent(ho.stateHo, fsm.NewEmptyEventData(model.HoRlinkPrepareReqSentEvent)); err != nil {
		t.Fatalf("HoRlinkPrepareReqSentEvent: %v", err)
	}

	executeTs := ho.GetExecuteStart()
	if executeTs.IsZero() {
		t.Fatal("executeStart should be non-zero after HoRlinkPrepareReqSentEvent")
	}
	if executeTs.Before(prepareTs) {
		t.Fatalf("executeStart (%v) should be >= prepareStart (%v)", executeTs, prepareTs)
	}

	if err := ho.fsmHo.SyncSendEvent(ho.stateHo, fsm.NewEmptyEventData(model.HoPathSwitchRequestEvent)); err != nil {
		t.Fatalf("HoPathSwitchRequestEvent: %v", err)
	}

	completeTs := ho.GetCompleteStart()
	if completeTs.IsZero() {
		t.Fatal("completeStart should be non-zero after HoPathSwitchRequestEvent")
	}
	if completeTs.Before(executeTs) {
		t.Fatalf("completeStart (%v) should be >= executeStart (%v)", completeTs, executeTs)
	}

	if err := ho.fsmHo.SyncSendEvent(ho.stateHo, fsm.NewEmptyEventData(model.HoRlinkSetupPduSessonEvent)); err != nil {
		t.Fatalf("HoRlinkSetupPduSessonEvent: %v", err)
	}

	if ho.GetDuration() == 0 {
		t.Fatal("duration should be non-zero after successful handover")
	}
}

func TestHoProcedure_ExecuteStartNotZero(t *testing.T) {
	initTestPool()

	ho := NewHoProcedure(1, "000008", "000009", 1, 0, 0, 0)
	ho.StartHo()
	if err := ho.fsmHo.SyncSendEvent(ho.stateHo, fsm.NewEventData(model.HoDecisionEvent, &HoEventData{
		PrUeId:      1,
		SourceGnbId: "000008",
		TargetGnbId: "000009",
	})); err != nil {
		t.Fatalf("HoDecisionEvent: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := ho.fsmHo.SyncSendEvent(ho.stateHo, fsm.NewEmptyEventData(model.HoRlinkPrepareReqSentEvent)); err != nil {
		t.Fatalf("HoRlinkPrepareReqSentEvent: %v", err)
	}

	executeTs := ho.GetExecuteStart()
	if executeTs.IsZero() {
		t.Fatal("executeStart must not be zero - this is the bug being fixed")
	}
}
