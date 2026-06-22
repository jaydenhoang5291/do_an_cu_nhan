package logger

import (
	"time"
)

// TimerEventEntry represents a timer event log (start, stop, timeout, fail)
type TimerEventEntry struct {
	Ts              time.Time `json:"ts"`                // Timestamp tuyệt đối
	StartTs         time.Time `json:"start_ts"`          // Timer start timestamp
	StopTs          time.Time `json:"stop_ts"`           // Timer stop/timeout/cancel timestamp
	TrialId         int       `json:"trial_id"`          // Số thứ tự lần HO trong session
	Pdr             float64   `json:"pdr"`               // Mức PDR đang inject
	DelayMs         float64   `json:"delay_ms"`          // Delay one-way đang inject
	HoType          string    `json:"ho_type"`           // "Xn" | "NG"
	TimerName       string    `json:"timer_name"`        // "T304" | "T310" | "T311" | ...
	Event           string    `json:"event"`             // "start" | "stop" | "timeout" | "cancel" | "restart"
	StopReason      string    `json:"stop_reason"`       // "success" | "timeout" | "zombie" | "cancel" | "restart" (optional)
	ElapsedMs       float64   `json:"elapsed_ms"`        // Thời gian thực tế từ start đến event
	Result          string    `json:"result"`            // "success" | "timeout" | "aborted"
	HoTrigger       string    `json:"ho_trigger"`        // "rsrp_drop" | "beam_fail" | "manual"
	SourceGnb       string    `json:"source_gnb"`        // ID gNB nguồn
	TargetGnb       string    `json:"target_gnb"`        // ID gNB đích
	RsrpAtTrigger   float64   `json:"rsrp_at_trigger"`   // RSRP (dBm) tại thời điểm trigger HO
	ParentTimer     string    `json:"parent_timer"`      // Timer nào chạy trước
	RlfDetected     bool      `json:"rlf_detected"`      // Có RLF xảy ra không
	Event_          string    `json:"event_"`            // Extra field for "summary" entries
	PacketLossCount int       `json:"packet_loss_count"` // Số gói bị drop thực tế
	BeamId          string    `json:"beam_id"`           // Beam ID khi failure xảy ra
	CandidateBeamId string    `json:"candidate_beam_id"` // Beam được chọn để recovery
	BeamRecoveryMs  float64   `json:"beam_recovery_ms"`  // Thời gian recovery
	N2RttMs         float64   `json:"n2_rtt_ms"`         // RTT trên N2 interface
}

// HandoverTrialSummaryEntry ghi một lần sau khi toàn bộ HO attempt kết thúc
type HandoverTrialSummaryEntry struct {
	Ts               time.Time `json:"ts"`                 // Timestamp kết thúc
	StartTs          time.Time `json:"start_ts"`           // Timestamp bắt đầu HO
	TrialId          int       `json:"trial_id"`           // Số thứ tự trial
	Pdr              float64   `json:"pdr"`                // Mức PDR
	DelayMs          float64   `json:"delay_ms"`           // Delay
	HoType           string    `json:"ho_type"`            // "Xn" | "NG"
	OverallResult    string    `json:"overall_result"`     // "success" | "fail"
	FailReason       string    `json:"fail_reason"`        // Timer nào gây fail, hoặc "" nếu success
	TotalHoMs        float64   `json:"total_ho_ms"`        // Tổng thời gian từ trigger đến hoàn tất
	SourceGnb        string    `json:"source_gnb"`         // Gnb source
	TargetGnb        string    `json:"target_gnb"`         // Gnb target
	RlfOffsetMs      float64   `json:"rlf_offset_ms"`      // RLF xảy ra sau bao nhiêu ms (kịch bản 3)
	RsrpDuringHo     float64   `json:"rsrp_during_ho"`     // RSRP khi RLF xảy ra
	ReestabTargetGnb string    `json:"reestab_target_gnb"` // UE chọn cell nào để reestablish
}

// T304BreakpointEntry dùng cho kịch bản 2 - tìm T304 breakpoint
type T304BreakpointEntry struct {
	DelayMs         float64 `json:"delay_ms"`
	TrialId         int     `json:"trial_id"`
	Result          string  `json:"result"` // "success" | "timeout"
	ElapsedMs       float64 `json:"elapsed_ms"`
	PacketLossCount int     `json:"packet_loss_count"`
}

// CascadeRlfEntry dùng cho kịch bản 3 - T310 → T311 → T301
type CascadeRlfEntry struct {
	Ts               time.Time `json:"ts"`
	TrialId          int       `json:"trial_id"`
	Pdr              float64   `json:"pdr"`
	DelayMs          float64   `json:"delay_ms"`
	RlfOffsetMs      float64   `json:"rlf_offset_ms"` // RLF xảy ra sau bao nhiêu ms kể từ HO start
	RsrpDuringHo     float64   `json:"rsrp_during_ho"`
	T310Result       string    `json:"t310_result"`    // "success" | "timeout"
	T311Result       string    `json:"t311_result"`    // "success" | "timeout"
	T301Result       string    `json:"t301_result"`    // "success" | "timeout"
	OverallResult    string    `json:"overall_result"` // "success" | "fail"
	ReestabTargetGnb string    `json:"reestab_target_gnb"`
}

// RingBuffer for timer events (separate from general logging)
type TimerEventRingBuffer struct {
	timerEvents      *RingBuffer[TimerEventEntry]
	summaryEvents    *RingBuffer[HandoverTrialSummaryEntry]
	t304Events       *RingBuffer[T304BreakpointEntry]
	cascadeRlfEvents *RingBuffer[CascadeRlfEntry]
}

// NewTimerEventRingBuffer creates a new timer event ring buffer
func NewTimerEventRingBuffer(capacity int) *TimerEventRingBuffer {
	if capacity <= 0 {
		capacity = 10000 // Default capacity for 10k+ events
	}
	return &TimerEventRingBuffer{
		timerEvents:      NewRingBuffer[TimerEventEntry](capacity),
		summaryEvents:    NewRingBuffer[HandoverTrialSummaryEntry](capacity / 10), // Summaries are fewer
		t304Events:       NewRingBuffer[T304BreakpointEntry](capacity),
		cascadeRlfEvents: NewRingBuffer[CascadeRlfEntry](capacity / 10),
	}
}

// PushTimerEvent adds a raw timer event to the ring buffer
func (trb *TimerEventRingBuffer) PushTimerEvent(entry TimerEventEntry) {
	trb.timerEvents.Push(entry)
}

// PushSummaryEvent adds a handover trial summary event to the ring buffer
func (trb *TimerEventRingBuffer) PushSummaryEvent(entry HandoverTrialSummaryEntry) {
	trb.summaryEvents.Push(entry)
}

// GetTimerEvents returns all timer events
func (trb *TimerEventRingBuffer) GetTimerEvents() []TimerEventEntry {
	return trb.timerEvents.GetAll()
}

// GetSummaryEvents returns all trial summary events
func (trb *TimerEventRingBuffer) GetSummaryEvents() []HandoverTrialSummaryEntry {
	return trb.summaryEvents.GetAll()
}

// GetT304Events returns all T304 breakpoint events
func (trb *TimerEventRingBuffer) GetT304Events() []T304BreakpointEntry {
	return trb.t304Events.GetAll()
}

// GetCascadeRlfEvents returns all cascade RLF events
func (trb *TimerEventRingBuffer) GetCascadeRlfEvents() []CascadeRlfEntry {
	return trb.cascadeRlfEvents.GetAll()
}
