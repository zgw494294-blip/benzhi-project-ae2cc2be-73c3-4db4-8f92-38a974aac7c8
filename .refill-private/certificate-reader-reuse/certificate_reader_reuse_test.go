package certificate_reader_reuse

import (
	"net/http/httptest"
	"ovencheck/internal/core"
	"ovencheck/internal/review"
	"ovencheck/internal/web"
	"testing"
	"time"
)

func TestRepeatedCertificateDownloadReturnsCompleteBody(t *testing.T) {
	store, err := core.NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := store.Create("重复下载验证", "K-9", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	stage := core.HeatingStage{ID: "stage-release", Sequence: 1, TargetCelsius: 100, MaxRampCelsiusPerHour: 200, SensorIDs: []string{"sensor-a"}}
	if err := store.AddStages(batch.ID, batch.Version, []core.HeatingStage{stage}); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	if err := store.Freeze(batch.ID, batch.Version); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	reading := core.TemperatureReading{StageID: stage.ID, SensorID: "sensor-a", RecordedAt: time.Now().Add(-time.Minute), Celsius: 100}
	if err := store.AddReadings(batch.ID, batch.Version, []core.TemperatureReading{reading}); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	certificate, err := review.New(store).Decide(batch.ID, batch.Version, "审核员", core.StatusApproved, "证据完整，同意放行")
	if err != nil {
		t.Fatal(err)
	}

	handler := web.NewServer(store).Handler()
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest("GET", "/api/certificates/"+certificate.ID, nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest("GET", "/api/certificates/"+certificate.ID, nil))

	if first.Code != 200 || second.Code != 200 {
		t.Fatalf("下载状态异常: first=%d second=%d", first.Code, second.Code)
	}
	if second.Body.String() != first.Body.String() {
		t.Fatalf("重复下载返回不完整内容: first=%q second=%q", first.Body.String(), second.Body.String())
	}
}
