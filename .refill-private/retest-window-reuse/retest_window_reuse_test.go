package retest_window_reuse_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"ovencheck/internal/core"
	"ovencheck/internal/review"
	"ovencheck/internal/web"
	"testing"
	"time"
)

func TestRetestUsesOnlyPostActionReadings(t *testing.T) {
	store, err := core.NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := store.Create("复测窗口批次", "K-RT-1", "高铝砖", "操作员", "工程师")
	if err != nil {
		t.Fatal(err)
	}
	stages := []core.HeatingStage{{
		ID: "stage-retest", Sequence: 1, TargetCelsius: 100,
		MaxRampCelsiusPerHour: 30, HoldMinutes: 10,
		ToleranceCelsius: 5, SensorIDs: []string{"S1"},
	}}
	if err := store.AddStages(batch.ID, batch.Version, stages); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	if err := store.Freeze(batch.ID, batch.Version); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	batch, _ = store.Get(batch.ID)
	prior := []core.TemperatureReading{
		{StageID: "stage-retest", SensorID: "S1", RecordedAt: now.Add(-20 * time.Minute), Celsius: 100},
		{StageID: "stage-retest", SensorID: "S1", RecordedAt: now.Add(-10 * time.Minute), Celsius: 100},
	}
	if err := store.AddReadings(batch.ID, batch.Version, prior); err != nil {
		t.Fatal(err)
	}
	batch, _ = store.Get(batch.ID)
	service := review.New(store)
	if err := service.RecordDeviationAt(
		batch.ID, batch.Version, "stage-retest", "equipment_check",
		"测点校准后需要独立复测", "重新校准测点", "操作员", true,
		now.Add(-5*time.Minute),
	); err != nil {
		t.Fatal(err)
	}

	batch, _ = store.Get(batch.ID)
	body, err := json.Marshal(map[string]any{
		"expectedVersion": batch.Version,
		"readings": []map[string]any{{
			"stageID": "stage-retest", "sensorID": "S1",
			"recordedAt": now.Add(-4 * time.Minute).Format(time.RFC3339),
			"celsius":    100, "unit": "C",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/batches/%s/readings", batch.ID), bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	web.NewServer(store).Handler().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("追加复测读数失败: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	after, _ := store.Get(batch.ID)
	if _, pending := after.PendingRetests["stage-retest"]; !pending {
		t.Fatalf("处置后只有一条读数，尚未满足 10 分钟保温要求，但待复测状态已被清除，status=%s", after.Status)
	}
}
