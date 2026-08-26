package review

import (
	"ovencheck/internal/core"
	"ovencheck/internal/validation"
	"strings"
)

func StatusMessage(batch core.KilnBatch, report validation.BatchReport) string {
	if batch.Status == core.StatusApproved {
		return "批次已批准，数据已冻结"
	}
	if !report.Compliant {
		return "存在未通过阶段，需要整改后复测"
	}
	if batch.Status == core.StatusCollect {
		return "采集完成后可提交质量审核"
	}
	return "批次等待审核"
}

func NormalizeComment(comment string) string {
	lines := strings.Split(strings.TrimSpace(comment), "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}
