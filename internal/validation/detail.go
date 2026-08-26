package validation

import (
	"fmt"
	"math"
	"ovencheck/internal/core"
	"ovencheck/internal/measurements"
	"sort"
	"time"
)

type SensorResult struct {
	SensorID      string    `json:"sensorID"`
	Samples       int       `json:"samples"`
	StartCelsius  float64   `json:"startCelsius"`
	EndCelsius    float64   `json:"endCelsius"`
	MaxRate       float64   `json:"maxRate"`
	HoldDeviation float64   `json:"holdDeviation"`
	Findings      []Finding `json:"findings"`
}

func CheckSensor(stage core.HeatingStage, rs []core.TemperatureReading) SensorResult {
	out := SensorResult{Findings: []Finding{}, SensorID: ""}
	if len(rs) == 0 {
		return out
	}
	out.SensorID = rs[0].SensorID
	out.Samples = len(rs)
	sort.Slice(rs, func(i, j int) bool { return rs[i].RecordedAt.Before(rs[j].RecordedAt) })
	out.StartCelsius = rs[0].Celsius
	out.EndCelsius = rs[len(rs)-1].Celsius
	out.MaxRate = 0
	for i := 1; i < len(rs); i++ {
		h := rs[i].RecordedAt.Sub(rs[i-1].RecordedAt).Hours()
		if h <= 0 {
			continue
		}
		rate := (rs[i].Celsius - rs[i-1].Celsius) / h
		if rate > out.MaxRate {
			out.MaxRate = rate
		}
		if rate > stage.MaxRampCelsiusPerHour {
			out.Findings = append(out.Findings, Finding{Code: "RAMP_HIGH", Severity: "error", Message: fmt.Sprintf("%s 速率 %.2f 超限", out.SensorID, rate), SensorID: out.SensorID, Value: rate})
		}
	}
	out.HoldDeviation = out.EndCelsius - stage.TargetCelsius
	if math.Abs(out.HoldDeviation) > stage.ToleranceCelsius {
		out.Findings = append(out.Findings, Finding{Code: "TARGET_DEVIATION", Severity: "error", Message: fmt.Sprintf("%s 偏差 %.2f 超出容差", out.SensorID, out.HoldDeviation), SensorID: out.SensorID, Value: out.HoldDeviation})
	}
	return out
}
func StageWindow(batch core.KilnBatch, stageID string) (time.Time, time.Time, bool) {
	var start, end time.Time
	found := false
	for _, r := range batch.Readings {
		if r.StageID != stageID {
			continue
		}
		if !found || r.RecordedAt.Before(start) {
			start = r.RecordedAt
		}
		if !found || r.RecordedAt.After(end) {
			end = r.RecordedAt
		}
		found = true
	}
	return start, end, found
}
func DetailedStage(batch core.KilnBatch, stage core.HeatingStage) []SensorResult {
	rs := measurements.StageReadings(batch, stage.ID)
	by := map[string][]core.TemperatureReading{}
	for _, r := range rs {
		by[r.SensorID] = append(by[r.SensorID], r)
	}
	out := []SensorResult{}
	for _, sid := range stage.SensorIDs {
		group := by[sid]
		result := CheckSensor(stage, group)
		result.SensorID = sid
		out = append(out, result)
	}
	return out
}
