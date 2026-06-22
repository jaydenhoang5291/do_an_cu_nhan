package gnbcontext

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"stormsim/internal/common/logger"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/internal/transport/rlink"
	"stormsim/internal/transport/sctpngap"
	"stormsim/pkg/model"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/ngap/utils"
	"github.com/reogac/nas"
	gtpv1 "github.com/wmnsk/go-gtp/gtpv1"
)

// HandoverStateCallback enables reporting HO FSM events back to the HandoverManager
// without creating an import cycle.
var HandoverStateCallback func(prUeId int64, eventType model.EventType)

type GnbContext struct {
	*logger.BufferedLogger
	dataPlaneInfo      DataInfo    // gnb data plane information
	controlPlaneInfo   ControlInfo // gnb control plane information
	msinPool           sync.Map    // map[string]*GnbUeContext, Msin as key
	ranUePool          sync.Map    // map[in64]*GnbUeContext, UeRanNgapId as key
	prUeIdPool         sync.Map    // map[in64]*GnbUeContext, PrUeId as key
	downlinkTeidPool   sync.Map    // map[uint32]*GnbUeContext, downlinkTeid as key
	amfPool            sync.Map    // map[int64]*GNBAmf, AmfId as key
	sliceConfiguration Slice
	ranUeIdGenerator   int64  // ran UE id.
	amfIdGenerator     int64  // ran amf id
	teidGenerator      uint32 // ran UE downlink Teid
	ueIpGenerator      uint8  // ran ue ip.
	pagedUEs           []model.PagedUE
	pagedUELock        sync.Mutex

	// check ue
	ueHoStatusPool sync.Map // map[int64]*bool, PrUeId as key, state is handover status

	// stats
	connCount int // connection counter for unique port assignment
	connMutex sync.Mutex

	// delay tracking for NGAP messages
	delayTracker *logger.DelayTracker

	// timer engine for Xn/N2 handover timers
	timerEngine *timer.TimerEngine

	// check
	ctx     context.Context
	mu      sync.Mutex
	isReady chan bool
	close   chan struct{}
}

type DataInfo struct {
	gnbIp        string            // gnb ip for data plane.
	gnbPort      int               // gnb port for data plane.
	upfIp        string            // upf ip
	upfPort      int               // upf port
	gtpPlane     *gtpv1.UPlaneConn // N3 connection
	gatewayGnbIp string            // IP gateway that communicates with UE data plane.
}

type Slice struct {
	sd  string
	sst string
}

type ControlInfo struct {
	mcc            string
	mnc            string
	tac            string
	gnbId          string
	gnbIp          string
	gnbPort        int
	inboundChannel chan rlink.Message
	rlinkPool      sync.Map
	n2             *sctpngap.SctpConn
	xnPacketLoss   float64
	xnLatencyMs    int
	xnJitterMs     int
	radioLoss      float64
	radioLatencyMs int
	radioJitterMs  int
}

func (gnb *GnbContext) newRanGnbContext(gnbId, mcc, mnc, tac, sst, sd, ip, ipData string, port, portData int, logBufferSize int) {
	gnb.controlPlaneInfo.mcc = mcc
	gnb.controlPlaneInfo.mnc = mnc
	gnb.controlPlaneInfo.tac = tac
	gnb.controlPlaneInfo.gnbId = gnbId
	gnb.controlPlaneInfo.inboundChannel = make(chan rlink.Message, 1000)
	gnb.sliceConfiguration.sd = sd
	gnb.sliceConfiguration.sst = sst
	gnb.ranUeIdGenerator = 1
	gnb.amfIdGenerator = 1
	gnb.controlPlaneInfo.gnbIp = ip
	gnb.teidGenerator = 1
	gnb.ueIpGenerator = 3
	gnb.controlPlaneInfo.gnbPort = port
	gnb.dataPlaneInfo.upfPort = 2152
	gnb.dataPlaneInfo.gtpPlane = nil
	gnb.dataPlaneInfo.gatewayGnbIp = "127.0.0.2"
	gnb.dataPlaneInfo.upfIp = ""
	gnb.dataPlaneInfo.gnbIp = ipData
	gnb.dataPlaneInfo.gnbPort = portData
	gnb.BufferedLogger = logger.NewBufferedLogger(
		logBufferSize,
		"gnb",
		gnbId,
		map[string]string{"mod": "gnb", "gnbid": gnbId},
		func() string { return "RUNNING" },
	)
	gnb.delayTracker = logger.NewDelayTracker(100) // NGAP delay tracking

	// init timer engine
	gnb.timerEngine = timer.NewTimerEngine(gnb.ctx)

	// check
	gnb.isReady = make(chan bool)
	gnb.close = make(chan struct{})
}

func (gnb *GnbContext) newGnBUe(conn *rlink.Connection, prUeId int64, msin string, tmsi *nas.Guti) (*GnbUeContext, error) {
	gnb.mu.Lock()
	defer gnb.mu.Unlock()

	//TODO: if necessary add more information for UE.
	ue := &GnbUeContext{
		// Connect gNB and UE's channels
		rlinkConn: conn,
		prUeId:    prUeId,
		msin:      msin,
		tmsi:      tmsi,

		// set state to UE.
		state: UE_INITIALIZED,

		// Logegr
		Logger: logger.InitLogger("", map[string]string{
			"mod": "gnbue", "gnb": gnb.controlPlaneInfo.gnbId, "msin": msin}),
	}

	// set ran UE Ngap Id.
	ranId := gnb.getRanUeId()
	ue.ranUeNgapId = ranId
	ue.amfUeNgapId = 0

	// store UE in the UE Pool of GNB.
	gnb.ranUePool.Store(ranId, ue)
	gnb.prUeIdPool.Store(prUeId, ue)
	gnb.msinPool.Store(msin, ue)

	// store rlink connection
	gnb.controlPlaneInfo.rlinkPool.Store(rlink.ConnectionKey(prUeId, gnb.dataPlaneInfo.gnbIp), conn)

	// apply current radio params to the new connection
	conn.SetRadioParams(gnb.controlPlaneInfo.radioLoss, gnb.controlPlaneInfo.radioLatencyMs, gnb.controlPlaneInfo.radioJitterMs)

	// select AMF with Capacity is more than 0.
	amf := gnb.selectAmfByActive()
	if amf == nil {
		return nil, fmt.Errorf("No AMF available for this UE")
	}

	// set amfId and SCTP association for UE.
	ue.amfId = amf.amfId
	ue.sctpConnection = amf.tlnaAssoc.sctpConn

	gnb.Info("Added UE %s to pool (ranUeNgapId: %d, prUeId: %d) gnbid=%s mod=gnb", msin, ranId, prUeId, gnb.controlPlaneInfo.gnbId)
	return ue, nil
}

func (gnb *GnbContext) removeConnection(key string) {
	if conn, exists := gnb.controlPlaneInfo.rlinkPool.Load(key); exists {
		conn.(*rlink.Connection).Close()
		gnb.controlPlaneInfo.rlinkPool.Delete(key)
	}
}

func (gnb *GnbContext) deleteGnBUe(ue *GnbUeContext) {
	ue.lock.Lock()
	// Clean up any active timers related to this UE to prevent zombie timers
	if gnb.timerEngine != nil {
		// Best-effort: try to stop/remove common handover timers
		_ = gnb.timerEngine.RemoveTimer(timer.TXnRELOCprep, ue.prUeId)
		_ = gnb.timerEngine.RemoveTimer(timer.TXnRELOCoverall, ue.prUeId)
		_ = gnb.timerEngine.RemoveTimer(timer.T304, ue.prUeId)
	}

	gnb.ranUePool.Delete(ue.ranUeNgapId)
	gnb.msinPool.Delete(ue.msin)
	gnb.prUeIdPool.CompareAndDelete(ue.prUeId, ue)
	gnb.ueHoStatusPool.Delete(ue.prUeId)
	for _, pduSession := range ue.context.PduSession {
		if pduSession != nil {
			gnb.downlinkTeidPool.Delete(pduSession.DownlinkTeid)
		}
	}
	// ue.rlinkConn.Close() // CRITICAL: Do NOT close connection here, it will disconnect UE from ALL gNBs
	ue.lock.Unlock()
	gnb.Warn("Removed UE %s from pool (State: %v) gnbid=%s mod=gnb", ue.msin, ue.state, gnb.controlPlaneInfo.gnbId)
}

func (gnb *GnbContext) ResetUeContext(pRUeId int64) {
	if ue, ok := gnb.prUeIdPool.Load(pRUeId); ok {
		gnb.deleteGnBUe(ue.(*GnbUeContext))
		gnb.Info("Reset stale UE context for pRUeId-%d gnbid=%s mod=gnb", pRUeId, gnb.controlPlaneInfo.gnbId)
	}
}

func (gnb *GnbContext) SetXnSimParams(loss float64, latencyMs, jitterMs int) {
	gnb.controlPlaneInfo.xnPacketLoss = loss
	gnb.controlPlaneInfo.xnLatencyMs = latencyMs
	gnb.controlPlaneInfo.xnJitterMs = jitterMs
}

func (gnb *GnbContext) SetRadioLoss(loss float64) {
	gnb.controlPlaneInfo.radioLoss = loss
	gnb.controlPlaneInfo.rlinkPool.Range(func(key, value any) bool {
		if conn, ok := value.(*rlink.Connection); ok {
			conn.SetLoss(loss)
		}
		return true
	})
}

func (gnb *GnbContext) SetRadioParams(loss float64, latencyMs, jitterMs int) {
	gnb.controlPlaneInfo.radioLoss = loss
	gnb.controlPlaneInfo.radioLatencyMs = latencyMs
	gnb.controlPlaneInfo.radioJitterMs = jitterMs
	gnb.controlPlaneInfo.rlinkPool.Range(func(key, value any) bool {
		if conn, ok := value.(*rlink.Connection); ok {
			conn.SetRadioParams(loss, latencyMs, jitterMs)
		}
		return true
	})
}

func (gnb *GnbContext) GetId() string { return gnb.controlPlaneInfo.gnbId }

func (gnb *GnbContext) getGnbUe(ranUeId int64) (*GnbUeContext, error) {
	ue, err := gnb.ranUePool.Load(ranUeId)
	if !err {
		return nil, fmt.Errorf("UE ranUeId-%d is not find in GNB UE POOL", ranUeId)
	}
	return ue.(*GnbUeContext), nil
}

func (gnb *GnbContext) GetGnbUeByPrUeId(pRUeId int64) (*GnbUeContext, error) {
	ue, err := gnb.prUeIdPool.Load(pRUeId)
	if !err {
		return nil, fmt.Errorf("UE pRUeId-%d is not find in GNB PR UE POOL", pRUeId)
	}
	return ue.(*GnbUeContext), nil
}

func (gnb *GnbContext) GetPrUePoolSize() int {
	size := 0
	gnb.prUeIdPool.Range(func(key, value any) bool {
		size++
		return true
	})
	return size
}

func (gnb *GnbContext) getGnbUeByMsin(msin string) (*GnbUeContext, error) {
	ue, err := gnb.msinPool.Load(msin)
	if !err {
		return nil, fmt.Errorf("UE is not find in MSIN UE POOL")
	}
	return ue.(*GnbUeContext), nil
}

// func (gnb *GnbContext) getGnbUeByTeid(teid uint32) (*GnbUeContext, error) {
// 	ue, err := gnb.teidPool.Load(teid)
// 	if !err {
// 		return nil, fmt.Errorf("UE is not find in GNB UE POOL using TEID")
// 	}
// 	return ue.(*GnbUeContext), nil
// }

func (gnb *GnbContext) newGnbAmf(ip string, port int) *GnbAmfContext {
	// TODO if necessary add more information for AMF.
	amf := &GnbAmfContext{}

	// set id for AMF.
	amfId := gnb.getRanAmfId()
	amf.amfId = amfId

	// set AMF ip and AMF port.
	amf.amfIp = ip
	amf.amfPort = port

	// set state to AMF.
	amf.state = AMF_INACTIVE

	// store AMF in the AMF Pool of GNB.
	gnb.amfPool.Store(amfId, amf)

	// Plmns and slices supported by AMF initialized.
	amf.lenPlmn = 0
	amf.lenSlice = 0

	// return AMF Context
	return amf
}

// func (gnb *GnbContext) selectAmfByCapacity() *GNBAmf {
// 	var amfSelect *GNBAmf
// 	var maxWeightFactor int64 = -1
// 	gnb.amfPool.Range(func(key, value any) bool {
// 		amf := value.(*GNBAmf)
// 		if amf.relativeAmfCapacity > 0 {
// 			if maxWeightFactor < amf.tnla.tnlaWeightFactor {
// 				// select AMF
// 				maxWeightFactor = amf.tnla.tnlaWeightFactor
// 				amfSelect = amf
// 			}
// 		}
// 		return true
// 	})
//
// 	return amfSelect
// }

func (gnb *GnbContext) selectAmfByActive() *GnbAmfContext {
	var amfSelect *GnbAmfContext
	var maxWeightFactor int64 = -1
	gnb.amfPool.Range(func(key, value any) bool {
		amf := value.(*GnbAmfContext)
		if amf.state == AMF_ACTIVE {
			if maxWeightFactor < amf.tlnaAssoc.weightFactor {
				maxWeightFactor = amf.tlnaAssoc.weightFactor
				amfSelect = amf
			}
		}

		return true
	})

	return amfSelect
}

// func (gnb *GnbContext) getGnbAmf(amfId int64) (*GNBAmf, error) {
// 	amf, err := gnb.amfPool.Load(amfId)
// 	if !err {
// 		return nil, fmt.Errorf("AMF is not find in GNB AMF POOL ")
// 	}
// 	return amf.(*GNBAmf), nil
// }

func (gnb *GnbContext) getRanUeId() int64 {
	return atomic.AddInt64(&gnb.ranUeIdGenerator, 1) - 1
}

func (gnb *GnbContext) getUeTeid(ue *GnbUeContext) uint32 {
	id := atomic.AddUint32(&gnb.teidGenerator, 1) - 1
	gnb.downlinkTeidPool.Store(id, ue)
	return id
}

// for AMFs Pools.
func (gnb *GnbContext) getRanAmfId() int64 {
	return atomic.AddInt64(&gnb.amfIdGenerator, 1) - 1
}

func (gnb *GnbContext) addPagedUE(tmsi *ies.FiveGSTMSI) {
	gnb.pagedUELock.Lock()
	defer gnb.pagedUELock.Unlock()

	pagedUE := model.PagedUE{
		FiveGSTMSI: tmsi,
		Timestamp:  time.Now(),
	}
	gnb.pagedUEs = append(gnb.pagedUEs, pagedUE)

	go func() {
		time.Sleep(time.Second)
		gnb.pagedUELock.Lock()
		i := slices.Index(gnb.pagedUEs, pagedUE)
		if i == -1 {
			return
		}
		gnb.pagedUEs = slices.Delete(gnb.pagedUEs, i, i)
		gnb.pagedUELock.Unlock()
	}()
}

func (gnb *GnbContext) getPagedUEs() []model.PagedUE {
	gnb.pagedUELock.Lock()
	defer gnb.pagedUELock.Unlock()

	return gnb.pagedUEs[:]
}

func (gnb *GnbContext) getGnbIdInBytes() []byte {
	// changed for bytes.
	resu, err := hex.DecodeString(gnb.controlPlaneInfo.gnbId)
	if err != nil {
		gnb.Error("can not get gnbid in byte")
	}
	return resu
}

func (gnb *GnbContext) getTacInBytes() []byte {
	// changed for bytes.
	resu, err := hex.DecodeString(gnb.controlPlaneInfo.tac)
	if err != nil {
		gnb.Error("can not get Tac in byte")
	}
	return resu
}

func (gnb *GnbContext) getSliceInBytes() ([]byte, []byte) {
	sstBytes, err := hex.DecodeString(gnb.sliceConfiguration.sst)
	if err != nil {
		gnb.Error("can not get Slice-sst in byte")
	}

	if gnb.sliceConfiguration.sd != "" {
		sdBytes, err := hex.DecodeString(gnb.sliceConfiguration.sd)
		if err != nil {
			gnb.Error("can not get Slice-sd in byte")
		}
		return sstBytes, sdBytes
	}
	return sstBytes, nil
}

// func (gnb *GnbContext) GetPLMNIdentity() ies.PLMNIdentity {
func (gnb *GnbContext) getPLMNIdentityInBytes() []byte {
	return utils.PlmnIdToNgap(utils.PlmnId{Mcc: gnb.controlPlaneInfo.mcc, Mnc: gnb.controlPlaneInfo.mnc})
}

func (gnb *GnbContext) getNRCellIdentity() aper.BitString {
	nci := gnb.getGnbIdInBytes()
	var slice = make([]byte, 2)

	return aper.BitString{
		Bytes:   append(nci, slice...),
		NumBits: 36,
	}
}

func (gnb *GnbContext) getPlmnInOctets() []byte {
	var res string

	// reverse mcc and mnc
	mcc := reverse(gnb.controlPlaneInfo.mcc)
	mnc := reverse(gnb.controlPlaneInfo.mnc)

	if len(mnc) == 2 {
		res = fmt.Sprintf("%c%cf%c%c%c", mcc[1], mcc[2], mcc[0], mnc[0], mnc[1])
	} else {
		res = fmt.Sprintf("%c%c%c%c%c%c", mcc[1], mcc[2], mnc[2], mcc[0], mnc[0], mnc[1])
	}

	resu, _ := hex.DecodeString(res)
	return resu
}

// Check gnb: ready for ue send registraion request msg to amf
func (gnb *GnbContext) IsReady() bool {
	t := time.NewTicker(3 * time.Second)
	select {
	case <-gnb.isReady:
		return true
	case <-t.C:
		return false
	}
}

// Check handover status with ueid (prueid)
func (gnb *GnbContext) IsHandoverSuccess(prUeId int64) bool {
	v, ok := gnb.ueHoStatusPool.Load(prUeId)
	if ok {
		val := v.(bool)
		fmt.Fprintf(os.Stderr, "DEBUG IsHandoverSuccess(prUeId=%d) gnb=%s ok=true val=%v\n", prUeId, gnb.GetId(), val)
		return val
	}
	fmt.Fprintf(os.Stderr, "DEBUG IsHandoverSuccess(prUeId=%d) gnb=%s ok=false\n", prUeId, gnb.GetId())
	return false
}

func (gnb *GnbContext) Terminate() {
	gnb.close <- struct{}{}

	close(gnb.controlPlaneInfo.inboundChannel)
	gnb.Info("NAS channel Terminated")

	n2 := gnb.controlPlaneInfo.n2
	if n2 != nil {
		gnb.Info("N2/TNLA Terminated")
		n2.Close()
	}

	gnb.Info("GNB Terminated")
}

func (gnb *GnbContext) GetRlinkStatsSnapshot() map[string]rlink.Stats {
	snapshot := make(map[string]rlink.Stats)
	gnb.controlPlaneInfo.rlinkPool.Range(func(key, value any) bool {
		conn, ok := value.(*rlink.Connection)
		if ok {
			snapshot[key.(string)] = conn.GetStats()
		}
		return true
	})
	return snapshot
}

func (gnb *GnbContext) GetTimerEngine() *timer.TimerEngine {
	return gnb.timerEngine
}

func reverse(s string) string {
	// reverse string.
	var reversed string
	for _, valor := range s {
		reversed = string(valor) + reversed
	}
	return reversed
}
