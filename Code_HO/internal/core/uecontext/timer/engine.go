package timer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TimerCallback represents a callback function for timeout or expiration
type TimerCallback func()

type TimerEventHook func(timerType TimerType, prUeId int64, event string, elapsed time.Duration)

// TimerConfig holds the configuration for a timer
type TimerConfig struct {
	TimerType   TimerType     // Unique identifier for the timer
	Duration    time.Duration // Duration before timeout
	CountMax    int           // Maximum number of retries
	TimeoutFunc TimerCallback // Function to call on each timeout (can be nil)
	ExpireFunc  TimerCallback // Function to call when timer expires
}

// Timer represents a single timer with its state
type Timer struct {
	config        TimerConfig
	ueId          int64 // PrUeId this timer belongs to
	currentCount  int
	isActive      bool
	startTime     time.Time
	cancel        context.CancelFunc
	manualTrigger chan struct{}
	mu            sync.RWMutex
}

// TimerEngine manages multiple concurrent timers
type TimerEngine struct {
	timers          map[string]*Timer
	BlockTimerEvent bool
	mu              sync.RWMutex
	ctx             context.Context
	EventHook       TimerEventHook
}

// NewTimerEngine creates a new timer engine tied to a context
func NewTimerEngine(ctx context.Context) *TimerEngine {
	return &TimerEngine{
		timers: make(map[string]*Timer),
		ctx:    ctx,
	}
}

// StartWatchdog starts a background goroutine that periodically scans active
// timers and emits a "zombie" event via the EventHook for timers whose
// elapsed duration exceeds config.Duration * thresholdMultiplier.
// This is a diagnostic helper and does not modify timer state.
func (te *TimerEngine) StartWatchdog(interval time.Duration, thresholdMultiplier float64) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-te.ctx.Done():
				return
			case <-ticker.C:
				// Copy timers under read lock to avoid holding lock while checking each timer
				te.mu.RLock()
				timersCopy := make([]*Timer, 0, len(te.timers))
				for _, t := range te.timers {
					timersCopy = append(timersCopy, t)
				}
				hook := te.EventHook
				te.mu.RUnlock()

				if hook == nil {
					continue
				}

				now := time.Now()
				for _, t := range timersCopy {
					t.mu.RLock()
					active := t.isActive
					start := t.startTime
					dur := t.config.Duration
					t.mu.RUnlock()

					if !active || dur <= 0 {
						continue
					}

					elapsed := now.Sub(start)
					if float64(elapsed) > float64(dur)*thresholdMultiplier {
						// emit a diagnostic 'zombie' event
						hook(t.config.TimerType, t.ueId, "zombie", elapsed)
					}
				}
			}
		}
	}()
}

func (te *TimerEngine) SetEventHook(hook TimerEventHook) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.EventHook = hook
}

func (te *TimerEngine) getTimerKey(timerType TimerType, prUeId int64) string {
	return fmt.Sprintf("%d-%d", timerType, prUeId)
}

// CreateTimer creates a new timer with the given configuration.
// If a timer with the same key already exists and is active, it is cancelled
// first to prevent stale goroutines and ensure a clean lifecycle.
func (te *TimerEngine) CreateTimer(config TimerConfig, prUeId int64) {
	te.mu.Lock()

	key := te.getTimerKey(config.TimerType, prUeId)
	var replacedActive bool
	var replacedElapsed time.Duration

	// Cancel any existing active timer with the same key to prevent
	// stale goroutines from blocking new timer start or emitting
	// spurious timeout events.
	if existing, ok := te.timers[key]; ok {
		existing.mu.Lock()
		if existing.isActive {
			replacedElapsed = time.Since(existing.startTime)
			existing.cancel()
			existing.isActive = false
			replacedActive = true
		}
		existing.mu.Unlock()
	}

	timer := &Timer{
		config:        config,
		ueId:          prUeId,
		currentCount:  0,
		isActive:      false,
		manualTrigger: make(chan struct{}, 1), // Buffered channel to prevent blocking
	}

	te.timers[key] = timer
	hook := te.EventHook
	te.mu.Unlock()

	if replacedActive && hook != nil {
		hook(config.TimerType, prUeId, "restart", replacedElapsed)
	}
}

// Start begins the timer countdown in a goroutine
func (te *TimerEngine) Start(timerType TimerType, prUeId int64) error {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.RLock()
	timer, exists := te.timers[key]
	te.mu.RUnlock()

	if !exists {
		return fmt.Errorf("timer %s not found", key)
	}

	timer.mu.Lock()
	if timer.isActive {
		timer.mu.Unlock()
		return fmt.Errorf("timer %s is already active", key)
	}

	// Create a sub-context for this specific timer run, tied to the engine's context
	ctx, cancel := context.WithCancel(te.ctx)
	timer.cancel = cancel
	timer.startTime = time.Now()
	timer.isActive = true
	timer.mu.Unlock()

	timer.currentCount = 0

	// Start the timer in a goroutine
	go timer.run(ctx, te)

	te.mu.RLock()
	hook := te.EventHook
	te.mu.RUnlock()
	if hook != nil {
		hook(timerType, prUeId, "start", 0)
	}

	return nil
}

// Stop stops the timer
func (te *TimerEngine) Stop(timerType TimerType, prUeId int64) error {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.Lock()
	defer te.mu.Unlock()

	timer, exists := te.timers[key]
	if !exists {
		return fmt.Errorf("timer %s not found", key)
	}

	var elapsed time.Duration
	timer.mu.Lock()
	if timer.isActive {
		elapsed = time.Since(timer.startTime)
		timer.cancel()
		timer.isActive = false
	}
	timer.mu.Unlock()

	delete(te.timers, key)

	hook := te.EventHook
	if hook != nil {
		hook(timerType, prUeId, "stop", elapsed)
	}
	return nil
}

// Stop stops the timer then start expire func()
func (te *TimerEngine) StopWithExpireFunc(timerType TimerType, prUeId int64) error {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.Lock()
	defer te.mu.Unlock()

	timer, exists := te.timers[key]
	if !exists {
		return fmt.Errorf("timer %s not found", key)
	}

	var elapsed time.Duration
	timer.mu.Lock()
	if timer.isActive {
		elapsed = time.Since(timer.startTime)
		timer.cancel()
		timer.isActive = false
	}
	timer.mu.Unlock()

	if timer.config.ExpireFunc != nil {
		timer.config.ExpireFunc()
	}

	delete(te.timers, key)

	hook := te.EventHook
	if hook != nil {
		hook(timerType, prUeId, "stop", elapsed)
	}
	return nil
}

// StopWithEvent stops the timer and invokes the EventHook with a custom event string.
// This is useful when a caller wants to record a stop that should be treated as a
// timeout/cancel event in higher-level logging/analysis.
func (te *TimerEngine) StopWithEvent(timerType TimerType, prUeId int64, event string) error {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.Lock()
	defer te.mu.Unlock()

	timer, exists := te.timers[key]
	if !exists {
		return fmt.Errorf("timer %s not found", key)
	}

	var elapsed time.Duration
	timer.mu.Lock()
	if timer.isActive {
		elapsed = time.Since(timer.startTime)
		timer.cancel()
		timer.isActive = false
	}
	timer.mu.Unlock()

	delete(te.timers, key)

	hook := te.EventHook
	if hook != nil {
		hook(timerType, prUeId, event, elapsed)
	}
	return nil
}

// TriggerTimeout manually triggers a timeout event for the specified timer
func (te *TimerEngine) TriggerTimeout(timerType TimerType, prUeId int64) error {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.RLock()
	timer, exists := te.timers[key]
	te.mu.RUnlock()

	if !exists {
		return fmt.Errorf("timer with key %s not found", key)
	}

	timer.mu.RLock()
	isActive := timer.isActive
	timer.mu.RUnlock()

	if !isActive {
		return fmt.Errorf("timer %s is not active", key)
	}

	// Send manual trigger (non-blocking)
	select {
	case timer.manualTrigger <- struct{}{}:
		return nil
	default:
		// Channel is full, trigger already pending
		return fmt.Errorf("timer %s already has a pending manual trigger", key)
	}
}

// FireEvent allows external code to inject a custom timer event through the hook
// without modifying the timer's lifecycle. Used for reporting transport-level failures
// separately from natural timer timeouts (e.g., Xn context forwarding failure).
func (te *TimerEngine) FireEvent(timerType TimerType, prUeId int64, event string, _ time.Duration) {
	te.mu.RLock()
	hook := te.EventHook
	key := te.getTimerKey(timerType, prUeId)
	elapsed := time.Duration(0)
	if t, ok := te.timers[key]; ok {
		t.mu.RLock()
		if t.isActive {
			elapsed = time.Since(t.startTime)
		}
		t.mu.RUnlock()
	}
	te.mu.RUnlock()
	if hook != nil {
		hook(timerType, prUeId, event, elapsed)
	}
}

// GetTimerStatus returns the current status of a timer
func (te *TimerEngine) GetTimerStatus(timerType TimerType, prUeId int64) (bool, int, error) {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.RLock()
	timer, exists := te.timers[key]
	te.mu.RUnlock()

	if !exists {
		return false, 0, nil
	}

	timer.mu.RLock()
	defer timer.mu.RUnlock()

	return timer.isActive, timer.currentCount, nil
}

// RemoveTimer removes a timer from the engine
func (te *TimerEngine) RemoveTimer(timerType TimerType, prUeId int64) error {
	key := te.getTimerKey(timerType, prUeId)
	te.mu.Lock()
	defer te.mu.Unlock()

	timer, exists := te.timers[key]
	if !exists {
		return fmt.Errorf("timer %s not found", key)
	}

	// Stop the timer if it's active
	timer.mu.Lock()
	if timer.isActive {
		timer.cancel()
		timer.isActive = false
	}
	timer.mu.Unlock()

	delete(te.timers, key)
	return nil
}

// ListActiveTimers returns a list of active timer keys
func (te *TimerEngine) ListActiveTimers() []string {
	te.mu.RLock()
	defer te.mu.RUnlock()

	var activeTimers []string
	for id, timer := range te.timers {
		timer.mu.RLock()
		if timer.isActive {
			activeTimers = append(activeTimers, id)
		}
		timer.mu.RUnlock()
	}
	return activeTimers
}

// run executes the timer logic in a goroutine
func (t *Timer) run(ctx context.Context, te *TimerEngine) {
	ticker := time.NewTicker(t.config.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context is canceled when Stop() is called. Do NOT execute ExpireFunc.
			return
		case <-t.manualTrigger:
			t.excute(te)
			ticker.Reset(t.config.Duration)
		case <-ticker.C:
			t.excute(te)
		}
		if t.currentCount >= t.config.CountMax {
			return
		}
	}
}

func (t *Timer) excute(te *TimerEngine) {
	t.mu.Lock()
	t.currentCount++
	currentCount := t.currentCount
	t.mu.Unlock()

	// Check if we've reached the maximum count
	if currentCount == t.config.CountMax {
		// Timer has expired
		t.mu.Lock()
		if !t.isActive {
			t.mu.Unlock()
			return
		}
		t.isActive = false
		t.mu.Unlock()

		elapsed := time.Since(t.startTime)

		// Remove from engine map
		te.mu.Lock()
		key := te.getTimerKey(t.config.TimerType, t.ueId) // I need to add ueId to Timer struct
		if te.timers[key] == t {
			delete(te.timers, key)
		}
		hook := te.EventHook
		te.mu.Unlock()

		if hook != nil {
			hook(t.config.TimerType, t.ueId, "timeout", elapsed)
		}

		// Call expire function if provided
		if t.config.ExpireFunc != nil {
			go t.config.ExpireFunc()
		}
		return
	}

	// Call timeout function if provided
	if t.config.TimeoutFunc != nil {
		go t.config.TimeoutFunc()
	}
}
