package rlink

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultBufferSize int = 256
const DefaultDuration time.Duration = 3 * time.Second

// Unified message types that both UE and GNB use
type Message interface {
	GetType() string
}

type Stats struct {
	Sent      int64
	Dropped   int64
	Timeout   int64
	Delivered int64
}

// Shared connection structure used by both UE and GNB
type Connection struct {
	UEID       int64
	UEmsin     string
	GNBID      string
	UplinkCh   chan Message // UE → GNB
	DownlinkCh chan Message // GNB → UE
	Timeout    time.Duration

	// simulation params
	UplinkDelay    time.Duration
	DownlinkDelay  time.Duration
	UplinkJitter   time.Duration
	DownlinkJitter time.Duration
	UplinkLoss     float64 // 0.0 → 1.0
	DownlinkLoss   float64

	closed bool
	mu     sync.RWMutex

	wg sync.WaitGroup

	stats Stats
}

func NewConnection(
	ueID int64,
	msin string,
	gnbID string,
	bufferSize int,
	timeout time.Duration,
	workers int,
) *Connection {

	c := &Connection{
		UEID:       ueID,
		UEmsin:     msin,
		GNBID:      gnbID,
		UplinkCh:   make(chan Message, bufferSize),
		DownlinkCh: make(chan Message, bufferSize),
		Timeout:    timeout,
	}

	return c
}

func (c *Connection) sendAsync(
	ch chan Message,
	msg Message,
	delay time.Duration,
	isUplink bool,
) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		if delay > 0 {
			time.Sleep(delay)
		}

		c.mu.RLock()
		if c.closed {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		timer := time.NewTimer(c.Timeout)
		defer timer.Stop()

		select {
		case ch <- msg:
			atomic.AddInt64(&c.stats.Delivered, 1)
		case <-timer.C:
			atomic.AddInt64(&c.stats.Timeout, 1)
			fmt.Printf("DEBUG RLink TIMEOUT: msg=%s\n", msg.GetType())
		}
	}()
}

func (c *Connection) SendUplink(msg Message) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	c.mu.RUnlock()

	atomic.AddInt64(&c.stats.Sent, 1)

	fmt.Fprintf(os.Stderr, "DEBUG SendUplink: UplinkLoss=%.2f msg=%s\n", c.UplinkLoss, msg.GetType())
	if rand.Float64() < c.UplinkLoss {
		atomic.AddInt64(&c.stats.Dropped, 1)
		return nil
	}

	delay := c.UplinkDelay
	if c.UplinkJitter > 0 {
		j := time.Duration(rand.Int63n(int64(c.UplinkJitter))) - c.UplinkJitter/2
		delay += j
		if delay < 0 {
			delay = 0
		}
	}

	c.sendAsync(c.UplinkCh, msg, delay, true)
	return nil
}

func (c *Connection) SendDownlink(msg Message) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	c.mu.RUnlock()

	atomic.AddInt64(&c.stats.Sent, 1)

	if rand.Float64() < c.DownlinkLoss {
		atomic.AddInt64(&c.stats.Dropped, 1)
		return nil
	}

	delay := c.DownlinkDelay
	if c.DownlinkJitter > 0 {
		j := time.Duration(rand.Int63n(int64(c.DownlinkJitter))) - c.DownlinkJitter/2
		delay += j
		if delay < 0 {
			delay = 0
		}
	}

	c.sendAsync(c.DownlinkCh, msg, delay, false)
	return nil
}

// SendDownlinkCritical sends a message bypassing loss/jitter simulation.
// Used for critical control messages (handover commands, RRC signaling)
// that must reach the UE for correct protocol behavior.
func (c *Connection) SendDownlinkCritical(msg Message) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	c.mu.RUnlock()

	atomic.AddInt64(&c.stats.Sent, 1)

	c.sendAsync(c.DownlinkCh, msg, 0, false)
	return nil
}

// SendUplinkCritical bypasses loss/jitter for critical uplink control messages.
func (c *Connection) SendUplinkCritical(msg Message) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return fmt.Errorf("connection closed")
	}
	c.mu.RUnlock()

	atomic.AddInt64(&c.stats.Sent, 1)

	c.sendAsync(c.UplinkCh, msg, 0, false)
	return nil
}

func (c *Connection) GetUplinkChan() <-chan Message {
	return c.UplinkCh
}

func (c *Connection) GetDownlinkChan() <-chan Message {
	return c.DownlinkCh
}

func (c *Connection) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.wg.Wait()
		return // ← early return prevents double-close of all channels
	}
	c.closed = true
	c.mu.Unlock()
	c.wg.Wait()
	close(c.UplinkCh)
	close(c.DownlinkCh)
}

func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func (c *Connection) GetStats() Stats {
	return Stats{
		Sent:      atomic.LoadInt64(&c.stats.Sent),
		Dropped:   atomic.LoadInt64(&c.stats.Dropped),
		Timeout:   atomic.LoadInt64(&c.stats.Timeout),
		Delivered: atomic.LoadInt64(&c.stats.Delivered),
	}
}

func ConnectionKey(ueID int64, gnbID string) string {
	return fmt.Sprintf("%d:%s", ueID, gnbID)
}

func (c *Connection) SetLoss(loss float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UplinkLoss = loss
	c.DownlinkLoss = loss
}

func (c *Connection) SetRadioParams(loss float64, delayMs, jitterMs int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UplinkLoss = loss
	c.DownlinkLoss = loss
	c.UplinkDelay = time.Duration(delayMs) * time.Millisecond
	c.DownlinkDelay = time.Duration(delayMs) * time.Millisecond
	c.UplinkJitter = time.Duration(jitterMs) * time.Millisecond
	c.DownlinkJitter = time.Duration(jitterMs) * time.Millisecond
}
