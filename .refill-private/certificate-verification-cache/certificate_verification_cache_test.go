package certificateverificationcache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"ovencheck/internal/core"
	"ovencheck/internal/web"
	"testing"
	"time"
)

func TestCertificateVerificationCacheIsolatedPerCertificate(t *testing.T) {
	validSnapshot := `{"batch":"valid"}`
	validHash := sha256.Sum256([]byte(validSnapshot))
	snapshot := core.Snapshot{
		SchemaVersion: 1,
		SavedAt:       time.Unix(1, 0).UTC(),
		Batches: []core.KilnBatch{
			{
				ID:     "batch-valid",
				Status: core.StatusApproved,
				Certificate: &core.ReleaseCertificate{
					ID:       "certificate-valid",
					BatchID:  "batch-valid",
					Snapshot: validSnapshot,
					Digest:   hex.EncodeToString(validHash[:]),
				},
			},
			{
				ID:     "batch-corrupt",
				Status: core.StatusApproved,
				Certificate: &core.ReleaseCertificate{
					ID:       "certificate-corrupt",
					BatchID:  "batch-corrupt",
					Snapshot: `{"batch":"corrupt"}`,
					Digest:   "digest-does-not-match",
				},
			},
		},
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/snapshot.json"
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	handler := web.NewServer(store).Handler()

	verify := func(id string) bool {
		t.Helper()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/certificates/"+id+"/verify", nil)
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("verify %s returned status %d", id, recorder.Code)
		}
		var response struct {
			Valid bool `json:"valid"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Valid
	}

	if !verify("certificate-valid") {
		t.Fatal("valid certificate was rejected")
	}
	if verify("certificate-corrupt") {
		t.Fatal("corrupt certificate reused the previous certificate's verification result")
	}
}
