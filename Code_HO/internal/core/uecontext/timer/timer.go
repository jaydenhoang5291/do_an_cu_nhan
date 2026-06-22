package timer

import "time"

type TimerType int

const (
	T3346 TimerType = iota + 1
	T3396
	T3445
	T3502
	T3510
	T3511
	T3512
	T3516
	T3517
	T3519
	T3520
	T3521
	T3525
	T3540
	T3527

	TXnRELOCprep
	TXnRELOCoverall

	TNGRELOCprep
	TNGRELOCoverall

	T301
	T304
	T310
	T311
	T312
	T430
)

func (t TimerType) String() string {
	switch t {
	case T3346:
		return "T3346"
	case T3396:
		return "T3396"
	case T3445:
		return "T3445"
	case T3502:
		return "T3502"
	case T3510:
		return "T3510"
	case T3511:
		return "T3511"
	case T3512:
		return "T3512"
	case T3516:
		return "T3516"
	case T3517:
		return "T3517"
	case T3519:
		return "T3519"
	case T3520:
		return "T3520"
	case T3521:
		return "T3521"
	case T3525:
		return "T3525"
	case T3540:
		return "T3540"
	case T3527:
		return "T3527"
	case TXnRELOCprep:
		return "TXnRELOCprep"
	case TXnRELOCoverall:
		return "TXnRELOCoverall"
	case TNGRELOCprep:
		return "TNGRELOCprep"
	case TNGRELOCoverall:
		return "TNGRELOCoverall"
	case T301:
		return "T301"
	case T304:
		return "T304"
	case T310:
		return "T310"
	case T311:
		return "T311"
	case T312:
		return "T312"
	case T430:
		return "T430"
	default:
		return "Unknown"
	}
}

const (
	T3346_duration = 15 * time.Minute
	T3396_duration = 30 * time.Minute
	T3445_duration = 12 * time.Hour
	T3502_duration = 12 * time.Minute
	T3510_duration = 15 * time.Second
	T3511_duration = 10 * time.Second
	T3512_duration = 54 * time.Minute
	T3516_duration = 30 * time.Second
	T3517_duration = 15 * time.Second
	T3519_duration = time.Minute
	T3520_duration = 15 * time.Second
	T3521_duration = 15 * time.Second
	T3525_duration = time.Minute
	T3540_duration = 10 * time.Second
	T3527_duration = 15 * time.Second

	// Xn Handover Timers
	TXnRELOCprep_duration    = 3 * time.Second
	TXnRELOCoverall_duration = 6 * time.Second

	// N2/NG Handover Timers
	TNGRELOCprep_duration    = 5 * time.Second
	TNGRELOCoverall_duration = 10 * time.Second

	// RRC Timers
	T301_duration = 2 * time.Second
	T310_duration = 1 * time.Second
	T311_duration = 3 * time.Second
	T312_duration = 500 * time.Millisecond
	T430_duration = 30 * time.Second
)

var T304_duration = 1000 * time.Millisecond // overridable at test startup
