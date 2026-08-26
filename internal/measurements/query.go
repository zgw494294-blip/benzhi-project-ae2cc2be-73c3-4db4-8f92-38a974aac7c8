package measurements

import (
	"ovencheck/internal/core"
	"sort"
	"time"
)

func StageReadings(batch core.KilnBatch, stageID string) []core.TemperatureReading {
	out := make([]core.TemperatureReading, 0)
	for _, r := range batch.Readings {
		if r.StageID == stageID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SensorID == out[j].SensorID {
			return out[i].RecordedAt.Before(out[j].RecordedAt)
		}
		return out[i].SensorID < out[j].SensorID
	})
	return out
}

type Summary struct {
	Count   int        `json:"count"`
	First   *time.Time `json:"first,omitempty"`
	Last    *time.Time `json:"last,omitempty"`
	Min     float64    `json:"min"`
	Max     float64    `json:"max"`
	Average float64    `json:"average"`
}

func Summarize(rs []core.TemperatureReading) Summary {
	if len(rs) == 0 {
		return Summary{}
	}
	s := Summary{Count: len(rs), Min: rs[0].Celsius, Max: rs[0].Celsius}
	var total float64
	for i, r := range rs {
		if r.Celsius < s.Min {
			s.Min = r.Celsius
		}
		if r.Celsius > s.Max {
			s.Max = r.Celsius
		}
		total += r.Celsius
		if i == 0 {
			x := r.RecordedAt
			s.First = &x
		}
		if i == len(rs)-1 {
			x := r.RecordedAt
			s.Last = &x
		}
	}
	s.Average = total / float64(len(rs))
	return s
}
