package trailingjsonwrite_test

import (
	"net/http"
	"net/http/httptest"
	"ovencheck/internal/core"
	appweb "ovencheck/internal/web"
	"strings"
	"testing"
)

func TestTrailingJSONIsRejectedBeforePersistentWrite(t *testing.T) {
	s, err := core.NewStore(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"Name":"合法批次","KilnCode":"K-6"}{"Name":"尾随对象","KilnCode":"K-7"}`
	req := httptest.NewRequest(http.MethodPost, "/api/batches", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	appweb.NewServer(s).Handler().ServeHTTP(rec, req)
	if rec.Code < 400 || len(s.List()) != 0 {
		t.Fatalf("trailing JSON was accepted: status=%d persisted=%d", rec.Code, len(s.List()))
	}
}
