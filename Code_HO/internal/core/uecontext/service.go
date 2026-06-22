package uecontext

import (
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
	"time"
)

func (ue *UeContext) connectGnb(gnbId string) {
	ue.gnbId = gnbId
	ue.wg.Add(1)
	go ue.listenGnb()
}

func (ue *UeContext) listenGnb() {
	defer ue.wg.Done()

	ue.initConn() // starting communication with GNB and listen.

	// Block until a signal is received from gnb
	for {
		select {
		case <-ue.ctx.Done():
			return
		case msg, open := <-ue.rlinkConn.GetDownlinkChan():
			if !open {
				ue.Warn("Stopping UE as communication with gNB was closed")
				ue.rlinkConn.Close()
				//logger.RLinkConnStats[ue.msin].MessageReceivedDropped.Add(1)
				return
			}
			//logger.RLinkConnStats[ue.msin].MessageReceived.Add(1)
			ue.handleGnbMsg(msg)
		case <-ue.getDRX():
			ue.verifyPaging()
		}

		if ue.state_mm.CurrentState() == model.Deregistered {
			break
		}
	}
}

func (ue *UeContext) verifyPaging() {
	gnbcontext.SendToGnb(ue.msin, &model.RlinkUeReadyPaging{
		PrUeId:        int64(ue.id),
		Conn:          ue.rlinkConn,
		FetchPagedUEs: true,
	}, ue.gnbId, false)
}

func (ue *UeContext) initConn() {
	conn := rlink.NewConnection(
		int64(ue.id),
		ue.msin,
		ue.gnbId,
		rlink.DefaultBufferSize,
		rlink.DefaultDuration,
		1,
	)
	ue.rlinkConn = conn

	ue.Info("UE %d: Sending RLinkCreateConnectionRequest to gNB %s", ue.id, ue.gnbId)
	gnbcontext.SendToGnb(ue.msin, &model.RLinkCreateConnectionRequest{
		PrUeId: int64(ue.id), Msin: ue.msin, Tmsi: ue.guti, Conn: conn}, ue.gnbId, false)

	select {
	case msg, ok := <-ue.rlinkConn.GetDownlinkChan():
		if !ok {
			ue.Error("UE %d: RLink Downlink channel closed", ue.id)
			return
		}
		if message, ok := msg.(*model.RLinkCreateConnectionResponse); ok {
			ue.Info("UE %d: Connection established with gNB %s", ue.id, ue.gnbId)
			ue.auth.snn = []byte(deriveSNN(message.Mcc, message.Mnc))
		} else {
			ue.Error("UE %d: Received unexpected message type: %T", ue.id, msg)
		}
	case <-time.After(5 * time.Second):
		ue.Error("UE %d: Timeout waiting for CreateConnectionResponse from gNB %s", ue.id, ue.gnbId)
	}
}
