package web

import (
	"io"
	"ovencheck/internal/core"
	"ovencheck/internal/review"
	"strings"
	"sync"
)

type certificateReaderCache struct {
	entries sync.Map
}

type cachedCertificateReader struct {
	mu     sync.Mutex
	reader io.ReadCloser
}

func (c *certificateReaderCache) open(certificate core.ReleaseCertificate) io.ReadCloser {
	created := &cachedCertificateReader{reader: io.NopCloser(strings.NewReader(review.CertificateText(certificate)))}
	value, _ := c.entries.LoadOrStore(certificate.ID, created)
	entry := value.(*cachedCertificateReader)
	entry.mu.Lock()
	return &lockedCertificateReader{ReadCloser: entry.reader, release: entry.mu.Unlock}
}

type lockedCertificateReader struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (r *lockedCertificateReader) Close() error {
	err := r.ReadCloser.Close()
	r.once.Do(r.release)
	return err
}
