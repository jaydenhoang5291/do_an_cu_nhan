package monitor

import "errors"

var (
	ErrEventChannelFull    = errors.New("event channel is full")
	ErrGnbNotFound         = errors.New("gNB not found")
	ErrInvalidHandoverType = errors.New("invalid handover type")
	ErrHandoverNotPossible = errors.New("handover not possible")
	ErrQueueEmpty          = errors.New("handover queue is empty")
	ErrAlreadyCancelled    = errors.New("handover already cancelled")
)
