package fitnesscert

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// MergeCertificateFirst appends reportPDF after certPDF (certificate is page 1 of the result).
func MergeCertificateFirst(certPDF, reportPDF []byte) ([]byte, error) {
	if len(certPDF) == 0 || len(reportPDF) == 0 {
		return nil, fmt.Errorf("merge pdf: empty input")
	}
	r1 := bytes.NewReader(certPDF)
	r2 := bytes.NewReader(reportPDF)
	var out bytes.Buffer
	conf := model.NewDefaultConfiguration()
	if err := api.MergeRaw([]io.ReadSeeker{r1, r2}, &out, false, conf); err != nil {
		return nil, fmt.Errorf("pdf merge: %w", err)
	}
	return out.Bytes(), nil
}
