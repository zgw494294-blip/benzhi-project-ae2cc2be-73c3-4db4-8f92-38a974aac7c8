package storesnapshotrace_test

import (
	"ovencheck/internal/core"
	"sync"
	"testing"
)

func TestStoreSnapshotIsDetachedDuringConcurrentUse(t *testing.T) {
	s, err := core.NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create("并发快照", "K-1", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	stages := []core.HeatingStage{{Sequence: 1, TargetCelsius: 100, MaxRampCelsiusPerHour: 30, HoldMinutes: 10, ToleranceCelsius: 5, SensorIDs: []string{"S1"}}}
	if err := s.AddStages(b.ID, b.Version, stages); err != nil {
		t.Fatal(err)
	}
	view, ok := s.Get(b.ID)
	if !ok {
		t.Fatal("batch missing")
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			view.Stages[0].TargetCelsius = float64(200 + i)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 1000; i++ {
			current, found := s.Get(b.ID)
			if found {
				_ = current.Stages[0].TargetCelsius
			}
		}
	}()
	close(start)
	wg.Wait()
	stored, _ := s.Get(b.ID)
	if stored.Stages[0].TargetCelsius != 100 {
		t.Fatalf("exported snapshot mutated store without versioned mutation: got %.0f", stored.Stages[0].TargetCelsius)
	}
}
