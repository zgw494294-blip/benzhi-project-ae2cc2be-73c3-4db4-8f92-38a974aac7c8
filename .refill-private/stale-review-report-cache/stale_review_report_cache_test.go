package stale_review_report_cache_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"ovencheck/internal/core"
	"ovencheck/internal/web"
	"testing"
	"time"
)

func TestApprovalRejectsStaleCachedEvidenceAfterReadingsChange(t *testing.T) {
	store, err := core.NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := store.Create("缓存版本核验", "K-27", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	stage := core.HeatingStage{
		Sequence:              1,
		TargetCelsius:         100,
		MaxRampCelsiusPerHour: 120,
		HoldMinutes:           20,
		ToleranceCelsius:      3,
		SensorIDs:             []string{"S1"},
	}
	if err := store.AddStages(batch.ID, batch.Version, []core.HeatingStage{stage}); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	if err := store.Freeze(batch.ID, batch.Version); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	stageID := batch.Stages[0].ID
	server := web.NewServer(store).Handler()

	evidence := httptest.NewRecorder()
	evidenceRequest := httptest.NewRequest(http.MethodGet, "/api/batches/"+batch.ID+"/evidence", nil)
	server.ServeHTTP(evidence, evidenceRequest)
	if evidence.Code != http.StatusOK {
		t.Fatalf("预览审核证据失败: status=%d body=%s", evidence.Code, evidence.Body.String())
	}

	start := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	readingsBody := fmt.Sprintf(`{"expectedVersion":%d,"readings":[{"stageID":%q,"sensorID":"S1","recordedAt":%q,"celsius":100,"unit":"C"},{"stageID":%q,"sensorID":"S1","recordedAt":%q,"celsius":100,"unit":"C"}]}`,
		batch.Version, stageID, start.Format(time.RFC3339), stageID, start.Add(30*time.Minute).Format(time.RFC3339))
	readings := httptest.NewRecorder()
	readingsRequest := httptest.NewRequest(http.MethodPost, "/api/batches/"+batch.ID+"/readings", bytes.NewBufferString(readingsBody))
	server.ServeHTTP(readings, readingsRequest)
	if readings.Code != http.StatusOK {
		t.Fatalf("追加合规读数失败: status=%d body=%s", readings.Code, readings.Body.String())
	}

	batch, _ = store.Get(batch.ID)
	reviewBody := fmt.Sprintf(`{"expectedVersion":%d,"reviewer":"质量审核员","decision":"approved","comment":"证据完整，同意放行"}`, batch.Version)
	review := httptest.NewRecorder()
	reviewRequest := httptest.NewRequest(http.MethodPost, "/api/batches/"+batch.ID+"/review", bytes.NewBufferString(reviewBody))
	server.ServeHTTP(review, reviewRequest)
	if review.Code != http.StatusOK {
		t.Fatalf("批次版本更新后应按最新读数重新核验并批准: status=%d body=%s", review.Code, review.Body.String())
	}

	approved, ok := store.Get(batch.ID)
	if !ok || approved.Status != core.StatusApproved || approved.Certificate == nil {
		t.Fatalf("批准成功后应持久化合格证: %#v", approved)
	}
}
