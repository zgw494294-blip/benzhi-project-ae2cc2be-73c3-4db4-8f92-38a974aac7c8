package stageconfigalias

import (
	"testing"

	"ovencheck/internal/core"
)

func TestAddStagesDetachesCallerConfiguration(t *testing.T) {
	store, err := core.NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := store.Create("批次", "K-1", "耐材", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	stages := []core.HeatingStage{{
		Sequence:              1,
		TargetCelsius:         120,
		MaxRampCelsiusPerHour: 10,
		HoldMinutes:           30,
		ToleranceCelsius:      5,
		SensorIDs:             []string{"T1"},
	}}
	if err := store.AddStages(batch.ID, batch.Version, stages); err != nil {
		t.Fatal(err)
	}

	// The caller reuses its request buffer after the command has returned.
	stages[0].Sequence = 99
	stages[0].SensorIDs[0] = "T9"

	got, ok := store.Get(batch.ID)
	if !ok {
		t.Fatal("batch disappeared")
	}
	if len(got.Stages) != 1 || got.Stages[0].Sequence != 1 || got.Stages[0].SensorIDs[0] != "T1" {
		t.Fatalf("caller-owned stage configuration contaminated store: %+v", got.Stages)
	}
}
