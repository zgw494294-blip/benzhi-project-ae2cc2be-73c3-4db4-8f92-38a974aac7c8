package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type BatchIndex struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	KilnCode       string    `json:"kilnCode"`
	Status         string    `json:"status"`
	Version        int       `json:"version"`
	PlanVersion    int       `json:"planVersion"`
	StageCount     int       `json:"stageCount"`
	ReadingCount   int       `json:"readingCount"`
	ActionCount    int       `json:"actionCount"`
	LastActivityAt time.Time `json:"lastActivityAt"`
}

func IndexOf(batch KilnBatch) BatchIndex {
	last := batch.CreatedAt
	for _, reading := range batch.Readings {
		if reading.RecordedAt.After(last) {
			last = reading.RecordedAt
		}
	}
	for _, action := range batch.Actions {
		if action.PerformedAt.After(last) {
			last = action.PerformedAt
		}
	}
	if batch.FrozenAt != nil && batch.FrozenAt.After(last) {
		last = *batch.FrozenAt
	}
	if batch.ApprovedAt != nil && batch.ApprovedAt.After(last) {
		last = *batch.ApprovedAt
	}
	return BatchIndex{
		ID:             batch.ID,
		Name:           batch.Name,
		KilnCode:       batch.KilnCode,
		Status:         batch.Status,
		Version:        batch.Version,
		PlanVersion:    batch.PlanVersion,
		StageCount:     len(batch.Stages),
		ReadingCount:   len(batch.Readings),
		ActionCount:    len(batch.Actions),
		LastActivityAt: last,
	}
}

func Indexes(batches []KilnBatch) []BatchIndex {
	indexes := make([]BatchIndex, 0, len(batches))
	for _, batch := range batches {
		indexes = append(indexes, IndexOf(batch))
	}
	sort.Slice(indexes, func(i, j int) bool {
		if indexes[i].LastActivityAt.Equal(indexes[j].LastActivityAt) {
			return indexes[i].ID < indexes[j].ID
		}
		return indexes[i].LastActivityAt.After(indexes[j].LastActivityAt)
	})
	return indexes
}

func (s *Store) Search(keyword string, status string) []BatchIndex {
	items := make([]KilnBatch, 0)
	for _, batch := range s.List() {
		if status != "" && batch.Status != status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(batch.Name), strings.ToLower(keyword)) && !strings.Contains(strings.ToLower(batch.KilnCode), strings.ToLower(keyword)) && !strings.Contains(strings.ToLower(batch.ID), strings.ToLower(keyword)) {
			continue
		}
		items = append(items, batch)
	}
	return Indexes(items)
}

func (s *Store) SearchChecked(keyword, status string) ([]BatchIndex, error) {
	if !ValidStatus(status) {
		return nil, fmt.Errorf("未知状态 %s", status)
	}
	return s.Search(keyword, status), nil
}

func ValidStatus(status string) bool {
	if status == "" {
		return true
	}
	switch status {
	case StatusDraft, StatusCollect, StatusReview, StatusApproved, StatusRejected:
		return true
	}
	return false
}

func CanonicalStatus(status string) string {
	switch status {
	case "草稿":
		return StatusDraft
	case "采集中":
		return StatusCollect
	case "待审核":
		return StatusReview
	case "已批准":
		return StatusApproved
	case "已退回":
		return StatusRejected
	default:
		return status
	}
}

func StageProgress(batch KilnBatch) map[string]float64 {
	progress := make(map[string]float64, len(batch.Stages))
	for _, stage := range batch.Stages {
		count := 0
		for _, reading := range batch.Readings {
			if reading.StageID == stage.ID {
				count++
			}
		}
		if len(stage.SensorIDs) == 0 {
			progress[stage.ID] = 0
			continue
		}
		progress[stage.ID] = float64(count) / float64(len(stage.SensorIDs))
		if progress[stage.ID] > 1 {
			progress[stage.ID] = 1
		}
	}
	return progress
}
