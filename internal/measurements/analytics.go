package measurements

import (
	"ovencheck/internal/core"
	"sort"
	"time"
)

type RateSample struct {
	SensorID       string    `json:"sensorID"`
	From           time.Time `json:"from"`
	To             time.Time `json:"to"`
	DeltaCelsius   float64   `json:"deltaCelsius"`
	Hours          float64   `json:"hours"`
	CelsiusPerHour float64   `json:"celsiusPerHour"`
}
type Gap struct {
	SensorID string    `json:"sensorID"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Minutes  float64   `json:"minutes"`
}

func Rates(rs []core.TemperatureReading) []RateSample {
	groups := map[string][]core.TemperatureReading{}
	for _, r := range rs {
		groups[r.SensorID] = append(groups[r.SensorID], r)
	}
	out := []RateSample{}
	for sid, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].RecordedAt.Before(g[j].RecordedAt) })
		for i := 1; i < len(g); i++ {
			h := g[i].RecordedAt.Sub(g[i-1].RecordedAt).Hours()
			if h <= 0 {
				continue
			}
			out = append(out, RateSample{SensorID: sid, From: g[i-1].RecordedAt, To: g[i].RecordedAt, DeltaCelsius: g[i].Celsius - g[i-1].Celsius, Hours: h, CelsiusPerHour: (g[i].Celsius - g[i-1].Celsius) / h})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].From.Before(out[j].From) })
	return out
}
func Gaps(rs []core.TemperatureReading, maxMinutes float64) []Gap {
	groups := map[string][]core.TemperatureReading{}
	for _, r := range rs {
		groups[r.SensorID] = append(groups[r.SensorID], r)
	}
	out := []Gap{}
	for sid, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].RecordedAt.Before(g[j].RecordedAt) })
		for i := 1; i < len(g); i++ {
			mins := g[i].RecordedAt.Sub(g[i-1].RecordedAt).Minutes()
			if mins > maxMinutes {
				out = append(out, Gap{SensorID: sid, Start: g[i-1].RecordedAt, End: g[i].RecordedAt, Minutes: mins})
			}
		}
	}
	return out
}
func Window(rs []core.TemperatureReading, start, end time.Time) []core.TemperatureReading {
	out := []core.TemperatureReading{}
	for _, r := range rs {
		if !r.RecordedAt.Before(start) && !r.RecordedAt.After(end) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RecordedAt.Before(out[j].RecordedAt) })
	return out
}
func LatestBySensor(rs []core.TemperatureReading) map[string]core.TemperatureReading {
	out := map[string]core.TemperatureReading{}
	for _, r := range rs {
		old, ok := out[r.SensorID]
		if !ok || r.RecordedAt.After(old.RecordedAt) {
			out[r.SensorID] = r
		}
	}
	return out
}
