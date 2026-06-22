package timer_test

import (
	"context"
	"stormsim/internal/core/uecontext/timer"
	"sync"
	"testing"
	"time"
)

func TestTimerEventHook(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	te := timer.NewTimerEngine(ctx)

	var mu sync.Mutex
	events := make([]struct {
		timerType timer.TimerType
		prUeId    int64
		event     string
		elapsed   time.Duration
	}, 0)

	te.SetEventHook(func(timerType timer.TimerType, prUeId int64, event string, elapsed time.Duration) {
		mu.Lock()
		events = append(events, struct {
			timerType timer.TimerType
			prUeId    int64
			event     string
			elapsed   time.Duration
		}{timerType, prUeId, event, elapsed})
		mu.Unlock()
	})

	cfg := timer.TimerConfig{
		TimerType: timer.T304,
		Duration:  50 * time.Millisecond,
		CountMax:  2,
	}

	te.CreateTimer(cfg, 123)

	// Test Start
	err := te.Start(timer.T304, 123)
	if err != nil {
		t.Fatalf("failed to start timer: %v", err)
	}

	// Wait for expiration
	time.Sleep(120 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have at least start and timeout events
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	if events[0].event != "start" || events[0].timerType != timer.T304 || events[0].prUeId != 123 {
		t.Errorf("unexpected start event: %+v", events[0])
	}

	foundTimeout := false
	for _, e := range events {
		if e.event == "timeout" {
			foundTimeout = true
			if e.timerType != timer.T304 || e.prUeId != 123 {
				t.Errorf("unexpected timeout event properties: %+v", e)
			}
			if e.elapsed < 40*time.Millisecond {
				t.Errorf("elapsed duration too small: %v", e.elapsed)
			}
		}
	}

	if !foundTimeout {
		t.Error("expected timeout event not found")
	}
}
