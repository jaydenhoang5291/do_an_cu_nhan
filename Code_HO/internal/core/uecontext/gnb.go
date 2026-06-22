package uecontext

import (
	"stormsim/internal/common/fsm"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
	"time"

	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
)

func (ue *UeContext) handleGnbMsg(msg rlink.Message) {
	if msg.GetType() == model.NasMsgType {
		message := msg.(*model.NasMsg)
		ue.sendEventMm(fsm.NewEventData(model.GmmMessageEvent, &message.Nas))
	} else if msg.GetType() == model.RlinkSetupPagingType {
		message := msg.(*model.RlinkSetupPaging)
		for _, pagedUE := range message.PagedUEs {
			if ue.guti != nil && pagedUE.FiveGSTMSI != nil &&
				[4]uint8(pagedUE.FiveGSTMSI.FiveGTMSI) == ue.getTmsiBytes() {
				ue.sendEventMm(fsm.NewEmptyEventData(model.ServiceRequestInit))
				return
			}
		}
	} else if msg.GetType() == model.RlinkSetupPduSessonCommandType {
		// Path switch completed on target gNB; stop T304 if still running.
		_ = ue.timerEngine.Stop(timer.T304, int64(ue.id))
		if ue.tunnelMode == model.TunnelDisabled {
			// ue.Warn("[GTP]Interface has not been created: tunnel has been disabled")
		} else { // setup pdu session
			ue.setupGtpInterface(msg.(*model.RlinkSetupPduSessonCommand))
		}
	} else if msg.GetType() == model.RLinkHandoverPrepareRequestType {
		message := msg.(*model.RLinkHandoverPrepareRequest)
		if message.Conn != nil && message.TargetGnbId != "" {
			// Ignore duplicate HO command (retry from gNB)
			if ue.gnbId == message.TargetGnbId {
				ue.Debug("Already on target gNB %s, ignoring duplicate HO command", message.TargetGnbId)
				return
			}
			ue.Info("gNodeB-%s is asking to use another gNodeB", ue.gnbId)

			// Start T304: Handover execution timer (RRCReconfiguration with reconfigurationWithSync)
			hoAborted := false

			ue.timerEngine.CreateTimer(timer.TimerConfig{
				TimerType: timer.T304,
				Duration:  timer.T304_duration,
				CountMax:  1,
				ExpireFunc: func() {
					hoAborted = true
					ue.Warn("T304 expired: handover failure, initiating RRC re-establishment")
					// Trigger T311: cell selection timer
					ue.timerEngine.CreateTimer(timer.TimerConfig{
						TimerType: timer.T311,
						Duration:  timer.T311_duration,
						CountMax:  1,
						ExpireFunc: func() {
							ue.Warn("T311 expired: cell selection failed, starting RRC re-establishment (T301)")
							// Start T301: RRC re-establishment timer
							ue.timerEngine.CreateTimer(timer.TimerConfig{
								TimerType: timer.T301,
								Duration:  timer.T301_duration,
								CountMax:  1,
								ExpireFunc: func() {
									ue.Error("T301 expired: RRC re-establishment failed, entering RRC_IDLE")
									ue.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.Terminate))
								},
							}, int64(ue.id))
							ue.timerEngine.Start(timer.T301, int64(ue.id))
						},
					}, int64(ue.id))
					ue.timerEngine.Start(timer.T311, int64(ue.id))
				},
			}, int64(ue.id))
			ue.timerEngine.Start(timer.T304, int64(ue.id))

			// Simulate HO execution delay (random access + RRC signaling)
			if message.Conn.DownlinkDelay > 0 {
				time.Sleep(message.Conn.DownlinkDelay)
			}

			if hoAborted {
				ue.Warn("T304 already expired, aborting handover to %s", message.TargetGnbId)
				// Stop T304 with cancel event for telemetry
				ue.timerEngine.StopWithEvent(timer.T304, int64(ue.id), "cancel")
				return
			}

			ue.sendGnb(&model.RlinkRlinkHandoverPrepareResponse{
				PrUeId:       int64(ue.id),
				IsXnHandover: message.IsXnHandover,
				IsN2Handover: message.IsN2Handover,
			})
			// ensure flushing
			time.Sleep(10 * time.Millisecond)
			ue.rlinkConn.Close()

			ue.gnbId = message.TargetGnbId
			ue.rlinkConn = message.Conn

			// Stop T304: handover execution successful (connected to target gNB)
			ue.timerEngine.Stop(timer.T304, int64(ue.id))
			ue.Info("T304 stopped: handover to gNB %s successful", ue.gnbId)
		}
	} else if msg.GetType() == model.CHOConfigType {
		config := msg.(*model.CHOConfig)
		ue.Info("Received CHOConfig with %d candidates", len(config.Candidates))

		ue.mutex.Lock()
		ue.choCandidates = config.Candidates
		ue.choConditions = config.Conditions
		ue.mutex.Unlock()

		if config.ServingRsrp != 0 {
			ue.UpdateRsrpValues(map[string]float64{ue.gnbId: config.ServingRsrp})
		}

		ue.triggerMeasurementReport()
		ue.startChoEvaluator()
	} else {
		ue.Error("Received unknown message from gNodeB: %v", msg)
	}
}

func (ue *UeContext) sendGnb(message rlink.Message) {
	ue.mutex.Lock()
	ue.rlinkConn.SendUplink(message)
	ue.mutex.Unlock()
}

func (ue *UeContext) sendN1Sm(
	n1Sm []byte,
	sessionId uint8,
	requestType *uint8,
	params *map[string]any,
) {
	msg := &nas.UlNasTransport{
		PayloadContainer:     n1Sm,
		PayloadContainerType: nas.PayloadContainerTypeN1SMInfo,
		PduSessionId:         &sessionId,
		SNssai:               new(nas.SNssai),
	}
	if requestType != nil {
		msg.RequestType = requestType
	}

	if val, ok := (*params)["dnn"]; ok && val != "" {
		msg.Dnn = nas.NewDnn(val.(string))
	} else if len(ue.dnn) > 0 {
		msg.Dnn = nas.NewDnn(ue.dnn)
	}

	if val, ok := (*params)["snssai"]; ok && val != nil {
		if s, ok := val.(models.Snssai); ok {
			msg.SNssai.Set(uint8(s.Sst), s.Sd)
		}
	} else {
		msg.SNssai.Set(uint8(ue.snssai.Sst), ue.snssai.Sd)
	}

	nasCtx := ue.getNasContext() //must be non nil
	msg.SetSecurityHeader(nas.NasSecBoth)
	if nasPdu, err := nas.EncodeMm(nasCtx, msg); err != nil {
		ue.Fatal("Error send N1 Sm: ul nas transport: %v", err)
	} else {
		ue.Info("send n1sm msg to amf")
		ue.sendNas(nasPdu) // sending to GNB
	}
}

func (ue *UeContext) sendNas(nasPdu []byte) {
	if ue.enableFuzz {
		Capture.CaptureMsgFromUe(ue.state_mm.CurrentState(), nasPdu)
	}
	ue.sendGnb(&model.NasMsg{PrUeId: int64(ue.id), Nas: nasPdu})
}
