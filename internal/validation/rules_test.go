package validation

import (
	"ovencheck/internal/core"
	"testing"
	"time"
)

func TestEvaluateMissing(t *testing.T) {
	b := core.KilnBatch{ID: "b", Stages: []core.HeatingStage{{ID: "s", Sequence: 1, TargetCelsius: 100, SensorIDs: []string{"x"}}}}
	r := Evaluate(b)
	if r.Compliant || len(r.Stages[0].Findings) == 0 {
		t.Fatal("expected finding")
	}
	_ = time.Now()
}
