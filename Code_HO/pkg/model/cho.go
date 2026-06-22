package model

import (
	"stormsim/internal/transport/rlink"
	"time"
)

type ChoConditionType int

const (
	ChoConditionA3 ChoConditionType = iota
	ChoConditionA5
	ChoConditionNtnPosition
	ChoConditionNtnCoverage
)

type ChoCondition struct {
	Type            ChoConditionType
	Threshold       float64
	Hysteresis      float64
	Offset          float64
	TimeToTrigger   time.Duration
	MinCoverageTime time.Duration
	MaxDistanceKm   float64
}

type CellMeasurement struct {
	GnbId string
	Rsrp  float64
	Rsrq  float64
}

type CandidateGnb struct {
	GnbId         string
	Conn          *rlink.Connection
	RsrpOffset    float64
	Hysteresis    float64
	TimeToTrigger time.Duration
}

type MeasurementReport struct {
	PrUeId      int64
	Neighbors   []CellMeasurement
	ServingRsrp float64
}

type CHOConfig struct {
	PrUeId      int64
	Candidates  []CandidateGnb
	Conditions  []ChoCondition
	ServingRsrp float64
}

type RRCReconfigurationComplete struct {
	PrUeId        int64
	TargetGnbId   string
	SelectedGnbId string
}

type CHOExecuteNotification struct {
	PrUeId      int64
	TargetGnbId string
	SourceGnbId string
}

type ConditionalHandoverCancel struct {
	PrUeId      int64
	TargetIds   []string
	SourceGnbId string
}
