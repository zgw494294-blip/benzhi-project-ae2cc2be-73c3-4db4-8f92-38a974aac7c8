package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func contentType(w http.ResponseWriter, value string) { w.Header().Set("Content-Type", value) }

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	contentType(w, "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeText(w http.ResponseWriter, status int, value string) {
	contentType(w, "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(value))
}

func pathID(path, prefix string) (string, error) {
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("路径前缀不匹配")
	}
	value := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if value == "" || strings.Contains(value, "/") {
		return "", fmt.Errorf("缺少资源标识")
	}
	return value, nil
}

func noCache(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }

func requestID(r *http.Request) string {
	if value := r.Header.Get("X-Request-ID"); value != "" {
		return value
	}
	return "local"
}
