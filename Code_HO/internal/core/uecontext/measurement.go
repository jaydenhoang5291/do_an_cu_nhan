package uecontext

import (
	"stormsim/internal/core/gnbcontext"
	"stormsim/pkg/model"
	"time"
)

func (ue *UeContext) triggerMeasurementReport() {
	ue.measurementTicker = time.NewTicker(500 * time.Millisecond)

	go func() {
		for {
			select {
			case <-ue.measurementTicker.C:
				if ue.ctx.Err() != nil {
					return
				}
				ue.sendMeasurementReport()
			case <-ue.ctx.Done():
				return
			}
		}
	}()
}

func (ue *UeContext) sendMeasurementReport() {
	neighbors, servingRsrp := ue.readNeighborRsrp()
	if len(neighbors) == 0 {
		return
	}

	report := &model.MeasurementReport{
		PrUeId:      int64(ue.id),
		Neighbors:   neighbors,
		ServingRsrp: servingRsrp,
	}

	gnbcontext.SendToGnb(ue.msin, report, ue.gnbId, false)
}

func (ue *UeContext) readNeighborRsrp() ([]model.CellMeasurement, float64) {
	ue.rsrpMu.RLock()
	defer ue.rsrpMu.RUnlock()

	if len(ue.rsrpValues) == 0 {
		return nil, 0
	}

	servingRsrp, _ := ue.rsrpValues[ue.gnbId]
	measurements := make([]model.CellMeasurement, 0, len(ue.rsrpValues)-1)
	for gnbId, rsrp := range ue.rsrpValues {
		if gnbId == ue.gnbId {
			continue
		}
		measurements = append(measurements, model.CellMeasurement{
			GnbId: gnbId,
			Rsrp:  rsrp,
		})
	}
	return measurements, servingRsrp
}

func (ue *UeContext) UpdateRsrpValues(rsrpMap map[string]float64) {
	ue.rsrpMu.Lock()
	defer ue.rsrpMu.Unlock()
	ue.rsrpValues = rsrpMap
}
