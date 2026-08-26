package review

import (
	"errors"
	"ovencheck/internal/core"
	"ovencheck/internal/validation"
	"strings"
)

type DecisionRequest struct {
	Reviewer        string `json:"reviewer"`
	Decision        string `json:"decision"`
	Comment         string `json:"comment"`
	ExpectedVersion int    `json:"expectedVersion"`
}

func ValidateDecision(req DecisionRequest, batch core.KilnBatch, report validation.BatchReport) error {
	if strings.TrimSpace(req.Reviewer) == "" {
		return errors.New("审核人不能为空")
	}
	if strings.TrimSpace(req.Comment) == "" {
		return errors.New("审核意见不能为空")
	}
	if req.ExpectedVersion <= 0 {
		return errors.New("expectedVersion 必须为正数")
	}
	if req.Decision == core.StatusApproved && !report.Compliant {
		return errors.New("不合规批次不能批准")
	}
	if req.Decision == core.StatusApproved {
		if err := core.CanReview(batch); err != nil {
			return err
		}
		if !validation.ReadyForApproval(batch, report) {
			return errors.New("证据不完整，不能批准")
		}
		if len(batch.PendingRetests) > 0 {
			return errors.New("存在未完成复测")
		}
	}
	if req.Decision != core.StatusApproved && req.Decision != core.StatusRejected {
		return errors.New("决定必须为批准或退回")
	}
	return nil
}
