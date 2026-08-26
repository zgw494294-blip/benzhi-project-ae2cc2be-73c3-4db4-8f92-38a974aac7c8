package measurements

import (
	"fmt"
	"math"
	"ovencheck/internal/core"
	"sort"
	"strings"
	"time"
)

type Input struct {
	StageID    string  `json:"stageID"`
	SensorID   string  `json:"sensorID"`
	RecordedAt string  `json:"recordedAt"`
	Celsius    float64 `json:"celsius"`
	Unit       string  `json:"unit"`
	Source     string  `json:"source"`
}

func Normalize(batch core.KilnBatch, inputs []Input) ([]core.TemperatureReading, error) {
	stageSensors := map[string]map[string]bool{}
	for _, st := range batch.Stages {
		stageSensors[st.ID] = map[string]bool{}
		for _, sid := range st.SensorIDs {
			stageSensors[st.ID][sid] = true
		}
	}
	out := make([]core.TemperatureReading, 0, len(inputs))
	seen := map[string]bool{}
	for _, in := range inputs {
		if math.IsNaN(in.Celsius) || math.IsInf(in.Celsius, 0) {
			return nil, fmt.Errorf("温度必须是有限数字")
		}
		if in.Unit != "" && strings.ToUpper(in.Unit) != "C" && strings.ToUpper(in.Unit) != "CELSIUS" {
			return nil, fmt.Errorf("温度单位必须为 C")
		}
		if !stageSensors[in.StageID][in.SensorID] {
			return nil, fmt.Errorf("测点 %s 不属于阶段 %s", in.SensorID, in.StageID)
		}
		t, err := time.Parse(time.RFC3339, in.RecordedAt)
		if err != nil {
			return nil, fmt.Errorf("时间戳格式错误: %w", err)
		}
		key := in.StageID + "/" + in.SensorID + "/" + t.UTC().Format(time.RFC3339Nano)
		if seen[key] {
			return nil, fmt.Errorf("存在重复读数")
		}
		seen[key] = true
		out = append(out, core.TemperatureReading{BatchID: batch.ID, StageID: in.StageID, SensorID: in.SensorID, RecordedAt: t.UTC(), Celsius: in.Celsius, Source: in.Source})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RecordedAt.Before(out[j].RecordedAt) })
	return out, nil
}

func ValidateOrder(existing []core.TemperatureReading, incoming []core.TemperatureReading) error {
	all := append(append([]core.TemperatureReading{}, existing...), incoming...)
	sort.Slice(all, func(i, j int) bool { return all[i].RecordedAt.Before(all[j].RecordedAt) })
	for i := 1; i < len(all); i++ {
		if all[i].RecordedAt.Equal(all[i-1].RecordedAt) && all[i].StageID == all[i-1].StageID && all[i].SensorID == all[i-1].SensorID {
			return fmt.Errorf("同一测点存在重复时间")
		}
	}
	return nil
}
