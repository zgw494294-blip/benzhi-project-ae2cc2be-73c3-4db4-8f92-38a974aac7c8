package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"ovencheck/internal/core"
	"ovencheck/internal/validation"
)

func EvidenceDigest(batch core.KilnBatch, report validation.BatchReport) string {
	b, _ := json.Marshal(struct {
		B core.KilnBatch
		R validation.BatchReport
	}{batch, report})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
