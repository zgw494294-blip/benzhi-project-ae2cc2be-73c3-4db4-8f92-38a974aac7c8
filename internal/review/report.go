package review

import (
	"fmt"
	"ovencheck/internal/core"
	"ovencheck/internal/measurements"
	"ovencheck/internal/validation"
	"strings"
	"time"
)

type ReviewEvidence struct {
	BatchID          string                 `json:"batchId"`
	BatchVersion     int                    `json:"batchVersion"`
	Reviewer         string                 `json:"reviewer"`
	Decision         string                 `json:"decision"`
	Comment          string                 `json:"comment"`
	Checklist        []ChecklistItem        `json:"checklist"`
	FailedStages     []string               `json:"failedStages"`
	GeneratedAt      time.Time              `json:"generatedAt"`
	ReadingSummaries []measurements.Summary `json:"readingSummaries"`
	AuditLines       []AuditLine            `json:"auditLines"`
	AuditDigest      string                 `json:"auditDigest"`
}

func BuildEvidence(batch core.KilnBatch, reviewer, decision, comment string) ReviewEvidence {
	report := validation.Evaluate(batch)
	e := BuildEvidenceWithReport(batch, report)
	e.Reviewer, e.Decision, e.Comment = reviewer, decision, NormalizeComment(comment)
	return e
}

func BuildEvidenceWithReport(batch core.KilnBatch, report validation.BatchReport) ReviewEvidence {
	lines := AuditLines(batch)
	summaries := make([]measurements.Summary, 0, len(batch.Stages))
	for _, stage := range core.OrderedStages(batch) {
		summaries = append(summaries, measurements.Summarize(measurements.StageReadings(batch, stage.ID)))
	}
	return ReviewEvidence{BatchID: batch.ID, BatchVersion: batch.Version, Checklist: Checklist(batch, report), FailedStages: report.FailedStages(), GeneratedAt: time.Now(), ReadingSummaries: summaries, AuditLines: lines, AuditDigest: AuditDigest(lines)}
}

func EvidenceText(e ReviewEvidence) string {
	var b strings.Builder
	fmt.Fprintf(&b, "批次 %s（版本 %d）\n", e.BatchID, e.BatchVersion)
	fmt.Fprintf(&b, "审核人：%s\n决定：%s\n意见：%s\n", e.Reviewer, e.Decision, e.Comment)
	for _, item := range e.Checklist {
		state := "未通过"
		if item.Passed {
			state = "通过"
		}
		fmt.Fprintf(&b, "[%s] %s：%s\n", state, item.Label, item.Detail)
	}
	if len(e.FailedStages) > 0 {
		fmt.Fprintf(&b, "未通过阶段：%s\n", strings.Join(e.FailedStages, ", "))
	}
	return b.String()
}
