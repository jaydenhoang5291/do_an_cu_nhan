package logger

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CSVExporter handles exporting timer and handover events to CSV
type CSVExporter struct {
	outputDir string
}

// NewCSVExporter creates a new CSV exporter
func NewCSVExporter(outputDir string) *CSVExporter {
	if outputDir == "" {
		outputDir = "./test-results"
	}
	// Create directory if not exists
	os.MkdirAll(outputDir, 0755)
	return &CSVExporter{
		outputDir: outputDir,
	}
}

// ExportTimerEvents exports timer events to NDJSON file
func (ce *CSVExporter) ExportTimerEventsNDJSON(events []TimerEventEntry, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/timer-events-%d.ndjson", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create NDJSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to encode event: %w", err)
		}
	}

	return nil
}

// ExportHandoverTrialSummaries exports trial summaries to NDJSON file
func (ce *CSVExporter) ExportHandoverTrialSummariesNDJSON(summaries []HandoverTrialSummaryEntry, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/ho-summaries-%d.ndjson", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create NDJSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, summary := range summaries {
		if err := encoder.Encode(summary); err != nil {
			return fmt.Errorf("failed to encode summary: %w", err)
		}
	}

	return nil
}

// ExportHandoverTrialSummariesCSV exports one row per handover attempt.
func (ce *CSVExporter) ExportHandoverTrialSummariesCSV(summaries []HandoverTrialSummaryEntry, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/ho-summaries-%d.csv", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"ts", "start_ts", "trial_id", "pdr", "delay_ms", "ho_type", "overall_result",
		"fail_reason", "total_ho_ms", "source_gnb", "target_gnb",
		"rlf_offset_ms", "rsrp_during_ho", "reestab_target_gnb",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, s := range summaries {
		row := []string{
			s.Ts.Format(time.RFC3339Nano),
			formatOptionalTime(s.StartTs),
			strconv.Itoa(s.TrialId),
			fmt.Sprintf("%.2f", s.Pdr),
			fmt.Sprintf("%.2f", s.DelayMs),
			s.HoType,
			s.OverallResult,
			s.FailReason,
			fmt.Sprintf("%.3f", s.TotalHoMs),
			s.SourceGnb,
			s.TargetGnb,
			fmt.Sprintf("%.2f", s.RlfOffsetMs),
			fmt.Sprintf("%.2f", s.RsrpDuringHo),
			s.ReestabTargetGnb,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// PerTimerStats represents statistics for a specific timer
type PerTimerStats struct {
	Scenario      string
	Pdr           float64
	DelayMs       float64
	HoType        string
	TimerName     string
	TotalTrials   int
	SuccessCount  int
	TimeoutCount  int
	SuccessRate   float64
	ElapsedP50Ms  float64
	ElapsedP95Ms  float64
	ElapsedP99Ms  float64
	ElapsedMeanMs float64
}

// ExportTimerEventsCSV exports raw timer events to CSV
func (ce *CSVExporter) ExportTimerEventsCSV(events []TimerEventEntry, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/timer-events-%d.csv", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"ts", "start_ts", "stop_ts", "trial_id", "pdr", "delay_ms", "ho_type",
		"timer_name", "event", "elapsed_ms", "result", "stop_reason", "ho_trigger",
		"source_gnb", "target_gnb", "rsrp_at_trigger", "parent_timer", "rlf_detected",
		"packet_loss_count", "beam_id", "candidate_beam_id", "beam_recovery_ms", "n2_rtt_ms",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, e := range events {
		rlfStr := "false"
		if e.RlfDetected {
			rlfStr = "true"
		}
		row := []string{
			e.Ts.Format(time.RFC3339Nano),
			formatOptionalTime(e.StartTs),
			formatOptionalTime(e.StopTs),
			strconv.Itoa(e.TrialId),
			fmt.Sprintf("%.2f", e.Pdr),
			fmt.Sprintf("%.2f", e.DelayMs),
			e.HoType,
			e.TimerName,
			e.Event,
			fmt.Sprintf("%.2f", e.ElapsedMs),
			e.Result,
			e.StopReason,
			e.HoTrigger,
			e.SourceGnb,
			e.TargetGnb,
			fmt.Sprintf("%.2f", e.RsrpAtTrigger),
			e.ParentTimer,
			rlfStr,
			strconv.Itoa(e.PacketLossCount),
			e.BeamId,
			e.CandidateBeamId,
			fmt.Sprintf("%.2f", e.BeamRecoveryMs),
			fmt.Sprintf("%.2f", e.N2RttMs),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

type timerSummaryRow struct {
	trialId           int
	timerName         string
	startTs           string
	stopTs            string
	recordedElapsedMs string
	computedElapsedMs string
	event             string
	result            string
	expired           bool
}

// ExportTimerEventSummaryCSV exports one paired row per timer lifecycle and one
// HO row per handover attempt.
func (ce *CSVExporter) ExportTimerEventSummaryCSV(events []TimerEventEntry, summaries []HandoverTrialSummaryEntry, timerEventsFilename string) error {
	if timerEventsFilename == "" {
		timerEventsFilename = fmt.Sprintf("timer-events-%d.csv", time.Now().Unix())
	}
	stem := strings.TrimSuffix(filepath.Base(timerEventsFilename), filepath.Ext(timerEventsFilename))
	summaryFilename := fmt.Sprintf("timer_summary_%s.csv", stem)

	sortedEvents := make([]TimerEventEntry, len(events))
	copy(sortedEvents, events)
	sort.SliceStable(sortedEvents, func(i, j int) bool {
		a := sortedEvents[i]
		b := sortedEvents[j]
		if a.TrialId != b.TrialId {
			return a.TrialId < b.TrialId
		}
		if a.TimerName != b.TimerName {
			return a.TimerName < b.TimerName
		}
		return a.Ts.Before(b.Ts)
	})

	type timerKey struct {
		trialId   int
		timerName string
	}

	startQueues := make(map[timerKey][]TimerEventEntry)
	rows := make([]timerSummaryRow, 0, len(sortedEvents))

	flushStarts := func(key timerKey) {
		for _, start := range startQueues[key] {
			rows = append(rows, timerSummaryRow{
				trialId:   key.trialId,
				timerName: key.timerName,
				startTs:   formatOptionalTime(start.Ts),
				expired:   false,
			})
		}
		delete(startQueues, key)
	}

	var prevKey *timerKey
	for _, event := range sortedEvents {
		key := timerKey{trialId: event.TrialId, timerName: event.TimerName}
		if prevKey != nil && *prevKey != key {
			flushStarts(*prevKey)
		}
		keyCopy := key
		prevKey = &keyCopy

		switch event.Event {
		case "start":
			startQueues[key] = append(startQueues[key], event)
		case "stop", "timeout", "cancel", "restart":
			startTs := ""
			computedElapsedMs := ""
			if queue := startQueues[key]; len(queue) > 0 {
				start := queue[0]
				startQueues[key] = queue[1:]
				startTs = formatOptionalTime(start.Ts)
				computedElapsedMs = formatMs(event.Ts.Sub(start.Ts).Seconds() * 1000)
			}

			rows = append(rows, timerSummaryRow{
				trialId:           event.TrialId,
				timerName:         event.TimerName,
				startTs:           startTs,
				stopTs:            formatOptionalTime(event.Ts),
				recordedElapsedMs: formatMs(event.ElapsedMs),
				computedElapsedMs: computedElapsedMs,
				event:             event.Event,
				result:            event.Result,
				expired:           event.Event == "timeout",
			})
		}
	}
	if prevKey != nil {
		flushStarts(*prevKey)
	}

	for _, summary := range summaries {
		event := "timeout"
		expired := true
		if summary.OverallResult == "success" {
			event = "stop"
			expired = false
		}
		stopTs := summary.Ts
		startTs := summary.StartTs
		if startTs.IsZero() {
			startTs = stopTs.Add(-time.Duration(summary.TotalHoMs * float64(time.Millisecond)))
		}
		rows = append(rows, timerSummaryRow{
			trialId:           summary.TrialId,
			timerName:         "HO",
			startTs:           formatOptionalTime(startTs),
			stopTs:            formatOptionalTime(stopTs),
			recordedElapsedMs: formatMs(summary.TotalHoMs),
			computedElapsedMs: formatMs(stopTs.Sub(startTs).Seconds() * 1000),
			event:             event,
			result:            summary.OverallResult,
			expired:           expired,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].trialId != rows[j].trialId {
			return rows[i].trialId < rows[j].trialId
		}
		return rows[i].timerName < rows[j].timerName
	})

	file, err := os.Create(summaryFilename)
	if err != nil {
		return fmt.Errorf("failed to create timer summary CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"trial_id", "timer_name", "start_ts", "stop_ts", "recorded_elapsed_ms",
		"computed_elapsed_ms", "event", "result", "expired",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writer.Write([]string{
			strconv.Itoa(row.trialId),
			row.timerName,
			row.startTs,
			row.stopTs,
			row.recordedElapsedMs,
			row.computedElapsedMs,
			row.event,
			row.result,
			formatTitleBool(row.expired),
		}); err != nil {
			return err
		}
	}

	return nil
}

func formatOptionalTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339Nano)
}

func formatMs(value float64) string {
	if value == 0 {
		return "0.0"
	}
	formatted := strconv.FormatFloat(value, 'f', 3, 64)
	formatted = strings.TrimRight(formatted, "0")
	if strings.HasSuffix(formatted, ".") {
		formatted += "0"
	}
	return formatted
}

func formatTitleBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

// ExportPerTimerStats exports per-timer statistics to CSV
func (ce *CSVExporter) ExportPerTimerStats(stats []PerTimerStats, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/results-per-timer-%d.csv", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"scenario", "pdr", "delay_ms", "ho_type", "timer_name", "total_trials",
		"success_count", "timeout_count", "success_rate",
		"elapsed_p50_ms", "elapsed_p95_ms", "elapsed_p99_ms", "elapsed_mean_ms",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write rows
	for _, s := range stats {
		row := []string{
			s.Scenario,
			fmt.Sprintf("%.2f", s.Pdr),
			fmt.Sprintf("%.2f", s.DelayMs),
			s.HoType,
			s.TimerName,
			strconv.Itoa(s.TotalTrials),
			strconv.Itoa(s.SuccessCount),
			strconv.Itoa(s.TimeoutCount),
			fmt.Sprintf("%.4f", s.SuccessRate),
			fmt.Sprintf("%.2f", s.ElapsedP50Ms),
			fmt.Sprintf("%.2f", s.ElapsedP95Ms),
			fmt.Sprintf("%.2f", s.ElapsedP99Ms),
			fmt.Sprintf("%.2f", s.ElapsedMeanMs),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// TrialSummaryStats represents trial summary statistics
type TrialSummaryStats struct {
	Scenario      string
	Pdr           float64
	DelayMs       float64
	HoType        string
	TrialId       int
	OverallResult string
	FailReason    string
	TotalHoMs     float64
	SourceGnb     string
	TargetGnb     string
}

// ExportTrialSummaryStats exports trial summary statistics to CSV
func (ce *CSVExporter) ExportTrialSummaryStats(stats []TrialSummaryStats, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/results-trial-summary-%d.csv", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"scenario", "pdr", "delay_ms", "ho_type", "trial_id",
		"overall_result", "fail_reason", "total_ho_ms", "source_gnb", "target_gnb",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write rows
	for _, s := range stats {
		row := []string{
			s.Scenario,
			fmt.Sprintf("%.2f", s.Pdr),
			fmt.Sprintf("%.2f", s.DelayMs),
			s.HoType,
			strconv.Itoa(s.TrialId),
			s.OverallResult,
			s.FailReason,
			fmt.Sprintf("%.3f", s.TotalHoMs),
			s.SourceGnb,
			s.TargetGnb,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// T304BreakpointStats represents T304 breakpoint statistics
type T304BreakpointStats struct {
	DelayMs      float64
	TotalTrials  int
	TimeoutCount int
	TimeoutRate  float64
}

// ExportT304BreakpointStats exports T304 breakpoint statistics to CSV (Scenario 2)
func (ce *CSVExporter) ExportT304BreakpointStats(stats []T304BreakpointStats, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/results-t304-breakpoint-%d.csv", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{"delay_ms", "total_trials", "timeout_count", "timeout_rate"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write rows
	for _, s := range stats {
		row := []string{
			fmt.Sprintf("%.2f", s.DelayMs),
			strconv.Itoa(s.TotalTrials),
			strconv.Itoa(s.TimeoutCount),
			fmt.Sprintf("%.4f", s.TimeoutRate),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// CascadeRlfStats represents cascade RLF statistics
type CascadeRlfStats struct {
	Pdr              float64
	DelayMs          float64
	TrialId          int
	RlfOffsetMs      float64
	RsrpDuringHo     float64
	T310Result       string
	T311Result       string
	T301Result       string
	OverallResult    string
	ReestabTargetGnb string
}

// ExportCascadeRlfStats exports cascade RLF statistics to CSV (Scenario 3)
func (ce *CSVExporter) ExportCascadeRlfStats(stats []CascadeRlfStats, filename string) error {
	if filename == "" {
		filename = fmt.Sprintf("%s/results-cascade-%d.csv", ce.outputDir, time.Now().Unix())
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{
		"pdr", "delay_ms", "trial_id", "rlf_offset_ms", "rsrp_during_ho",
		"t310_result", "t311_result", "t301_result", "overall_result", "reestab_target_gnb",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write rows
	for _, s := range stats {
		row := []string{
			fmt.Sprintf("%.2f", s.Pdr),
			fmt.Sprintf("%.2f", s.DelayMs),
			strconv.Itoa(s.TrialId),
			fmt.Sprintf("%.2f", s.RlfOffsetMs),
			fmt.Sprintf("%.2f", s.RsrpDuringHo),
			s.T310Result,
			s.T311Result,
			s.T301Result,
			s.OverallResult,
			s.ReestabTargetGnb,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// CalculatePercentile calculates the nth percentile from a sorted list of values
func CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if percentile < 0 || percentile > 100 {
		return 0
	}

	// Sort values (assumes they're already sorted in real usage)
	// index = (percentile / 100) * (len - 1)
	index := percentile / 100.0 * float64(len(values)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(values) {
		return values[lower]
	}

	// Linear interpolation
	fraction := index - float64(lower)
	return values[lower]*(1-fraction) + values[upper]*fraction
}
