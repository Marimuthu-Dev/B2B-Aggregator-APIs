package emailtemplates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
	"unicode"
)

// Template layout (same event, one HTML file per DB_SCHEMA):
//
//	templates/{DB_SCHEMA}/{event}.html
//	templates/default/{event}.html          // fallback when schema folder is missing
//
// Example: DB_SCHEMA=MediAdmin → templates/MediAdmin/client_created.html
//
//go:embed templates/*/*.html
var files embed.FS

const (
	EventClientCreated   = "client_created"
	EventLabCreated      = "lab_created"
	EventEmployeeCreated = "employee_created"
	EventStoreCreated    = "store_created"
	EventForgotPassword  = "forgot_password"
	defaultTemplateDir   = "default"
	templatesRoot        = "templates"
)

// ClientCreatedData is passed to {schema}/client_created.html.
type ClientCreatedData struct {
	ClientName          string
	Username            string
	GeneratePasswordURL string
	PortalURL           string
	LogoURL             string
	SupportPhone        string
	SupportEmail        string
	Year                int
}

// LabCreatedData is passed to {schema}/lab_created.html.
type LabCreatedData struct {
	LabName             string
	Username            string
	GeneratePasswordURL string
	PortalURL           string
	LogoURL             string
	SupportPhone        string
	SupportEmail        string
	Year                int
}

// RenderClientCreated executes the client-created template for the given SQL schema (DB_SCHEMA).
func RenderClientCreated(schema string, data ClientCreatedData) (string, error) {
	return RenderEvent(schema, EventClientCreated, data)
}

// RenderLabCreated executes the lab-created template for the given SQL schema (DB_SCHEMA).
func RenderLabCreated(schema string, data LabCreatedData) (string, error) {
	return RenderEvent(schema, EventLabCreated, data)
}

// EmployeeCreatedData is passed to {schema}/employee_created.html.
type EmployeeCreatedData struct {
	FullName            string
	Username            string
	GeneratePasswordURL string
	PortalURL           string
	LogoURL             string
	SupportPhone        string
	SupportEmail        string
	Year                int
}

// RenderEmployeeCreated executes the employee-created template for the given SQL schema (DB_SCHEMA).
func RenderEmployeeCreated(schema string, data EmployeeCreatedData) (string, error) {
	return RenderEvent(schema, EventEmployeeCreated, data)
}

// StoreCreatedData is passed to {schema}/store_created.html (MedLyfe tbl_StoreMaster).
type StoreCreatedData struct {
	StoreName           string
	Username            string
	GeneratePasswordURL string
	PortalURL           string
	LogoURL             string
	SupportPhone        string
	SupportEmail        string
	Year                int
}

// RenderStoreCreated executes the store-created template for the given SQL schema (DB_SCHEMA).
func RenderStoreCreated(schema string, data StoreCreatedData) (string, error) {
	return RenderEvent(schema, EventStoreCreated, data)
}

// ForgotPasswordData is passed to {schema}/forgot_password.html.
type ForgotPasswordData struct {
	DisplayName         string
	Username            string
	GeneratePasswordURL string
	PortalURL           string
	LogoURL             string
	SupportPhone        string
	SupportEmail        string
	Year                int
	LinkExpiry          string
}

// RenderForgotPassword executes the forgot-password template for the given SQL schema (DB_SCHEMA).
func RenderForgotPassword(schema string, data ForgotPasswordData) (string, error) {
	return RenderEvent(schema, EventForgotPassword, data)
}

// RenderEvent executes templates/{schema}/{event}.html, then templates/default/{event}.html.
func RenderEvent(schema, event string, data any) (string, error) {
	path, err := ResolveTemplatePath(schema, event)
	if err != nil {
		return "", err
	}
	return render(path, data)
}

// ResolveTemplatePath returns the embedded HTML path for schema + event.
func ResolveTemplatePath(schema, event string) (string, error) {
	return resolveTemplatePath(schema, event)
}

func resolveTemplatePath(schema, event string) (string, error) {
	event = sanitizeIdent(event)
	if event == "" {
		return "", fmt.Errorf("email event name is empty")
	}

	var candidates []string
	if dir := matchSchemaDir(schema); dir != "" {
		candidates = append(candidates, templatesRoot+"/"+dir+"/"+event+".html")
	}
	candidates = append(candidates, templatesRoot+"/"+defaultTemplateDir+"/"+event+".html")

	for _, path := range candidates {
		if _, err := files.ReadFile(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("email template not found for schema=%q event=%q", strings.TrimSpace(schema), event)
}

func matchSchemaDir(schema string) string {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return ""
	}
	entries, err := fs.ReadDir(files, templatesRoot)
	if err != nil {
		return sanitizeIdent(schema)
	}
	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(e.Name(), schema) {
			return e.Name()
		}
	}
	return sanitizeIdent(schema)
}

func sanitizeIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func render(name string, data any) (string, error) {
	raw, err := files.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("email template %s: %w", name, err)
	}
	t, err := template.New(name).Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse email template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute email template %s: %w", name, err)
	}
	return buf.String(), nil
}
