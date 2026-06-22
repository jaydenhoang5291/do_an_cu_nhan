package uecontext

import (
	"stormsim/pkg/model"
	"time"
)

func (ue *UeContext) startChoEvaluator() {
	if ue.choEnabled {
		return
	}
	ue.choEnabled = true

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if ue.ctx.Err() != nil {
					return
				}
				ue.evaluateChoConditions()
			case <-ue.ctx.Done():
				return
			}
		}
	}()
}

func (ue *UeContext) evaluateChoConditions() {
	ue.rsrpMu.RLock()
	servingRsrp, hasServing := ue.rsrpValues[ue.gnbId]
	candidates := make([]model.CandidateGnb, len(ue.choCandidates))
	copy(candidates, ue.choCandidates)
	conditions := make([]model.ChoCondition, len(ue.choConditions))
	copy(conditions, ue.choConditions)
	rsrpSnapshot := make(map[string]float64, len(ue.rsrpValues))
	for k, v := range ue.rsrpValues {
		rsrpSnapshot[k] = v
	}
	ue.rsrpMu.RUnlock()

	if !hasServing || len(candidates) == 0 {
		return
	}

	bestCandidate := ue.findBestCandidate(servingRsrp, candidates, conditions, rsrpSnapshot)
	if bestCandidate != nil {
		ue.triggerChoExecute(bestCandidate)
	}
}

func (ue *UeContext) findBestCandidate(servingRsrp float64, candidates []model.CandidateGnb, conditions []model.ChoCondition, rsrpSnapshot map[string]float64) *model.CandidateGnb {
	var best *model.CandidateGnb
	var bestRsrp float64 = -200

	for i := range candidates {
		c := &candidates[i]
		neighborRsrp, ok := rsrpSnapshot[c.GnbId]
		if !ok {
			continue
		}

		if ue.checkCandidateConditions(c, neighborRsrp, servingRsrp, conditions) {
			if neighborRsrp > bestRsrp {
				bestRsrp = neighborRsrp
				best = c
			}
		}
	}

	return best
}

func (ue *UeContext) checkCandidateConditions(candidate *model.CandidateGnb, neighborRsrp, servingRsrp float64, conditions []model.ChoCondition) bool {
	if len(conditions) == 0 {
		return ue.checkA3Condition(neighborRsrp, servingRsrp, candidate.RsrpOffset, candidate.Hysteresis)
	}

	for _, cond := range conditions {
		switch cond.Type {
		case model.ChoConditionA3:
			if !ue.checkA3Condition(neighborRsrp, servingRsrp, cond.Offset+cond.Hysteresis, 0) {
				return false
			}
		case model.ChoConditionA5:
			if !ue.checkA5Condition(servingRsrp, neighborRsrp, cond.Threshold, cond.Offset) {
				return false
			}
		case model.ChoConditionNtnPosition:
			if !ue.checkNtnPositionCondition(candidate, cond) {
				return false
			}
		case model.ChoConditionNtnCoverage:
			if !ue.checkNtnCoverageCondition(candidate, cond) {
				return false
			}
		}
	}

	return true
}

func (ue *UeContext) checkA3Condition(neighborRsrp, servingRsrp, offset, hysteresis float64) bool {
	return neighborRsrp >= servingRsrp+offset+hysteresis
}

func (ue *UeContext) checkA5Condition(servingRsrp, neighborRsrp, threshold1, threshold2 float64) bool {
	return servingRsrp < threshold1 && neighborRsrp > threshold2
}

func (ue *UeContext) checkNtnPositionCondition(candidate *model.CandidateGnb, cond model.ChoCondition) bool {
	return true
}

func (ue *UeContext) checkNtnCoverageCondition(candidate *model.CandidateGnb, cond model.ChoCondition) bool {
	return true
}
