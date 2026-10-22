package fitnesscert

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// MergeCertificateFirst appends reportPDF after certPDF (certificate is page 1 of the result).
func MergeCertificateFirst(certPDF, reportPDF []byte) ([]byte, error) {
	return MergeCertificateFirstContext(context.Background(), certPDF, reportPDF)
}

// MergeCertificateFirstContext appends reportPDF after certPDF with context support.
// If either PDF is encrypted with an empty user password (standard for lab diagnostic reports),
// it automatically decrypts it in-memory before merging.
func MergeCertificateFirstContext(ctx context.Context, certPDF, reportPDF []byte) ([]byte, error) {
	if len(certPDF) == 0 || len(reportPDF) == 0 {
		return nil, fmt.Errorf("merge pdf: empty input")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	c1, err := decryptIfEncrypted(ctx, certPDF)
	if err != nil {
		return nil, fmt.Errorf("prepare certificate pdf: %w", err)
	}

	r2, err := decryptIfEncrypted(ctx, reportPDF)
	if err != nil {
		return nil, fmt.Errorf("prepare report pdf: %w", err)
	}

	var out bytes.Buffer
	conf := model.NewDefaultConfiguration()
	if err := api.MergeRaw(ctx, []io.ReadSeeker{bytes.NewReader(c1), bytes.NewReader(r2)}, &out, false, conf); err != nil {
		return nil, fmt.Errorf("pdf merge: %w", err)
	}
	return out.Bytes(), nil
}

func decryptIfEncrypted(ctx context.Context, pdfBytes []byte) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	var out bytes.Buffer
	err := api.Decrypt(ctx, bytes.NewReader(pdfBytes), &out, conf)
	if err == nil {
		return out.Bytes(), nil
	}
	if errors.Is(err, pdfcpu.ErrNotEncrypted) {
		return pdfBytes, nil
	}
	return nil, err
}
