package validation

import (
	"fmt"
	"ovencheck/internal/core"
	"ovencheck/internal/measurements"
	"sort"
	"time"
)

type Finding struct {
	Code     string  `json:"code"`
	Severity string  `json:"severity"`
	Message  string  `json:"message"`
	SensorID string  `json:"sensorID,omitempty"`
	Value    float64 `json:"value,omitempty"`
}
type StageReport struct {
	StageID         string                    `json:"stageID"`
	Sequence        int                       `json:"sequence"`
	Compliant       bool                      `json:"compliant"`
	DurationMinutes float64                   `json:"durationMinutes"`
	Findings        []Finding                 `json:"findings"`
	Summary         measurements.Summary      `json:"summary"`
	Rates           []measurements.RateSample `json:"rates"`
	Gaps            []measurements.Gap        `json:"gaps"`
	Sensors         []SensorResult            `json:"sensors"`
	HoldWindows     map[string]HoldWindow     `json:"holdWindows"`
}
type BatchReport struct {
	BatchID     string        `json:"batchId"`
	Compliant   bool          `json:"compliant"`
	Stages      []StageReport `json:"stages"`
	GeneratedAt time.Time     `json:"generatedAt"`
}

func Evaluate(batch core.KilnBatch) BatchReport {
	out := BatchReport{BatchID: batch.ID, Compliant: true, GeneratedAt: time.Now(), Stages: []StageReport{}}
	for _, st := range batch.Stages {
		rs := measurements.StageReadings(batch, st.ID)
		rep := EvaluateStage(st, rs)
		out.Stages = append(out.Stages, rep)
		if !rep.Compliant {
			out.Compliant = false
		}
	}
	return out
}

// EvaluateStage judges a single stage against an explicit set of readings.
// It is the per-stage building block of Evaluate and is exposed so callers can
// evaluate a subset of readings (for example the post-action retest window)
// against the same rules without being affected by readings outside that window.
func EvaluateStage(stage core.HeatingStage, rs []core.TemperatureReading) StageReport {
	rep := StageReport{StageID: stage.ID, Sequence: stage.Sequence, Compliant: true, Findings: []Finding{}, Summary: measurements.Summarize(rs), Rates: measurements.Rates(rs), Gaps: measurements.Gaps(rs, 60), Sensors: DetailedStageReadings(stage, rs), HoldWindows: HoldWindows(stage, rs)}
	for _, gap := range rep.Gaps {
		rep.Compliant = false
		rep.Findings = append(rep.Findings, Finding{Code: "MISSING_GAP", Severity: "error", Message: fmt.Sprintf("测点 %s 存在 %.1f 分钟缺口", gap.SensorID, gap.Minutes), SensorID: gap.SensorID, Value: gap.Minutes})
	}
	bySensor := map[string][]core.TemperatureReading{}
	for _, r := range rs {
		bySensor[r.SensorID] = append(bySensor[r.SensorID], r)
	}
	for _, sid := range stage.SensorIDs {
		seq := bySensor[sid]
		if len(seq) == 0 {
			rep.Compliant = false
			rep.Findings = append(rep.Findings, Finding{Code: "MISSING", Severity: "error", Message: fmt.Sprintf("测点 %s 缺少读数", sid), SensorID: sid})
			continue
		}
		sort.Slice(seq, func(i, j int) bool { return seq[i].RecordedAt.Before(seq[j].RecordedAt) })
		if len(seq) > 1 {
			rep.DurationMinutes = seq[len(seq)-1].RecordedAt.Sub(seq[0].RecordedAt).Minutes()
		}
		for i := 1; i < len(seq); i++ {
			hours := seq[i].RecordedAt.Sub(seq[i-1].RecordedAt).Hours()
			if hours > 0 {
				rate := (seq[i].Celsius - seq[i-1].Celsius) / hours
				if rate > stage.MaxRampCelsiusPerHour {
					rep.Compliant = false
					rep.Findings = append(rep.Findings, Finding{Code: "RAMP_HIGH", Severity: "error", Message: fmt.Sprintf("测点 %s 升温速率 %.1f°C/h 超过 %.1f", sid, rate, stage.MaxRampCelsiusPerHour), SensorID: sid, Value: rate})
				}
			}
		}
		last := seq[len(seq)-1].Celsius
		if last < stage.TargetCelsius-stage.ToleranceCelsius {
			rep.Compliant = false
			rep.Findings = append(rep.Findings, Finding{Code: "UNDER_TEMP", Severity: "error", Message: fmt.Sprintf("测点 %s 末温 %.1f°C 低于目标 %.1f±%.1f", sid, last, stage.TargetCelsius, stage.ToleranceCelsius), SensorID: sid, Value: last})
		}
		if stage.HoldMinutes > 0 && rep.DurationMinutes < float64(stage.HoldMinutes) {
			rep.Compliant = false
			rep.Findings = append(rep.Findings, Finding{Code: "HOLD_SHORT", Severity: "error", Message: fmt.Sprintf("保温持续 %.1f 分钟，要求 %d 分钟", rep.DurationMinutes, stage.HoldMinutes), Value: rep.DurationMinutes})
		}
	}
	return rep
}
