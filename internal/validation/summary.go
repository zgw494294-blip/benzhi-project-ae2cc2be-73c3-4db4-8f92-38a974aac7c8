package validation

import (
	"ovencheck/internal/core"
	"sort"
)

type ComplianceSummary struct {
	Total    int      `json:"total"`
	Passed   int      `json:"passed"`
	Failed   int      `json:"failed"`
	Missing  int      `json:"missing"`
	Messages []string `json:"messages"`
}

func Summarize(report BatchReport) ComplianceSummary {
	out := ComplianceSummary{Messages: []string{}}
	out.Total = len(report.Stages)
	for _, stage := range report.Stages {
		if stage.Compliant {
			out.Passed++
		} else {
			out.Failed++
			for _, f := range stage.Findings {
				if f.Code == "MISSING" {
					out.Missing++
				}
				out.Messages = append(out.Messages, f.Message)
			}
		}
	}
	sort.Strings(out.Messages)
	return out
}
func ReadyForApproval(batch core.KilnBatch, report BatchReport) bool {
	return batch.Status != core.StatusApproved && batch.FrozenAt != nil && len(batch.Readings) > 0 && report.Compliant
}
