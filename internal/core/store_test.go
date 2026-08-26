package core

import "testing"

func TestStoreLifecycle(t *testing.T) {
	s, err := NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create("测试", "K-1", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	st := []HeatingStage{{Sequence: 1, TargetCelsius: 120, MaxRampCelsiusPerHour: 30, HoldMinutes: 10, ToleranceCelsius: 5, SensorIDs: []string{"S1"}}}
	if err = s.AddStages(b.ID, b.Version, st); err != nil {
		t.Fatal(err)
	}
	b, _ = s.Get(b.ID)
	if err = s.Freeze(b.ID, b.Version); err != nil {
		t.Fatal(err)
	}
	b, _ = s.Get(b.ID)
	if b.Status != StatusCollect {
		t.Fatal("not collecting")
	}
}
