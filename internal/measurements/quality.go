package measurements

import (
	"fmt"
	"math"
	"ovencheck/internal/core"
	"sort"
	"time"
)

type QualityIssue struct {
	Code      string    `json:"code"`
	SensorID  string    `json:"sensorID,omitempty"`
	ReadingID string    `json:"readingID,omitempty"`
	Message   string    `json:"message"`
	At        time.Time `json:"at,omitempty"`
}

func Inspect(batch core.KilnBatch) []QualityIssue {
	issues := []QualityIssue{}
	knownStages := map[string]core.HeatingStage{}
	for _, stage := range batch.Stages {
		knownStages[stage.ID] = stage
	}
	seen := map[string]string{}
	for _, reading := range batch.Readings {
		stage, ok := knownStages[reading.StageID]
		if !ok {
			issues = append(issues, QualityIssue{Code: "UNKNOWN_STAGE", ReadingID: reading.ID, Message: "读数引用了不存在的阶段", At: reading.RecordedAt})
			continue
		}
		if _, ok := core.SensorSet(stage)[reading.SensorID]; !ok {
			issues = append(issues, QualityIssue{Code: "UNKNOWN_SENSOR", ReadingID: reading.ID, SensorID: reading.SensorID, Message: "读数测点不在阶段配置中", At: reading.RecordedAt})
		}
		if math.IsNaN(reading.Celsius) || math.IsInf(reading.Celsius, 0) {
			issues = append(issues, QualityIssue{Code: "INVALID_NUMBER", ReadingID: reading.ID, Message: "温度必须是有限数字", At: reading.RecordedAt})
		}
		key := reading.StageID + "/" + reading.SensorID + "/" + reading.RecordedAt.UTC().Format(time.RFC3339Nano)
		if old, exists := seen[key]; exists {
			issues = append(issues, QualityIssue{Code: "DUPLICATE", ReadingID: reading.ID, SensorID: reading.SensorID, Message: fmt.Sprintf("与读数 %s 重复", old), At: reading.RecordedAt})
		} else {
			seen[key] = reading.ID
		}
	}
	return issues
}

func SensorTimeline(rs []core.TemperatureReading) map[string][]core.TemperatureReading {
	result := map[string][]core.TemperatureReading{}
	for _, reading := range rs {
		result[reading.SensorID] = append(result[reading.SensorID], reading)
	}
	for sensor := range result {
		sort.SliceStable(result[sensor], func(i, j int) bool {
			return result[sensor][i].RecordedAt.Before(result[sensor][j].RecordedAt)
		})
	}
	return result
}

func TemperatureRange(rs []core.TemperatureReading) (float64, float64, bool) {
	if len(rs) == 0 {
		return 0, 0, false
	}
	min, max := rs[0].Celsius, rs[0].Celsius
	for _, reading := range rs[1:] {
		if reading.Celsius < min {
			min = reading.Celsius
		}
		if reading.Celsius > max {
			max = reading.Celsius
		}
	}
	return min, max, true
}
