package core

import (
	"fmt"
	"sort"
)

func ValidateStage(stage HeatingStage) error {
	if stage.Sequence < 1 {
		return fmt.Errorf("阶段序号必须为正数")
	}
	if stage.TargetCelsius < 0 || stage.TargetCelsius > 1800 {
		return fmt.Errorf("目标温度超出范围")
	}
	if stage.MaxRampCelsiusPerHour <= 0 {
		return fmt.Errorf("升温速率必须为正数")
	}
	if stage.HoldMinutes < 0 {
		return fmt.Errorf("保温时长不能为负")
	}
	if stage.ToleranceCelsius < 0 {
		return fmt.Errorf("温度容差不能为负")
	}
	if len(stage.SensorIDs) == 0 {
		return fmt.Errorf("至少配置一个测点")
	}
	return nil
}

// ValidateStages checks constraints that only make sense across the complete plan.
func ValidateStages(stages []HeatingStage) error {
	if len(stages) == 0 {
		return fmt.Errorf("至少配置一个阶段")
	}
	ordered := append([]HeatingStage(nil), stages...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
	seenSeq := map[int]bool{}
	seenSensors := map[string]string{}
	for _, stage := range ordered {
		if seenSeq[stage.Sequence] {
			return fmt.Errorf("阶段序号 %d 重复", stage.Sequence)
		}
		seenSeq[stage.Sequence] = true
		if err := ValidateStage(stage); err != nil {
			return err
		}
		for _, sensor := range stage.SensorIDs {
			if sensor == "" {
				return fmt.Errorf("阶段 %d 包含空测点", stage.Sequence)
			}
			if previous, ok := seenSensors[sensor]; ok {
				return fmt.Errorf("测点 %s 在阶段 %s 和 %s 重复", sensor, previous, stage.ID)
			}
			seenSensors[sensor] = stage.ID
		}
	}
	for i := 1; i < len(ordered); i++ {
		if ordered[i].TargetCelsius < ordered[i-1].TargetCelsius {
			return fmt.Errorf("阶段 %d 目标温度低于阶段 %d", ordered[i].Sequence, ordered[i-1].Sequence)
		}
		if ordered[i].MaxRampCelsiusPerHour <= 0 || ordered[i].HoldMinutes < 0 || ordered[i].ToleranceCelsius < 0 {
			return fmt.Errorf("阶段 %d 的速率、保温时长或容差无效", ordered[i].Sequence)
		}
	}
	return nil
}

func PlanPreview(stages []HeatingStage) ([]HeatingStage, error) {
	if err := ValidateStages(stages); err != nil {
		return nil, err
	}
	out := append([]HeatingStage(nil), stages...)
	sort.Slice(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out, nil
}
func StatusLabel(status string) string {
	switch status {
	case StatusDraft:
		return "草稿"
	case StatusCollect:
		return "采集中"
	case StatusReview:
		return "待审核"
	case StatusApproved:
		return "已批准"
	case StatusRejected:
		return "已退回"
	default:
		return "未知"
	}
}
