package gnbcontext

import (
	"stormsim/internal/core/uecontext/timer"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
	"time"
)

func (gnb *GnbContext) PrepareChoHandover(candidates []string, prUeId int64) {
	sourceUe, err := gnb.GetGnbUeByPrUeId(prUeId)
	if err != nil || sourceUe == nil {
		gnb.Error("Cannot find UE %d for CHO preparation", prUeId)
		return
	}

	gnb.Info("Preparing CHO for UE %d with %d candidates", prUeId, len(candidates))

	choCandidates := make([]model.CandidateGnb, 0, len(candidates))
	for _, targetGnbId := range candidates {
		if targetGnbId == gnb.GetId() {
			continue
		}

		val, ok := Gnbs.Load(targetGnbId)
		if !ok {
			gnb.Warn("Target gNB %s not found, skipping", targetGnbId)
			continue
		}
		targetGnb := val.(*GnbContext)

		conn := rlink.NewConnection(prUeId, sourceUe.msin,
			targetGnb.controlPlaneInfo.gnbId, rlink.DefaultBufferSize, rlink.DefaultDuration, 1)
		conn.DownlinkDelay = timer.T304_duration + 500*time.Millisecond
		conn.DownlinkLoss = gnb.controlPlaneInfo.radioLoss
		conn.UplinkLoss = gnb.controlPlaneInfo.radioLoss

		SendToGnb(gnb.GetId(), &model.RLinkHandoverForwardUeContext{
			PrUeId:        prUeId,
			Msin:          sourceUe.msin,
			Conn:          conn,
			AmfUeNgapId:   sourceUe.amfUeNgapId,
			UeCoreContext: &sourceUe.context,
			SourceGnbId:   gnb.controlPlaneInfo.gnbId,
		}, targetGnbId, true)

		choCandidates = append(choCandidates, model.CandidateGnb{
			GnbId: targetGnbId,
			Conn:  conn,
		})

		gnb.Info("Prepared candidate gNB %s for UE %d", targetGnbId, prUeId)
	}

	if len(choCandidates) == 0 {
		gnb.Error("No valid CHO candidates found for UE %d", prUeId)
		return
	}

	sourceUe.sendCriticalMsgToUe(&model.CHOConfig{
		PrUeId:     prUeId,
		Candidates: choCandidates,
	})

	gnb.Info("CHOConfig sent to UE %d with %d candidates", prUeId, len(choCandidates))
}

func (gnb *GnbContext) HandleMeasurementReport(report *model.MeasurementReport) {
	if report.ServingRsrp == 0 || len(report.Neighbors) == 0 {
		return
	}

	if report.ServingRsrp < -100 {
		candidates := make([]string, 0, len(report.Neighbors))
		for _, n := range report.Neighbors {
			if n.Rsrp > -100 && n.Rsrp > report.ServingRsrp+5 {
				candidates = append(candidates, n.GnbId)
			}
		}
		if len(candidates) > 0 {
			gnb.Info("Serving RSRP=%.1f is low, preparing CHO with %d candidates for UE %d",
				report.ServingRsrp, len(candidates), report.PrUeId)
			gnb.PrepareChoHandover(candidates, report.PrUeId)
		}
	}
}

func HandleChoExecuteNotification(sourceGnbId string, notification *model.CHOExecuteNotification) {
	val, ok := Gnbs.Load(sourceGnbId)
	if !ok {
		return
	}
	sourceGnb := val.(*GnbContext)

	sourceGnb.Info("CHO executed: UE %d switched to %s", notification.PrUeId, notification.TargetGnbId)

	ue, err := sourceGnb.GetGnbUeByPrUeId(notification.PrUeId)
	if err != nil {
		sourceGnb.Warn("UE %d already cleaned up", notification.PrUeId)
		return
	}

	sourceGnb.deleteGnBUe(ue)
	sourceGnb.Info("Cleaned up UE %d from source gNB %s after CHO", notification.PrUeId, sourceGnbId)
}
