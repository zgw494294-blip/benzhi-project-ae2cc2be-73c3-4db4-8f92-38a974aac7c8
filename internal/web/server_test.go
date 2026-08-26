package web

import (
	"net/http/httptest"
	"ovencheck/internal/core"
	"testing"
)

func TestIndex(t *testing.T) {
	s, _ := core.NewStore(t.TempDir() + "/x.json")
	r := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	NewServer(s).Handler().ServeHTTP(r, req)
	if r.Code != 200 || len(r.Body.String()) < 100 {
		t.Fatal(r.Code)
	}
}
