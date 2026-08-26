package measurements

import (
	"ovencheck/internal/core"
	"testing"
)

func TestNormalize(t *testing.T) {
	b := core.KilnBatch{ID: "b", Stages: []core.HeatingStage{{ID: "s", SensorIDs: []string{"x"}}}}
	rs, err := Normalize(b, []Input{{StageID: "s", SensorID: "x", RecordedAt: "2026-01-01T00:00:00Z", Celsius: 20, Unit: "C"}})
	if err != nil || len(rs) != 1 {
		t.Fatal(err)
	}
}
