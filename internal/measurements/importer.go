package measurements

import (
	"fmt"
	"ovencheck/internal/core"
	"sort"
	"time"
)

type ImportResult struct {
	Accepted int                       `json:"accepted"`
	Rejected int                       `json:"rejected"`
	Errors   []string                  `json:"errors"`
	Readings []core.TemperatureReading `json:"readings"`
}

func Import(batch core.KilnBatch, inputs []Input) ImportResult {
	result := ImportResult{Errors: []string{}, Readings: []core.TemperatureReading{}}
	valid := make([]core.TemperatureReading, 0, len(inputs))
	for i, input := range inputs {
		rs, err := Normalize(batch, []Input{input})
		if err != nil {
			result.Rejected++
			result.Errors = append(result.Errors, fmt.Sprintf("第 %d 行: %s", i+1, err.Error()))
			continue
		}
		valid = append(valid, rs...)
	}
	if err := ValidateOrder(batch.Readings, valid); err != nil {
		result.Rejected += len(valid)
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	if err := CheckTimestampRange(valid, 0, time.Now()); err != nil {
		result.Rejected += len(valid)
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.Accepted = len(valid)
	result.Readings = SortStable(valid)
	return result
}
func CheckTimestampRange(rs []core.TemperatureReading, maxAge time.Duration, now time.Time) error {
	for _, r := range rs {
		if r.RecordedAt.After(now.Add(time.Second)) {
			return fmt.Errorf("读数时间不能晚于当前时间")
		}
		if maxAge > 0 && r.RecordedAt.Before(now.Add(-maxAge)) {
			return fmt.Errorf("读数超出允许追溯窗口")
		}
	}
	return nil
}
func SortStable(rs []core.TemperatureReading) []core.TemperatureReading {
	out := append([]core.TemperatureReading{}, rs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RecordedAt.Equal(out[j].RecordedAt) {
			return out[i].Sequence < out[j].Sequence
		}
		return out[i].RecordedAt.Before(out[j].RecordedAt)
	})
	return out
}
