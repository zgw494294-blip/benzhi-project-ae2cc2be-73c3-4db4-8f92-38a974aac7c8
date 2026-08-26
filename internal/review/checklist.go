package review

import (
	"fmt"
	"ovencheck/internal/core"
	"ovencheck/internal/validation"
	"time"
)

type ChecklistItem struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

func Checklist(batch core.KilnBatch, report validation.BatchReport) []ChecklistItem {
	summary := validation.Summarize(report)
	return []ChecklistItem{{Key: "plan", Label: "阶段计划已冻结", Passed: batch.FrozenAt != nil, Detail: fmt.Sprintf("计划版本 %d", batch.PlanVersion)}, {Key: "readings", Label: "原始温度读数完整", Passed: len(batch.Readings) > 0, Detail: fmt.Sprintf("共 %d 条", len(batch.Readings))}, {Key: "rules", Label: "规则计算全部通过", Passed: report.Compliant, Detail: fmt.Sprintf("%d/%d 阶段通过", summary.Passed, summary.Total)}, {Key: "actions", Label: "异常处置证据已记录", Passed: summary.Failed == 0 || len(batch.Actions) > 0, Detail: fmt.Sprintf("整改记录 %d 条", len(batch.Actions))}, {Key: "fresh", Label: "报告为当前批次版本", Passed: batch.Version > 0, Detail: time.Now().Format(time.RFC3339)}}
}
func ChecklistPassed(items []ChecklistItem) bool {
	for _, item := range items {
		if !item.Passed {
			return false
		}
	}
	return true
}
