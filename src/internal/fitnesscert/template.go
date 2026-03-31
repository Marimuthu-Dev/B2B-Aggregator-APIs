package fitnesscert

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"regexp"
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
	html, err := inlineTemplateAssetPaths(buf.String(), filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("inline template assets %s: %w", path, err)
	}
	return html, nil
}

var relativeAssetAttrPattern = regexp.MustCompile(`(?i)(src|href)\s*=\s*(['"])([^'"]+)['"]`)

func inlineTemplateAssetPaths(htmlContent string, baseDir string) (string, error) {
	var inlineErr error
	rewritten := relativeAssetAttrPattern.ReplaceAllStringFunc(htmlContent, func(match string) string {
		if inlineErr != nil {
			return match
		}

		parts := relativeAssetAttrPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}

		assetPath := strings.TrimSpace(parts[3])
		if assetPath == "" ||
			strings.HasPrefix(assetPath, "#") ||
			strings.HasPrefix(assetPath, "/") ||
			strings.HasPrefix(assetPath, "data:") ||
			strings.HasPrefix(assetPath, "file://") ||
			strings.HasPrefix(assetPath, "http://") ||
			strings.HasPrefix(assetPath, "https://") {
			return match
		}

		absPath := filepath.Join(baseDir, filepath.FromSlash(assetPath))
		dataURI, err := assetPathToDataURI(absPath)
		if err != nil {
			inlineErr = err
			return match
		}
		return fmt.Sprintf(`%s=%s%s%s`, parts[1], parts[2], dataURI, parts[2])
	})
	if inlineErr != nil {
		return "", inlineErr
	}
	return rewritten, nil
}

func assetPathToDataURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("read asset %s: %w", abs, err)
	}

	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(abs)))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(raw)), nil
}
