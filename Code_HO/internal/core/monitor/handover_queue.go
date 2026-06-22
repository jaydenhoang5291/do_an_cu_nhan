package monitor

import (
	"container/heap"
	"sync"
	"time"
)

type ScheduledHandover struct {
	PrUeId       int64
	SourceGnbId  string
	TargetGnbId  string
	HandoverType int
	ExecuteAt    time.Time
	Priority     int

	cancelled bool
	index     int
}

// priorityQueue: higher Priority first; tie → earlier ExecuteAt
type priorityQueue []*ScheduledHandover

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	return pq[i].ExecuteAt.Before(pq[j].ExecuteAt)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	sh := x.(*ScheduledHandover)
	sh.index = n
	*pq = append(*pq, sh)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	sh := old[n-1]
	old[n-1] = nil
	sh.index = -1
	*pq = old[:n-1]
	return sh
}

type HandoverQueue struct {
	mu   sync.Mutex
	pq   priorityQueue
	byUe map[int64]*ScheduledHandover
}

func NewHandoverQueue() *HandoverQueue {
	hq := &HandoverQueue{
		pq:   make(priorityQueue, 0),
		byUe: make(map[int64]*ScheduledHandover),
	}
	heap.Init(&hq.pq)
	return hq
}

func (hq *HandoverQueue) Schedule(sh *ScheduledHandover) {
	hq.mu.Lock()
	defer hq.mu.Unlock()

	if existing, ok := hq.byUe[sh.PrUeId]; ok {
		existing.cancelled = true
	}

	sh.cancelled = false
	heap.Push(&hq.pq, sh)
	hq.byUe[sh.PrUeId] = sh
}

func (hq *HandoverQueue) GetNext() *ScheduledHandover {
	hq.mu.Lock()
	defer hq.mu.Unlock()

	for hq.pq.Len() > 0 {
		top := hq.pq[0]
		if top.cancelled {
			heap.Pop(&hq.pq)
			continue
		}
		sh := heap.Pop(&hq.pq).(*ScheduledHandover)
		if cur, ok := hq.byUe[sh.PrUeId]; ok && cur == sh {
			delete(hq.byUe, sh.PrUeId)
		}
		return sh
	}
	return nil
}

func (hq *HandoverQueue) Cancel(prUeId int64) bool {
	hq.mu.Lock()
	defer hq.mu.Unlock()

	sh, ok := hq.byUe[prUeId]
	if !ok {
		return false
	}
	sh.cancelled = true
	delete(hq.byUe, prUeId)
	return true
}

func (hq *HandoverQueue) Size() int {
	hq.mu.Lock()
	defer hq.mu.Unlock()

	count := 0
	for _, sh := range hq.pq {
		if !sh.cancelled {
			count++
		}
	}
	return count
}

// GetPendingScheduled returns a snapshot of all non-cancelled items.
func (hq *HandoverQueue) GetPendingScheduled() []*ScheduledHandover {
	hq.mu.Lock()
	defer hq.mu.Unlock()

	result := make([]*ScheduledHandover, 0, len(hq.pq))
	for _, sh := range hq.pq {
		if !sh.cancelled {
			result = append(result, sh)
		}
	}
	return result
}
