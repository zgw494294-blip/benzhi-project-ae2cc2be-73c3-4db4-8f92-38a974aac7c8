package shallowrollbackstate_test

import (
	"os"
	"ovencheck/internal/core"
	"path/filepath"
	"testing"
)

func TestFailedSnapshotWritePreservesPendingRetest(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	store, err := core.NewStore(filepath.Join(stateDir, "snapshot.json"))
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	batch, err := store.Create("回滚边界批次", "K-ROLLBACK", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatalf("创建批次失败: %v", err)
	}
	stageID := "stage-rollback"
	stage := core.HeatingStage{
		ID:                    stageID,
		Sequence:              1,
		TargetCelsius:         300,
		MaxRampCelsiusPerHour: 100,
		HoldMinutes:           30,
		ToleranceCelsius:      10,
		SensorIDs:             []string{"TC-1"},
	}
	if err := store.AddStages(batch.ID, batch.Version, []core.HeatingStage{stage}); err != nil {
		t.Fatalf("配置阶段失败: %v", err)
	}
	batch, _ = store.Get(batch.ID)
	if err := store.Freeze(batch.ID, batch.Version); err != nil {
		t.Fatalf("冻结批次失败: %v", err)
	}
	batch, _ = store.Get(batch.ID)
	if err := store.AddAction(batch.ID, batch.Version, core.DeviationAction{
		StageID:        stageID,
		Kind:           "equipment-check",
		Reason:         "热电偶接线异常",
		ActionText:     "重新压接并要求定向复测",
		PerformedBy:    "操作员",
		RetestRequired: true,
	}); err != nil {
		t.Fatalf("登记复测处置失败: %v", err)
	}
	before, _ := store.Get(batch.ID)
	if _, ok := before.PendingRetests[stageID]; !ok {
		t.Fatal("测试前置条件失败：待复测状态不存在")
	}

	if err := os.RemoveAll(stateDir); err != nil {
		t.Fatalf("移除快照目录失败: %v", err)
	}
	if err := os.WriteFile(stateDir, []byte("invalidated snapshot resource"), 0600); err != nil {
		t.Fatalf("制造快照资源失效失败: %v", err)
	}

	err = store.ClearRetest(batch.ID, before.Version, stageID)
	if err == nil {
		t.Fatal("快照保存失败时 ClearRetest 应返回错误")
	}
	after, ok := store.Get(batch.ID)
	if !ok {
		t.Fatal("保存失败后批次从内存中消失")
	}
	if _, ok := after.PendingRetests[stageID]; !ok {
		t.Fatalf("保存失败仍删除了待复测状态；返回错误后内存聚合应保持版本 %d 的原状", before.Version)
	}
	if after.Version != before.Version || after.Status != before.Status {
		t.Fatalf("保存失败后顶层状态发生变化: before=(%d,%s) after=(%d,%s)", before.Version, before.Status, after.Version, after.Status)
	}
}
