package review

import (
	"fmt"
	"ovencheck/internal/core"
	"ovencheck/internal/validation"
	"sort"
	"time"
)

type AuditLine struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Actor   string    `json:"actor"`
	StageID string    `json:"stageID,omitempty"`
	Text    string    `json:"text"`
}

func AuditLines(batch core.KilnBatch) []AuditLine {
	lines := []AuditLine{}
	lines = append(lines, AuditLine{At: batch.CreatedAt, Kind: "batch", Actor: batch.Engineer, Text: fmt.Sprintf("创建批次 %s", batch.Name)})
	if batch.FrozenAt != nil {
		lines = append(lines, AuditLine{At: *batch.FrozenAt, Kind: "freeze", Actor: batch.Engineer, Text: fmt.Sprintf("冻结计划版本 %d", batch.PlanVersion)})
	}
	for _, action := range batch.Actions {
		lines = append(lines, AuditLine{At: action.PerformedAt, Kind: "deviation", Actor: action.PerformedBy, StageID: action.StageID, Text: action.ActionText})
	}
	for _, reading := range batch.Readings {
		lines = append(lines, AuditLine{At: reading.RecordedAt, Kind: "reading", Actor: reading.Source, StageID: reading.StageID, Text: fmt.Sprintf("测点 %s 读数 %.2f°C", reading.SensorID, reading.Celsius)})
	}
	if batch.Review != nil {
		lines = append(lines, AuditLine{At: batch.Review.DecidedAt, Kind: "review", Actor: batch.Review.Reviewer, Text: batch.Review.ReviewComment})
	} else if batch.Certificate != nil {
		lines = append(lines, AuditLine{At: batch.Certificate.IssuedAt, Kind: "review", Actor: batch.Certificate.Reviewer, Text: batch.Certificate.ReviewComment})
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].At.Before(lines[j].At) })
	return lines
}

func AuditDigest(lines []AuditLine) string {
	text := ""
	for _, line := range lines {
		text += line.At.UTC().Format(time.RFC3339Nano) + "|" + line.Kind + "|" + line.Actor + "|" + line.Text + "\n"
	}
	return EvidenceDigest(core.KilnBatch{ID: text}, structReport(lines))
}

// structReport keeps the digest helper deterministic without exposing mutable state.
func structReport(lines []AuditLine) validation.BatchReport {
	return validation.BatchReport{BatchID: fmt.Sprintf("audit-%d", len(lines)), GeneratedAt: time.Unix(0, 0)}
}
