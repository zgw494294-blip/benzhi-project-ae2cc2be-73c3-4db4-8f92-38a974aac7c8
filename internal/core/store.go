package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu        sync.RWMutex
	path      string
	auditPath string
	batches   map[string]*KilnBatch
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, auditPath: path + ".audit.jsonl", batches: map[string]*KilnBatch{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var snap Snapshot
	if err = json.Unmarshal(b, &snap); err != nil {
		return fmt.Errorf("快照损坏: %w", err)
	}
	if snap.SchemaVersion != 1 {
		return fmt.Errorf("不支持的快照版本 %d", snap.SchemaVersion)
	}
	for i := range snap.Batches {
		x := snap.Batches[i]
		s.batches[x.ID] = &x
	}
	return nil
}
func (s *Store) saveLocked() error {
	snap := Snapshot{SchemaVersion: 1, SavedAt: time.Now(), Batches: make([]KilnBatch, 0, len(s.batches))}
	for _, b := range s.batches {
		snap.Batches = append(snap.Batches, *b)
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".snapshot-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
func (s *Store) List() []KilnBatch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]KilnBatch, 0, len(s.batches))
	for _, b := range s.batches {
		out = append(out, *b)
	}
	return out
}
func (s *Store) Get(id string) (KilnBatch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.batches[id]
	if !ok {
		return KilnBatch{}, false
	}
	return *b, true
}
func (s *Store) mutate(id string, expected int, fn func(*KilnBatch) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.batches[id]
	if !ok {
		return errors.New("批次不存在")
	}
	if expected <= 0 {
		return errors.New("必须提供 expectedVersion")
	}
	if b.Version != expected {
		return fmt.Errorf("版本冲突: 当前版本 %d", b.Version)
	}
	if err := fn(b); err != nil {
		return err
	}
	b.Version++
	_ = appendAudit(s.auditPath, AuditEvent{Action: "mutate_batch", BatchID: id, Version: b.Version})
	return s.saveLocked()
}
func (s *Store) Create(name, kiln, lining, operator, engineer string) (KilnBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	b := KilnBatch{ID: NewID("batch"), Name: name, KilnCode: kiln, LiningMaterial: lining, Operator: operator, Engineer: engineer, Status: StatusDraft, Version: 1, CreatedAt: now, Stages: []HeatingStage{}, Readings: []TemperatureReading{}, Actions: []DeviationAction{}, PendingRetests: map[string]RetestState{}}
	s.batches[b.ID] = &b
	_ = appendAudit(s.auditPath, AuditEvent{Action: "create_batch", BatchID: b.ID, Version: b.Version})
	return b, s.saveLocked()
}
func (s *Store) AddStages(id string, expected int, stages []HeatingStage) error {
	return s.mutate(id, expected, func(b *KilnBatch) error {
		if err := CanEditPlan(*b); err != nil {
			return err
		}
		if err := ValidateStages(stages); err != nil {
			return err
		}
		for i := range stages {
			stages[i].BatchID = id
			if stages[i].ID == "" {
				stages[i].ID = NewID("stage")
			}
			stages[i].Frozen = false
		}
		sort.Slice(stages, func(i, j int) bool { return stages[i].Sequence < stages[j].Sequence })
		b.Stages = stages
		return nil
	})
}
func (s *Store) Freeze(id string, expected int) error {
	return s.mutate(id, expected, func(b *KilnBatch) error {
		if len(b.Stages) == 0 {
			return errors.New("至少配置一个阶段")
		}
		if b.Status != StatusDraft {
			return errors.New("批次已冻结")
		}
		now := time.Now()
		b.FrozenAt = &now
		b.PlanVersion++
		b.Status = StatusCollect
		for i := range b.Stages {
			b.Stages[i].Frozen = true
		}
		return nil
	})
}
func (s *Store) AddReadings(id string, expected int, rs []TemperatureReading) error {
	return s.mutate(id, expected, func(b *KilnBatch) error {
		if err := CanCollect(*b); err != nil {
			return err
		}
		if len(rs) == 0 {
			return errors.New("至少提交一条读数")
		}
		for _, reading := range rs {
			if reading.RecordedAt.After(time.Now().Add(time.Second)) {
				return errors.New("读数时间不能晚于当前时间")
			}
		}
		if b.PendingRetests == nil {
			b.PendingRetests = map[string]RetestState{}
		}
		for i := range rs {
			if pending, ok := b.PendingRetests[rs[i].StageID]; ok {
				if rs[i].RecordedAt.Before(pending.RequiredAfter) || rs[i].RecordedAt.Equal(pending.RequiredAfter) {
					return fmt.Errorf("阶段 %s 复测读数必须晚于处置时间", rs[i].StageID)
				}
			} else if len(b.PendingRetests) > 0 {
				return fmt.Errorf("当前批次只允许追加待复测阶段 %s 的读数", pendingStage(b.PendingRetests))
			}
			if rs[i].ID == "" {
				rs[i].ID = NewID("reading")
			}
			rs[i].BatchID = id
			if rs[i].Source == "" {
				rs[i].Source = "manual"
			}
			rs[i].Sequence = len(b.Readings) + i + 1
		}
		b.Readings = append(b.Readings, rs...)
		return nil
	})
}

func pendingStage(states map[string]RetestState) string {
	for stage := range states {
		return stage
	}
	return ""
}
func (s *Store) AddAction(id string, expected int, a DeviationAction) error {
	return s.mutate(id, expected, func(b *KilnBatch) error {
		if b.Status == StatusApproved {
			return errors.New("已批准批次不可修改")
		}
		if a.StageID == "" || a.Kind == "" || a.Reason == "" || a.ActionText == "" || a.PerformedBy == "" {
			return errors.New("阶段、措施类型、原因、措施和责任人不能为空")
		}
		a.ID = NewID("action")
		a.BatchID = id
		if a.PerformedAt.IsZero() {
			a.PerformedAt = time.Now()
		}
		b.Actions = append(b.Actions, a)
		if a.RetestRequired {
			if b.PendingRetests == nil {
				b.PendingRetests = map[string]RetestState{}
			}
			b.PendingRetests[a.StageID] = RetestState{StageID: a.StageID, ActionID: a.ID, RequiredAfter: a.PerformedAt, Required: true}
		}
		b.Status = StatusCollect
		return nil
	})
}

func (s *Store) PendingRetests(id string) map[string]RetestState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.batches[id]
	if !ok {
		return nil
	}
	out := map[string]RetestState{}
	for k, v := range b.PendingRetests {
		out[k] = v
	}
	return out
}

func (s *Store) ClearRetest(id string, expected int, stageID string) error {
	return s.mutate(id, expected, func(b *KilnBatch) error {
		if _, ok := b.PendingRetests[stageID]; !ok {
			return fmt.Errorf("阶段 %s 没有待复测整改", stageID)
		}
		delete(b.PendingRetests, stageID)
		if len(b.PendingRetests) == 0 {
			b.Status = StatusReview
		}
		return nil
	})
}
func (s *Store) SetReview(id string, expected int, status, reviewer, comment string, cert *ReleaseCertificate) error {
	return s.mutate(id, expected, func(b *KilnBatch) error {
		if b.Status == StatusApproved {
			return errors.New("合格证已签发")
		}
		if status != StatusApproved && status != StatusRejected {
			return errors.New("无效审核决定")
		}
		b.Status = status
		now := time.Now()
		if status == StatusApproved {
			b.ApprovedAt = &now
			b.Certificate = cert
		} else {
			b.Certificate = nil
		}
		b.Review = &ReviewRecord{Decision: status, Reviewer: reviewer, ReviewComment: comment, DecidedAt: now}
		return nil
	})
}
