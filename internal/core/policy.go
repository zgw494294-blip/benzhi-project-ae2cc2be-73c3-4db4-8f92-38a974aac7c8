package core

import (
	"errors"
	"sort"
	"time"
)

type Transition struct {
	From   string
	To     string
	At     time.Time
	Actor  string
	Reason string
}

func CanEditPlan(b KilnBatch) error {
	if b.Status != StatusDraft {
		return errors.New("阶段计划已冻结，不能修改")
	}
	if b.FrozenAt != nil {
		return errors.New("已记录冻结时间")
	}
	return nil
}
func CanCollect(b KilnBatch) error {
	if b.Status != StatusCollect && b.Status != StatusReview {
		return errors.New("批次尚未进入采集状态")
	}
	if b.FrozenAt == nil {
		return errors.New("缺少冻结计划")
	}
	return nil
}
func CanReview(b KilnBatch) error {
	if b.Status == StatusApproved {
		return errors.New("批次已经批准")
	}
	if b.FrozenAt == nil {
		return errors.New("没有冻结计划")
	}
	return nil
}
func CanApprove(b KilnBatch) error {
	if b.Status == StatusApproved {
		return errors.New("重复批准")
	}
	if len(b.Stages) == 0 {
		return errors.New("没有阶段证据")
	}
	if len(b.Readings) == 0 {
		return errors.New("没有温度证据")
	}
	return nil
}
func OrderedStages(b KilnBatch) []HeatingStage {
	out := append([]HeatingStage{}, b.Stages...)
	sort.Slice(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}
func SensorSet(stage HeatingStage) map[string]struct{} {
	out := map[string]struct{}{}
	for _, id := range stage.SensorIDs {
		out[id] = struct{}{}
	}
	return out
}
