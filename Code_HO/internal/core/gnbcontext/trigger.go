package gnbcontext

import (
	"bytes"
	"encoding/binary"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
	"time"

	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/ngap/utils"
)

func TriggerReleaseUe(gnb *GnbContext, msin string) error {
	ue, err := gnb.getGnbUeByMsin(msin)
	if err != nil || ue == nil {
		gnb.Error("Cannot trigger release ue %s: %s", msin, err.Error())
		return err
	}
	gnb.sendUeContextReleaseRequest(ue, &ies.Cause{
		Choice: ies.CausePresentRadionetwork, RadioNetwork: &ies.CauseRadioNetwork{
			Value: ies.CauseRadioNetworkUnspecified,
		},
	})

	return nil
}

func (gnb *GnbContext) GeListUeInfo() []*GnbUeContext {
	return nil
}

func (gnb *GnbContext) sendHandoverNotify(ue *GnbUeContext) {
	gnb.Info("Initiating Handover Notify")
	PLMNIdentity := gnb.getPLMNIdentityInBytes()
	NRCellIdentity := gnb.getNRCellIdentity()
	TAC := gnb.getTacInBytes()

	msg := &ies.HandoverNotify{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
		UserLocationInformation: ies.UserLocationInformation{
			Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
			UserLocationInformationNR: &ies.UserLocationInformationNR{
				NRCGI: ies.NRCGI{
					PLMNIdentity:   PLMNIdentity,
					NRCellIdentity: NRCellIdentity,
				},
				TAI: ies.TAI{
					PLMNIdentity: PLMNIdentity,
					TAC:          TAC,
				},
			},
		},
	}

	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create Handover Notify: ", err)
		return
	}

	gnb.LogNgapSend("HandoverNotify")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending Handover Notify: ", err)
	}

	// Notify FSM: N2 handover complete (target notified AMF)
	if HandoverStateCallback != nil {
		HandoverStateCallback(ue.prUeId, model.HO_CompleteEvent)
	}
}

func (gnb *GnbContext) sendPathSwitchRequest(ue *GnbUeContext) {
	gnb.Info("Initiating Path Switch Request")
	pduSessions := ue.context.PduSession

	secCap := ue.context.UeSecurityCapabilities
	if secCap == nil {
		gnb.Warn("UE security capabilities are nil, using default for testing Path Switch Request Failure handler")
		// NEA0, NEA1, NEA2 and NIA0, NIA1, NIA2
		secCap = &ies.UESecurityCapabilities{
			NRencryptionAlgorithms:             aper.BitString{Bytes: []byte{0xe0, 0x00}, NumBits: 16},
			NRintegrityProtectionAlgorithms:    aper.BitString{Bytes: []byte{0xe0, 0x00}, NumBits: 16},
			EUTRAencryptionAlgorithms:          aper.BitString{Bytes: []byte{0xe0, 0x00}, NumBits: 16},
			EUTRAintegrityProtectionAlgorithms: aper.BitString{Bytes: []byte{0xe0, 0x00}, NumBits: 16},
		}
	}

	msg := &ies.PathSwitchRequest{
		SourceAMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID:       ue.ranUeNgapId,
		UserLocationInformation: ies.UserLocationInformation{
			Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
			UserLocationInformationNR: &ies.UserLocationInformationNR{
				NRCGI: ies.NRCGI{
					PLMNIdentity:   gnb.getPLMNIdentityInBytes(),
					NRCellIdentity: gnb.getNRCellIdentity(),
				},
				TAI: ies.TAI{
					PLMNIdentity: gnb.getPLMNIdentityInBytes(),
					TAC:          gnb.getTacInBytes(),
				},
			},
		},
		UESecurityCapabilities:               *secCap,
		PDUSessionResourceToBeSwitchedDLList: []ies.PDUSessionResourceToBeSwitchedDLItem{},
	}

	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		ip := utils.IPAddressToNgap(gnb.dataPlaneInfo.gnbIp, "")
		// Allocate a NEW downlink TEID on the target gNB for the UPF to use
		newDlTeid := gnb.getUeTeid(ue)
		pduSession.DownlinkTeid = newDlTeid
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, newDlTeid)
		transfer := ies.PathSwitchRequestTransfer{
			DLNGUUPTNLInformation: ies.UPTransportLayerInformation{
				Choice: ies.UPTransportLayerInformationPresentGtptunnel,
				GTPTunnel: &ies.GTPTunnel{
					TransportLayerAddress: ip,
					GTPTEID:               buf.Bytes(),
				}},
			DLNGUTNLInformationReused:    nil,
			UserPlaneSecurityInformation: nil,
			QosFlowAcceptedList: []ies.QosFlowAcceptedItem{
				{QosFlowIdentifier: pduSession.QosId},
			},
		}

		var b []byte
		var err error
		if b, err = transfer.Encode(); err != nil {
			gnb.Info("Error encoding Path Switch Request ", err)
			return
		}
		msg.PDUSessionResourceToBeSwitchedDLList = append(
			msg.PDUSessionResourceToBeSwitchedDLList,
			ies.PDUSessionResourceToBeSwitchedDLItem{
				PDUSessionID:              pduSession.PduSessionId,
				PathSwitchRequestTransfer: b,
			},
		)
	}

	if len(msg.PDUSessionResourceToBeSwitchedDLList) == 0 {
		gnb.Warn("No PDU Session to handover: Xn Handover requires at least 1 PDU Session")
		msg.PDUSessionResourceToBeSwitchedDLList = nil
	}

	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create Path Switch Request ", err)
		return
	}
	gnb.LogNgapSend("PathSwitchRequest")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending Path Switch Request: ", err)
	}
}

func (gnb *GnbContext) sendUeContextReleaseRequest(ue *GnbUeContext, cause *ies.Cause) {
	gnb.Info("Initiating UE Context Release Request")
	msg := &ies.UEContextReleaseRequest{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
		Cause:       *cause,
	}

	activePduSession := []*model.GnbPDUSessionContext{}
	pduSessions := ue.context.PduSession
	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		activePduSession = append(activePduSession, pduSession)
	}

	if len(activePduSession) > 0 {
		msg.PDUSessionResourceListCxtRelReq = make([]ies.PDUSessionResourceItemCxtRelReq, len(activePduSession))

		// PDU Session Resource Item in PDU session Resource List
		for _, pduSessionID := range activePduSession {
			id := pduSessionID.PduSessionId
			msg.PDUSessionResourceListCxtRelReq = append(msg.PDUSessionResourceListCxtRelReq, ies.PDUSessionResourceItemCxtRelReq{
				PDUSessionID: id,
			})
		}
	}

	ngapPdu, err := ngap.NgapEncode(msg)
	if err != nil {
		gnb.Error("Error create UE Context Release Request: %s", err.Error())
		return
	}

	gnb.LogNgapSend("UEContextReleaseRequest")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending UE Context Release Request: %s", err.Error())
	}
}

func (gnb *GnbContext) sendAmfConfigurationUpdateAcknowledge(amf *GnbAmfContext) {
	gnb.Info("Initiating AMF Configuration Update Acknowledge")
	message := ies.AMFConfigurationUpdateAcknowledge{}

	ngapPdu, err := ngap.NgapEncode(&message)
	if err != nil {
		gnb.Warn("Error sending AMF Configuration Update Acknowledge: ", err)
	}

	gnb.LogNgapSend("AMFConfigurationUpdateAcknowledge")
	amf.sendNgap(ngapPdu)
	if err != nil {
		gnb.Warn("Error sending AMF Configuration Update Acknowledge: ", err)
	}
}

func (gnb *GnbContext) sendNgSetupRequest(amf *GnbAmfContext) {
	gnb.Info("Initiating NG Setup Request")

	msg := ies.NGSetupRequest{}

	msg.GlobalRANNodeID = ies.GlobalRANNodeID{
		Choice: ies.GlobalRANNodeIDPresentGlobalgnbId,
		GlobalGNBID: &ies.GlobalGNBID{
			PLMNIdentity: gnb.getPlmnInOctets(),
			GNBID: ies.GNBID{
				Choice: ies.GNBIDPresentGnbId,
				GNBID: &aper.BitString{
					Bytes:   gnb.getGnbIdInBytes(),
					NumBits: 24,
				},
			},
		},
	}

	msg.RANNodeName = []byte("StormSim")

	sst, sd := gnb.getSliceInBytes()
	msg.SupportedTAList = []ies.SupportedTAItem{
		{
			TAC: gnb.getTacInBytes(),
			BroadcastPLMNList: []ies.BroadcastPLMNItem{
				{
					PLMNIdentity: gnb.getPlmnInOctets(),
					TAISliceSupportList: []ies.SliceSupportItem{
						{SNSSAI: ies.SNSSAI{SST: sst, SD: sd}},
					},
				},
			},
		},
	}

	msg.DefaultPagingDRX = ies.PagingDRX{Value: ies.PagingDRXV128}

	ngapPdu, err := ngap.NgapEncode(&msg)

	if err != nil {
		gnb.Error("Error sending NG Setup Request: ", err)
	}

	gnb.LogNgapSend("NGSetupRequest")
	amf.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending NG Setup Request: ", err)
	}

}

// stopXnRelocTimersOnSource stops TXnRELOC timers on the source gNB. These timers
// are created on the source during Xn handover but path switch completion is
// handled on the target gNB.
func stopXnRelocTimersOnSource(sourceGnbId string, prUeId int64) {
	if sourceGnbId == "" {
		return
	}
	val, ok := Gnbs.Load(sourceGnbId)
	if !ok {
		return
	}
	sourceGnb := val.(*GnbContext)
	_ = sourceGnb.timerEngine.Stop(timer.TXnRELOCprep, prUeId)
	_ = sourceGnb.timerEngine.Stop(timer.TXnRELOCoverall, prUeId)
}

func TriggerXnHandover(oldGnb *GnbContext, newGnb *GnbContext, prUeId int64, loss float64, delay, jitter time.Duration) {
	// Apply simulation parameters to target gNB
	newGnb.controlPlaneInfo.xnPacketLoss = loss
	newGnb.controlPlaneInfo.xnLatencyMs = int(delay.Milliseconds())
	newGnb.controlPlaneInfo.xnJitterMs = int(jitter.Milliseconds())

	oldGnb.Info("Initiating Real Xn UE Handover gnbid=%s mod=gnb", oldGnb.controlPlaneInfo.gnbId)
	newGnb.SetXnSimParams(loss, int(delay.Milliseconds()), int(jitter.Milliseconds()))

	ue, err := oldGnb.GetGnbUeByPrUeId(prUeId)
	if err != nil {
		poolSize := 0
		oldGnb.prUeIdPool.Range(func(k, v interface{}) bool {
			poolSize++
			return true
		})
		oldGnb.Error("Error getting UE from PR UE ID: %v (Pool size: %d)", err, poolSize)
		return
	}

	// Mark UE as currently performing a handover on this gNB (guard)
	oldGnb.ueHoStatusPool.Store(prUeId, true)

	conn := rlink.NewConnection(
		ue.prUeId,
		ue.msin,
		newGnb.controlPlaneInfo.gnbId,
		rlink.DefaultBufferSize,
		rlink.DefaultDuration,
		1,
	)
	// Set handover execution delay > T304_duration so T304 times out
	// during the simulated random access + RRC signaling phase.
	conn.DownlinkDelay = timer.T304_duration + 500*time.Millisecond
	// Apply radio loss params to the new connection for consistency
	conn.DownlinkLoss = oldGnb.controlPlaneInfo.radioLoss
	conn.UplinkLoss = oldGnb.controlPlaneInfo.radioLoss

	// Start Xn relocation timers before any message can trigger a fast UE
	// response. Starting them after the HO command can miss Stop() and leave
	// stale timers that later emit false timeouts.
	oldGnb.timerEngine.CreateTimer(timer.TimerConfig{
		TimerType: timer.TXnRELOCprep,
		Duration:  timer.TXnRELOCprep_duration,
		CountMax:  1,
		ExpireFunc: func() {
			oldGnb.Warn("TXnRELOCprep expired: Xn handover cancelled for UE %d", ue.prUeId)
			ue.handoverTargetGnb = nil
			oldGnb.timerEngine.StopWithEvent(timer.TXnRELOCoverall, prUeId, "cancel")
			oldGnb.ueHoStatusPool.Delete(prUeId)
			SendHoEvent(prUeId, model.HoFailEvent, nil)
		},
	}, prUeId)
	oldGnb.timerEngine.Start(timer.TXnRELOCprep, prUeId)

	oldGnb.timerEngine.CreateTimer(timer.TimerConfig{
		TimerType: timer.TXnRELOCoverall,
		Duration:  timer.TXnRELOCoverall_duration,
		CountMax:  1,
		ExpireFunc: func() {
			if ue.state == UE_DOWN {
				oldGnb.Info("TXnRELOCoverall expired: cleaning up successful handover context for UE %d", ue.prUeId)
				oldGnb.deleteGnBUe(ue)
			} else {
				oldGnb.Warn("TXnRELOCoverall expired: handover failed/cancelled for UE %d, keeping context", ue.prUeId)
				oldGnb.ueHoStatusPool.Delete(prUeId)
				SendHoEvent(prUeId, model.HoFailEvent, nil)
			}
		},
	}, prUeId)
	oldGnb.timerEngine.Start(timer.TXnRELOCoverall, prUeId)

	forwardMsg := &model.RLinkHandoverForwardUeContext{
		PrUeId:        ue.prUeId,
		Msin:          ue.msin,
		Conn:          conn,
		AmfUeNgapId:   ue.amfUeNgapId,
		UeCoreContext: &ue.context,
		SourceGnbId:   oldGnb.controlPlaneInfo.gnbId,
	}

	// Retry mechanism (up to 3 attempts) for Xn handover context forwarding
	var sendErr error
	for i := 0; i < 3; i++ {
		sendErr = SendToGnb(oldGnb.controlPlaneInfo.gnbId, forwardMsg, newGnb.controlPlaneInfo.gnbId, true)
		if sendErr == nil {
			if i > 0 {
				oldGnb.Info("Handover Forward succeeded after %d retries", i)
			}
			break
		}
		oldGnb.Warn("Handover Forward attempt %d failed: %v. Retrying...", i+1, sendErr)
		time.Sleep(50 * time.Millisecond) // short wait before retry
	}

	if sendErr != nil {
		oldGnb.Warn("Failed to forward UE context to target gNB after 3 attempts: %v", sendErr)

		// Fire transport_fail marker event so the log clearly separates
		// "transport failure" (~150ms from 3x50ms retries) from natural
		// "timer timeout" (~3000ms from TXnRELOCprep). The prep timer is
		// deliberately NOT cancelled so it can be observed timing out naturally.
		oldGnb.timerEngine.FireEvent(timer.TXnRELOCprep, prUeId, "transport_fail", 0)

		// Stop overall timer (prep timer stays running for natural timeout)
		oldGnb.timerEngine.StopWithEvent(timer.TXnRELOCoverall, prUeId, "cancel")

		oldGnb.ueHoStatusPool.Delete(prUeId)
		SendHoEvent(prUeId, model.HoFailEvent, nil)
		return
	}
	oldGnb.Info("Forwarded UE Context to target gNB %s gnbid=%s mod=gnb", newGnb.controlPlaneInfo.gnbId, oldGnb.controlPlaneInfo.gnbId)

	ue.sendCriticalMsgToUe(&model.RLinkHandoverPrepareRequest{
		PrUeId:       ue.prUeId,
		Conn:         conn,
		TargetGnbId:  newGnb.controlPlaneInfo.gnbId,
		IsXnHandover: true,
		IsN2Handover: false,
	})

	SendHoEvent(prUeId, model.HoRlinkPrepareReqSentEvent, map[string]interface{}{
		"SourceGnbId": oldGnb.GetId(),
		"TargetGnbId": newGnb.GetId(),
	})
}

func TriggerNgapHandover(sourceGnb *GnbContext, targetGnb *GnbContext, prUeId int64, loss float64, delay, jitter time.Duration) {
	sourceGnb.Info("Initiating NGAP UE Handover")

	ue, err := sourceGnb.GetGnbUeByPrUeId(prUeId)
	if err != nil {
		sourceGnb.Error("Error getting UE from PR UE ID: %s", err.Error())
		return
	}
	pduSessions := ue.context.PduSession
	PLMNIdentity := targetGnb.getPLMNIdentityInBytes()
	TAC := targetGnb.getTacInBytes()
	transfer := getSourceToTargetTransparentTransfer(sourceGnb, targetGnb, pduSessions, ue.prUeId)

	msg := &ies.HandoverRequired{
		AMFUENGAPID:  ue.amfUeNgapId,
		RANUENGAPID:  ue.ranUeNgapId,
		HandoverType: ies.HandoverType{Value: ies.HandoverTypeIntra5Gs},
		Cause: ies.Cause{
			Choice:       ies.CausePresentRadionetwork,
			RadioNetwork: &ies.CauseRadioNetwork{Value: ies.CauseRadioNetworkHandoverdesirableforradioreason},
		},
		PDUSessionResourceListHORqd: make([]ies.PDUSessionResourceItemHORqd, len(pduSessions)),
		TargetID: ies.TargetID{
			Choice: ies.TargetIDPresentTargetrannodeid,
			TargetRANNodeID: &ies.TargetRANNodeID{
				GlobalRANNodeID: ies.GlobalRANNodeID{
					Choice: ies.GlobalRANNodeIDPresentGlobalgnbId,
					GlobalGNBID: &ies.GlobalGNBID{
						PLMNIdentity: PLMNIdentity,
						GNBID: ies.GNBID{
							Choice: ies.GNBIDPresentGnbId,
							GNBID: &aper.BitString{
								Bytes:   targetGnb.getGnbIdInBytes(),
								NumBits: uint64(len(targetGnb.getGnbIdInBytes()) * 8),
							},
						},
					},
				},
				SelectedTAI: ies.TAI{
					PLMNIdentity: PLMNIdentity,
					TAC:          TAC,
				},
			},
		},
		SourceToTargetTransparentContainer: transfer,
	}

	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		//PDU SessionResource Admittedy Item
		PDUSessionID := pduSession.PduSessionId

		transfer := ies.HandoverRequiredTransfer{}
		var buf []byte
		var err error
		if buf, err = transfer.Encode(); err != nil {
			sourceGnb.Warn("err encode HandoverRequiredBuilder <- HandoverRequiredTransfer ")
		}

		msg.PDUSessionResourceListHORqd = append(msg.PDUSessionResourceListHORqd,
			ies.PDUSessionResourceItemHORqd{
				PDUSessionID:             PDUSessionID,
				HandoverRequiredTransfer: buf,
			})
	}

	if len(msg.PDUSessionResourceListHORqd) == 0 {
		sourceGnb.Error("No PDU Session to set up in InitialContextSetupResponse. NGAP Handover requires at least a PDU Session.")
	}

	ue.handoverTargetGnb = targetGnb
	ngapPdu, err := ngap.NgapEncode(msg)
	if err != nil {
		sourceGnb.Info("Error sending Handover Required: %s", err.Error())
	}

	sourceGnb.LogNgapSend("HandoverRequired")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		sourceGnb.Error("Error sending Handover Required: %s", err.Error())
	}

	// Notify FSM: N2 handover preparation started
	if HandoverStateCallback != nil {
		HandoverStateCallback(ue.prUeId, model.HO_PrepareEvent)
	}

	// Start TNGRELOCprep: N2 handover preparation timer
	sourceGnb.timerEngine.CreateTimer(timer.TimerConfig{
		TimerType: timer.TNGRELOCprep,
		Duration:  timer.TNGRELOCprep_duration,
		CountMax:  1,
		ExpireFunc: func() {
			sourceGnb.Warn("TNGRELOCprep expired: N2 handover cancelled for UE %d", ue.prUeId)
			ue.handoverTargetGnb = nil
			sourceGnb.timerEngine.Stop(timer.TNGRELOCoverall, prUeId)
			if HandoverStateCallback != nil {
				HandoverStateCallback(ue.prUeId, model.TNGRELOCprepExpiredEvent)
			}
		},
	}, prUeId)
	sourceGnb.timerEngine.Start(timer.TNGRELOCprep, prUeId)

	// Start TNGRELOCoverall: N2 handover overall timer
	sourceGnb.timerEngine.CreateTimer(timer.TimerConfig{
		TimerType: timer.TNGRELOCoverall,
		Duration:  timer.TNGRELOCoverall_duration,
		CountMax:  1,
		ExpireFunc: func() {
			if ue.state == UE_DOWN {
				sourceGnb.Info("TNGRELOCoverall expired: cleaning up successful handover context for UE %d", ue.prUeId)
				sourceGnb.deleteGnBUe(ue)
			} else {
				sourceGnb.Warn("TNGRELOCoverall expired: handover failed/cancelled for UE %d, keeping context", ue.prUeId)
				if HandoverStateCallback != nil {
					HandoverStateCallback(ue.prUeId, model.TNGRELOCoverallExpiredEvent)
				}
			}
		},
	}, prUeId)
	sourceGnb.timerEngine.Start(timer.TNGRELOCoverall, prUeId)
}
