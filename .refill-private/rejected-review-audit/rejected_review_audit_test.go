package rejectedreviewaudit_test

import (
	"ovencheck/internal/core"
	"ovencheck/internal/review"
	"testing"
)

func TestRejectedDecisionSurvivesReloadInAuditEvidence(t *testing.T) {
	path := t.TempDir() + "/state.json"
	s, err := core.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create("退回留痕", "K-8", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	stages := []core.HeatingStage{{Sequence: 1, TargetCelsius: 100, MaxRampCelsiusPerHour: 30, HoldMinutes: 10, ToleranceCelsius: 5, SensorIDs: []string{"S1"}}}
	if err := s.AddStages(b.ID, b.Version, stages); err != nil {
		t.Fatal(err)
	}
	b, _ = s.Get(b.ID)
	if err := s.Freeze(b.ID, b.Version); err != nil {
		t.Fatal(err)
	}
	b, _ = s.Get(b.ID)
	if _, err := review.New(s).Decide(b.ID, b.Version, "审核员乙", core.StatusRejected, "证据不足，退回补充"); err != nil {
		t.Fatal(err)
	}
	reloaded, err := core.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	stored, ok := reloaded.Get(b.ID)
	if !ok {
		t.Fatal("batch missing after reload")
	}
	for _, line := range review.AuditLines(stored) {
		if line.Kind == "review" && line.Actor == "审核员乙" && line.Text == "证据不足，退回补充" {
			return
		}
	}
	t.Fatal("rejected decision lost reviewer and comment after reload")
}
