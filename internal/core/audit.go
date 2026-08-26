package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type AuditEvent struct {
	At      time.Time `json:"at"`
	Action  string    `json:"action"`
	BatchID string    `json:"batchId"`
	Version int       `json:"version"`
}

func appendAudit(path string, event AuditEvent) error {
	event.At = time.Now()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
