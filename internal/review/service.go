package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"ovencheck/internal/core"
	"ovencheck/internal/validation"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Service struct {
	Store *core.Store

	reportMu    sync.RWMutex
	reportCache map[string]validation.BatchReport
}

func New(s *core.Store) *Service {
	return &Service{Store: s, reportCache: map[string]validation.BatchReport{}}
}

// reportFor returns the validation report for the given batch, keyed by batch
// ID and version so that any version bump (new readings, deviations, retest
// clearance, review) invalidates a previously cached report. This keeps the
// report aligned with the current batch state when evidence is inspected
// before the batch is finalized and later approvals are submitted.
func (s *Service) reportFor(batch core.KilnBatch) validation.BatchReport {
	key := reportCacheKey(batch.ID, batch.Version)
	s.reportMu.RLock()
	report, ok := s.reportCache[key]
	s.reportMu.RUnlock()
	if ok {
		return report
	}
	report = validation.Evaluate(batch)
	s.reportMu.Lock()
	s.reportCache[key] = report
	s.reportMu.Unlock()
	return report
}

func reportCacheKey(batchID string, version int) string {
	return batchID + ":" + strconv.Itoa(version)
}
func (s *Service) RecordDeviation(id string, expected int, stageID, kind, reason, action, by string, retest bool) error {
	return s.RecordDeviationAt(id, expected, stageID, kind, reason, action, by, retest, time.Time{})
}

func (s *Service) RecordDeviationAt(id string, expected int, stageID, kind, reason, action, by string, retest bool, performedAt time.Time) error {
	if stageID == "" || kind == "" || reason == "" || action == "" || by == "" {
		return errors.New("阶段、措施类型、原因、措施和责任人不能为空")
	}
	b, ok := s.Store.Get(id)
	if !ok {
		return errors.New("批次不存在")
	}
	found := false
	for _, st := range b.Stages {
		if st.ID == stageID {
			found = true
		}
	}
	if !found {
		return errors.New("阶段不存在")
	}
	return s.Store.AddAction(id, expected, core.DeviationAction{StageID: stageID, Kind: kind, Reason: reason, ActionText: action, PerformedBy: by, RetestRequired: retest, PerformedAt: performedAt})
}
func (s *Service) Decide(id string, expected int, reviewer, decision, comment string) (core.ReleaseCertificate, error) {
	comment = NormalizeComment(comment)
	if reviewer == "" || comment == "" {
		return core.ReleaseCertificate{}, errors.New("审核人和意见不能为空")
	}
	b, ok := s.Store.Get(id)
	if !ok {
		return core.ReleaseCertificate{}, errors.New("批次不存在")
	}
	report := s.reportFor(b)
	if err := ValidateDecision(DecisionRequest{Reviewer: reviewer, Decision: decision, Comment: comment, ExpectedVersion: expected}, b, report); err != nil {
		return core.ReleaseCertificate{}, err
	}
	if decision == core.StatusApproved {
		if !validation.ReadyForApproval(b, report) {
			return core.ReleaseCertificate{}, errors.New("证据不完整，不能批准")
		}
		if pending := s.Store.PendingRetests(id); len(pending) > 0 {
			return core.ReleaseCertificate{}, fmt.Errorf("存在待复测阶段: %s", pendingStageIDs(pending))
		}
	}
	snap, _ := json.Marshal(struct {
		Batch  core.KilnBatch
		Report validation.BatchReport
	}{b, report})
	sum := sha256.Sum256(snap)
	cert := core.ReleaseCertificate{ID: core.NewID("cert"), BatchID: id, BatchVersion: b.Version, Decision: decision, Reviewer: reviewer, ReviewComment: comment, Digest: hex.EncodeToString(sum[:]), IssuedAt: time.Now(), Snapshot: string(snap)}
	var certPtr *core.ReleaseCertificate
	if decision == core.StatusApproved {
		certPtr = &cert
	}
	if err := s.Store.SetReview(id, expected, decision, reviewer, comment, certPtr); err != nil {
		return core.ReleaseCertificate{}, err
	}
	if decision != core.StatusApproved {
		return core.ReleaseCertificate{BatchID: id, BatchVersion: b.Version, Decision: decision, Reviewer: reviewer, ReviewComment: comment}, nil
	}
	return cert, nil
}

func pendingStageIDs(p map[string]core.RetestState) string {
	ids := make([]string, 0, len(p))
	for id := range p {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func (s *Service) Evidence(batch core.KilnBatch) ReviewEvidence {
	report := s.reportFor(batch)
	return BuildEvidenceWithReport(batch, report)
}

func (s *Service) RefreshRetests(id string) error {
	b, ok := s.Store.Get(id)
	if !ok {
		return errors.New("批次不存在")
	}
	if len(b.PendingRetests) == 0 {
		return nil
	}
	report := validation.Evaluate(b)
	for stageID := range b.PendingRetests {
		pending := b.PendingRetests[stageID]
		trial := b
		trial.Readings = make([]core.TemperatureReading, 0, len(b.Readings))
		for _, reading := range b.Readings {
			if reading.StageID != stageID || reading.RecordedAt.After(pending.RequiredAfter) {
				trial.Readings = append(trial.Readings, reading)
			}
		}
		trialReport := validation.Evaluate(trial)
		for _, stage := range report.Stages {
			if stage.StageID == stageID {
				for _, candidate := range trialReport.Stages {
					if candidate.StageID == stageID {
						stage = candidate
						break
					}
				}
			}
			if stage.StageID == stageID && stage.Compliant {
				current, ok := s.Store.Get(id)
				if !ok {
					return errors.New("批次不存在")
				}
				if err := s.Store.ClearRetest(id, current.Version, stageID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func VerifyCertificate(c core.ReleaseCertificate) bool {
	if c.Digest == "" || c.Snapshot == "" {
		return false
	}
	h := sha256.Sum256([]byte(c.Snapshot))
	return hex.EncodeToString(h[:]) == c.Digest
}
func CertificateText(c core.ReleaseCertificate) string {
	return fmt.Sprintf("工业窑炉烘炉合格证\n批次: %s\n批次版本: %d\n决定: %s\n审核人: %s\n审核意见: %s\n摘要校验值: %s\n签发时间: %s\n", c.BatchID, c.BatchVersion, c.Decision, c.Reviewer, c.ReviewComment, c.Digest, c.IssuedAt.Format(time.RFC3339))
}
