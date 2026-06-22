package scenarios

import (
	"fmt"
	"strings"
	"time"

	"stormsim/internal/core/monitor"
)

type HoPhaseCSVEntry struct {
	Step int
	Proc *monitor.HoProcedure
}

func buildHoPhaseCSV(entries []HoPhaseCSVEntry) string {
	var b strings.Builder
	b.WriteString("step,prepare,execute,complete,fail\n")
	for _, entry := range entries {
		b.WriteString(buildHoPhaseCSVRow(entry))
		b.WriteByte('\n')
	}
	return b.String()
}

func buildHoPhaseCSVRow(entry HoPhaseCSVEntry) string {
	return fmt.Sprintf(
		"%d,%s,%s,%s,%s",
		entry.Step,
		formatPhaseTime(entry.Proc.GetPrepareStart()),
		formatPhaseTime(entry.Proc.GetExecuteStart()),
		formatPhaseTime(entry.Proc.GetCompleteStart()),
		formatPhaseTime(entry.Proc.GetFailStart()),
	)
}

func formatPhaseTime(t time.Time) string {
	if t.IsZero() {
		return "0"
	}
	return t.UTC().Format(time.RFC3339Nano)
}
