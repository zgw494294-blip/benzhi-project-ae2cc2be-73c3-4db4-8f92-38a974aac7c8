package measurements

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseCSV accepts stageID,sensorID,recordedAt,celsius,unit columns.
func ParseCSV(r io.Reader) ([]Input, error) {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, err
	}
	if len(header) < 5 {
		return nil, fmt.Errorf("CSV 列不足")
	}
	want := []string{"stageID", "sensorID", "recordedAt", "celsius", "unit"}
	for i, name := range want {
		if strings.TrimSpace(header[i]) != name {
			return nil, fmt.Errorf("CSV 第 %d 列必须为 %s", i+1, name)
		}
	}
	out := []Input{}
	for row := 1; ; row++ {
		v, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return nil, e
		}
		if len(v) < 5 {
			return nil, fmt.Errorf("第 %d 行列不足", row)
		}
		c, e := strconv.ParseFloat(v[3], 64)
		if e != nil {
			return nil, fmt.Errorf("第 %d 行温度无效", row)
		}
		out = append(out, Input{StageID: v[0], SensorID: v[1], RecordedAt: v[2], Celsius: c, Unit: v[4], Source: "csv"})
	}
	return out, nil
}
