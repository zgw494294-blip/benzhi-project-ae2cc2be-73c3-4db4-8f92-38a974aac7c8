package validation

import (
	"ovencheck/internal/core"
	"sort"
	"time"
)

type HoldWindow struct {
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	Minutes         float64   `json:"minutes"`
	Average         float64   `json:"average"`
	Deviation       float64   `json:"deviation"`
	WithinTolerance bool      `json:"withinTolerance"`
}

func HoldWindows(stage core.HeatingStage, readings []core.TemperatureReading) map[string]HoldWindow {
	groups := map[string][]core.TemperatureReading{}
	for _, reading := range readings {
		groups[reading.SensorID] = append(groups[reading.SensorID], reading)
	}
	result := map[string]HoldWindow{}
	for sensor, group := range groups {
		sort.Slice(group, func(i, j int) bool { return group[i].RecordedAt.Before(group[j].RecordedAt) })
		if len(group) == 0 {
			continue
		}
		var total float64
		for _, reading := range group {
			total += reading.Celsius
		}
		start, end := group[0].RecordedAt, group[len(group)-1].RecordedAt
		average := total / float64(len(group))
		deviation := average - stage.TargetCelsius
		result[sensor] = HoldWindow{Start: start, End: end, Minutes: end.Sub(start).Minutes(), Average: average, Deviation: deviation, WithinTolerance: deviation >= -stage.ToleranceCelsius && deviation <= stage.ToleranceCelsius}
	}
	return result
}

func HoldPasses(stage core.HeatingStage, window HoldWindow) bool {
	if window.Minutes < float64(stage.HoldMinutes) {
		return false
	}
	return window.WithinTolerance
}

func HoldFindings(stage core.HeatingStage, readings []core.TemperatureReading) []Finding {
	findings := []Finding{}
	windows := HoldWindows(stage, readings)
	for _, sensor := range stage.SensorIDs {
		window, ok := windows[sensor]
		if !ok {
			findings = append(findings, Finding{Code: "MISSING", Severity: "error", Message: "保温窗口缺少测点 " + sensor, SensorID: sensor})
			continue
		}
		if window.Minutes < float64(stage.HoldMinutes) {
			findings = append(findings, Finding{Code: "HOLD_SHORT", Severity: "error", Message: "测点 " + sensor + " 保温时长不足", SensorID: sensor, Value: window.Minutes})
		}
		if !window.WithinTolerance {
			findings = append(findings, Finding{Code: "HOLD_DEVIATION", Severity: "error", Message: "测点 " + sensor + " 保温平均温度偏差超限", SensorID: sensor, Value: window.Deviation})
		}
	}
	return findings
}
