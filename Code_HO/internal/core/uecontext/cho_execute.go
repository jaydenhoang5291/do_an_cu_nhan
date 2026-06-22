package uecontext

import (
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
)

func (ue *UeContext) triggerChoExecute(candidate *model.CandidateGnb) {
	ue.Info("CHO executing: switching to target gNB %s", candidate.GnbId)

	if candidate.Conn == nil {
		conn := rlink.NewConnection(int64(ue.id), ue.msin,
			candidate.GnbId, rlink.DefaultBufferSize, rlink.DefaultDuration, 1)
		candidate.Conn = conn
	}

	ue.mutex.Lock()
	oldConn := ue.rlinkConn
	oldGnbId := ue.gnbId
	ue.rlinkConn = candidate.Conn
	ue.gnbId = candidate.GnbId
	ue.mutex.Unlock()

	if oldConn != nil {
		oldConn.Close()
	}

	gnbcontext.SendToGnb(ue.msin, &model.CHOExecuteNotification{
		PrUeId:      int64(ue.id),
		TargetGnbId: candidate.GnbId,
		SourceGnbId: oldGnbId,
	}, oldGnbId, false)

	ue.timerEngine.CreateTimer(timer.TimerConfig{
		TimerType: timer.T304,
		Duration:  timer.T304_duration,
		CountMax:  1,
		ExpireFunc: func() {
			ue.Warn("CHO T304 expired: handover to %s failed", candidate.GnbId)
		},
	}, int64(ue.id))
	ue.timerEngine.Start(timer.T304, int64(ue.id))

	ue.sendGnb(&model.RRCReconfigurationComplete{
		PrUeId:        int64(ue.id),
		TargetGnbId:   candidate.GnbId,
		SelectedGnbId: candidate.GnbId,
	})

	ue.Info("CHO to gNB %s initiated, waiting for completion", candidate.GnbId)

	ue.choCandidates = nil
	ue.choEnabled = false
}
