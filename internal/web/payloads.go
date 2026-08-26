package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code string, err error) {
	w.WriteHeader(status)
	writeJSON(w, APIError{Code: code, Message: "请求未完成", Detail: err.Error()})
}

func requireJSON(r *http.Request) error {
	content := r.Header.Get("Content-Type")
	if content == "" || !strings.HasPrefix(strings.ToLower(content), "application/json") {
		return fmt.Errorf("Content-Type 必须为 application/json")
	}
	return nil
}

func decodeWithLimit(r *http.Request, value any) error {
	if err := requireJSON(r); err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(nil, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}
