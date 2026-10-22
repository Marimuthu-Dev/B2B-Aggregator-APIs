package fitnesscert

import (
	"bytes"
	"context"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var minimalPDF = []byte(`%PDF-1.4
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >> endobj
xref
0 4
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
trailer << /Size 4 /Root 1 0 R >>
startxref
190
%%EOF
`)

func TestMergeCertificateFirst(t *testing.T) {
	ctx := context.Background()
	merged, err := MergeCertificateFirstContext(ctx, minimalPDF, minimalPDF)
	if err != nil {
		t.Fatalf("unexpected error merging PDFs: %v", err)
	}
	if len(merged) == 0 {
		t.Fatal("expected non-empty merged PDF")
	}

	// Also verify the non-context wrapper
	merged2, err := MergeCertificateFirst(minimalPDF, minimalPDF)
	if err != nil {
		t.Fatalf("unexpected error in MergeCertificateFirst: %v", err)
	}
	if len(merged2) == 0 {
		t.Fatal("expected non-empty merged2 PDF")
	}
}

func TestMergeCertificateFirst_EmptyInput(t *testing.T) {
	_, err := MergeCertificateFirst(nil, minimalPDF)
	if err == nil {
		t.Fatal("expected error for nil certPDF")
	}

	_, err = MergeCertificateFirst(minimalPDF, nil)
	if err == nil {
		t.Fatal("expected error for nil reportPDF")
	}
}

func TestMergeCertificateFirst_AES128Encrypted(t *testing.T) {
	ctx := context.Background()
	// Encrypt minimalPDF with 128-bit AES and empty user password (typical diagnostic report security)
	encryptConf := model.NewAESConfiguration("", "ownerpass123", 128)
	var encBuf bytes.Buffer
	err := api.Encrypt(ctx, bytes.NewReader(minimalPDF), &encBuf, encryptConf)
	if err != nil {
		t.Fatalf("failed to create AES128 encrypted PDF: %v", err)
	}

	encryptedReportPDF := encBuf.Bytes()

	// MergeCertificateFirstContext should automatically decrypt the AES128 report and merge cleanly
	merged, err := MergeCertificateFirstContext(ctx, minimalPDF, encryptedReportPDF)
	if err != nil {
		t.Fatalf("unexpected error merging with AES128 encrypted PDF: %v", err)
	}
	if len(merged) == 0 {
		t.Fatal("expected non-empty merged PDF")
	}
}
