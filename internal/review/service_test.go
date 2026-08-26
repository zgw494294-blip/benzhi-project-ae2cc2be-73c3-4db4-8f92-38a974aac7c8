package review

import (
	"ovencheck/internal/core"
	"testing"
)

func TestCertificateText(t *testing.T) {
	c := core.ReleaseCertificate{BatchID: "b", Decision: "approved"}
	if CertificateText(c) == "" {
		t.Fatal("empty")
	}
}
