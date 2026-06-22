package monitor

import (
	"sync"
	"time"
)

type StatsSummary struct {
	Total         int64            `json:"total"`
	Success       int64            `json:"success"`
	Failed        int64            `json:"failed"`
	SuccessRate   float64          `json:"success_rate"`
	AvgDurationMs float64          `json:"avg_duration_ms"`
	MinDurationMs float64          `json:"min_duration_ms"`
	MaxDurationMs float64          `json:"max_duration_ms"`
	XnHandovers   int64            `json:"xn_handovers"`
	N2Handovers   int64            `json:"n2_handovers"`
	ByFailReason  map[string]int64 `json:"by_fail_reason"`
}

type MonitorHandoverTracker struct {
	mu      sync.RWMutex
	monitor *HandoverMonitor

	total        int64
	success      int64
	failed       int64
	xnCount      int64
	n2Count      int64
	totalDurMs   float64
	minDurMs     float64
	maxDurMs     float64
	byFailReason map[string]int64
}

func NewMonitorHandoverTracker(monitor *HandoverMonitor) *MonitorHandoverTracker {
	return &MonitorHandoverTracker{
		monitor:      monitor,
		byFailReason: make(map[string]int64),
		minDurMs:     -1,
	}
}

// RecordResult updates aggregate stats for a completed handover.
func (t *MonitorHandoverTracker) RecordResult(result HandoverResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.total++

	if result.Success {
		t.success++
	} else {
		t.failed++
		if result.FailureReason != "" {
			t.byFailReason[result.FailureReason]++
		}
	}

	switch result.HandoverType {
	case HandoverTypeXn:
		t.xnCount++
	case HandoverTypeN2:
		t.n2Count++
	}

	durMs := float64(result.Duration) / float64(time.Millisecond)
	t.totalDurMs += durMs

	if t.minDurMs < 0 || durMs < t.minDurMs {
		t.minDurMs = durMs
	}
	if durMs > t.maxDurMs {
		t.maxDurMs = durMs
	}

}

func (t *MonitorHandoverTracker) GetStatsSummary() StatsSummary {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var successRate float64
	if t.total > 0 {
		successRate = float64(t.success) / float64(t.total) * 100.0
	}

	var avgDurMs float64
	if t.total > 0 {
		avgDurMs = t.totalDurMs / float64(t.total)
	}

	minDur := t.minDurMs
	if minDur < 0 {
		minDur = 0
	}

	// copy failure map so caller can't mutate internal state
	byFail := make(map[string]int64, len(t.byFailReason))
	for k, v := range t.byFailReason {
		byFail[k] = v
	}

	return StatsSummary{
		Total:         t.total,
		Success:       t.success,
		Failed:        t.failed,
		SuccessRate:   successRate,
		AvgDurationMs: avgDurMs,
		MinDurationMs: minDur,
		MaxDurationMs: t.maxDurMs,
		XnHandovers:   t.xnCount,
		N2Handovers:   t.n2Count,
		ByFailReason:  byFail,
	}
}

// GetAllResults satisfies the OAM interface — returns nil since we only
// keep aggregate stats, not individual records.
func (t *MonitorHandoverTracker) GetAllResults() []HandoverResult {
	return nil
}
