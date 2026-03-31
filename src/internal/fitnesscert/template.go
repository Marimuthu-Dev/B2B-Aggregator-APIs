package fitnesscert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateData is passed to certificate_<ClientTypeID>.html (fields: Name, Company, Date).
type TemplateData struct {
	Name    string
	Company string
	Date    string
}

// RenderCertificateHTML loads and executes certificate_{clientTypeID}.html from templateDir.
func RenderCertificateHTML(templateDir string, clientTypeID int8, data TemplateData) (string, error) {
	name := fmt.Sprintf("certificate_%d.html", clientTypeID)
	path := filepath.Join(templateDir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("fitness cert template %s: %w", path, err)
	}
	t, err := template.New(name).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", path, err)
	}
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", path, err)
	}
	return buf.String(), nil
}
