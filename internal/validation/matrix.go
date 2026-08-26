package validation

import (
	"fmt"
	"ovencheck/internal/core"
	"ovencheck/internal/measurements"
	"sort"
	"time"
)

type RuleCell struct {
	StageID  string  `json:"stageID"`
	SensorID string  `json:"sensorID"`
	Rule     string  `json:"rule"`
	Passed   bool    `json:"passed"`
	Observed float64 `json:"observed"`
	Limit    float64 `json:"limit"`
	Message  string  `json:"message"`
}

type RuleMatrix struct {
	GeneratedAt time.Time  `json:"generatedAt"`
	Cells       []RuleCell `json:"cells"`
}

func BuildMatrix(batch core.KilnBatch) RuleMatrix {
	generated := time.Time{}
	if batch.FrozenAt != nil {
		generated = *batch.FrozenAt
	}
	for _, reading := range batch.Readings {
		if reading.RecordedAt.After(generated) {
			generated = reading.RecordedAt
		}
	}
	matrix := RuleMatrix{GeneratedAt: generated, Cells: []RuleCell{}}
	for _, stage := range core.OrderedStages(batch) {
		readings := measurements.StageReadings(batch, stage.ID)
		bySensor := map[string][]core.TemperatureReading{}
		for _, reading := range readings {
			bySensor[reading.SensorID] = append(bySensor[reading.SensorID], reading)
		}
		for _, sensor := range stage.SensorIDs {
			group := bySensor[sensor]
			if len(group) == 0 {
				matrix.Cells = append(matrix.Cells, RuleCell{StageID: stage.ID, SensorID: sensor, Rule: "presence", Message: "缺少测点读数"})
				continue
			}
			detail := CheckSensor(stage, group)
			matrix.Cells = append(matrix.Cells, RuleCell{StageID: stage.ID, SensorID: sensor, Rule: "ramp", Passed: detail.MaxRate <= stage.MaxRampCelsiusPerHour, Observed: detail.MaxRate, Limit: stage.MaxRampCelsiusPerHour, Message: fmt.Sprintf("最大升温速率 %.2f / %.2f", detail.MaxRate, stage.MaxRampCelsiusPerHour)})
			matrix.Cells = append(matrix.Cells, RuleCell{StageID: stage.ID, SensorID: sensor, Rule: "target", Passed: detail.HoldDeviation >= -stage.ToleranceCelsius && detail.HoldDeviation <= stage.ToleranceCelsius, Observed: detail.HoldDeviation, Limit: stage.ToleranceCelsius, Message: fmt.Sprintf("末温偏差 %.2f / ±%.2f", detail.HoldDeviation, stage.ToleranceCelsius)})
		}
	}
	sort.Slice(matrix.Cells, func(i, j int) bool {
		if matrix.Cells[i].StageID == matrix.Cells[j].StageID {
			if matrix.Cells[i].SensorID == matrix.Cells[j].SensorID {
				return matrix.Cells[i].Rule < matrix.Cells[j].Rule
			}
			return matrix.Cells[i].SensorID < matrix.Cells[j].SensorID
		}
		return matrix.Cells[i].StageID < matrix.Cells[j].StageID
	})
	return matrix
}

func (m RuleMatrix) Passed() bool {
	for _, cell := range m.Cells {
		if !cell.Passed {
			return false
		}
	}
	return len(m.Cells) > 0
}
