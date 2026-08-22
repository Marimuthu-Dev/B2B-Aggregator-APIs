package emailtemplates

import (
	"strings"
	"testing"
)

func sampleData() ClientCreatedData {
	return ClientCreatedData{
		ClientName:          "Acme Industries",
		Username:            "9036302806",
		GeneratePasswordURL: "https://client.urmediconnect.com/login",
		PortalURL:           "https://client.urmediconnect.com/",
		LogoURL:             "https://urmediconnect.com/img/logo.jpeg",
		SupportPhone:        "+91 9036302806",
		SupportEmail:        "support@urmediconnect.com",
		Year:                2026,
	}
}

func TestRenderClientCreated_MediAdmin(t *testing.T) {
	html, err := RenderClientCreated("MediAdmin", sampleData())
	if err != nil {
		t.Fatalf("RenderClientCreated MediAdmin: %v", err)
	}
	for _, want := range []string{
		"UrMediconnect",
		"Acme Industries",
		"9036302806",
		"https://client.urmediconnect.com/",
		"https://urmediconnect.com/img/logo.jpeg",
		"7411558079",
		"support@urmediconnect.com",
		"&copy; 2026",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("MediAdmin HTML missing %q", want)
		}
	}
	if strings.Contains(html, "Team MedLyfe") {
		t.Error("MediAdmin template should not use MedLyfe branding")
	}
}

func TestRenderClientCreated_MediAdmin_resetPasswordHref(t *testing.T) {
	data := sampleData()
	data.GeneratePasswordURL = "https://client.urmediconnect.com/reset-password?token=abc%2Fdef%2Bghi%3D"
	html, err := RenderClientCreated("MediAdmin", data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "%252F") || strings.Contains(html, "%252B") {
		t.Fatal("href double-encoded the reset token")
	}
	if !strings.Contains(html, "reset-password?token=abc%2Fdef%2Bghi%3D") {
		t.Fatalf("href missing encoded token URL")
	}
}

func TestRenderClientCreated_MedLyfe(t *testing.T) {
	html, err := RenderClientCreated("MedLyfe", sampleData())
	if err != nil {
		t.Fatalf("RenderClientCreated MedLyfe: %v", err)
	}
	for _, want := range []string{
		"MedLyfe",
		"Dear Vendor Partner",
		"Acme Industries",
		"9036302806",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("MedLyfe HTML missing %q", want)
		}
	}
	if strings.Contains(html, "UrMediconnect") {
		t.Error("MedLyfe template should not use UrMediconnect branding")
	}
}

func TestResolveTemplatePath_schemaFolderAndFallback(t *testing.T) {
	path, err := ResolveTemplatePath("mediadmin", EventClientCreated)
	if err != nil {
		t.Fatalf("case-insensitive MediAdmin: %v", err)
	}
	if path != "templates/MediAdmin/client_created.html" {
		t.Errorf("got %q", path)
	}

	path, err = ResolveTemplatePath("UnknownSchema", EventClientCreated)
	if err != nil {
		t.Fatalf("fallback default: %v", err)
	}
	if path != "templates/default/client_created.html" {
		t.Errorf("fallback got %q", path)
	}

	path, err = ResolveTemplatePath("MediAdmin", EventLabCreated)
	if err != nil {
		t.Fatalf("lab MediAdmin: %v", err)
	}
	if path != "templates/MediAdmin/lab_created.html" {
		t.Errorf("lab path got %q", path)
	}

	path, err = ResolveTemplatePath("MediAdmin", EventEmployeeCreated)
	if err != nil {
		t.Fatalf("employee MediAdmin: %v", err)
	}
	if path != "templates/MediAdmin/employee_created.html" {
		t.Errorf("employee path got %q", path)
	}
}

func TestRenderEmployeeCreated_MediAdmin(t *testing.T) {
	html, err := RenderEmployeeCreated("MediAdmin", EmployeeCreatedData{
		FullName:            "Priya Sharma",
		Username:            "9876543210",
		GeneratePasswordURL: "https://ops.urmediconnect.com/reset-password?token=abc%2B",
		PortalURL:           "https://ops.urmediconnect.com/",
		LogoURL:             "https://urmediconnect.com/img/logo.jpeg",
		SupportPhone:        "+91 7411558079",
		SupportEmail:        "support@urmediconnect.com",
		Year:                2026,
	})
	if err != nil {
		t.Fatalf("RenderEmployeeCreated MediAdmin: %v", err)
	}
	for _, want := range []string{
		"Priya Sharma",
		"9876543210",
		"reset-password?token=abc%2B",
		"https://ops.urmediconnect.com/",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("MediAdmin employee HTML missing %q", want)
		}
	}
}

func TestRenderLabCreated_MediAdmin(t *testing.T) {
	html, err := RenderLabCreated("MediAdmin", LabCreatedData{
		LabName:             "City Diagnostics",
		Username:            "9876543210",
		GeneratePasswordURL: "https://lab.urmediconnect.com/reset-password?token=abc%2B",
		PortalURL:           "https://lab.urmediconnect.com/",
		LogoURL:             "https://urmediconnect.com/img/logo.jpeg",
		SupportPhone:        "+91 7411558079",
		SupportEmail:        "support@urmediconnect.com",
		Year:                2026,
	})
	if err != nil {
		t.Fatalf("RenderLabCreated MediAdmin: %v", err)
	}
	for _, want := range []string{
		"City Diagnostics",
		"9876543210",
		"reset-password?token=abc%2B",
		"https://lab.urmediconnect.com/",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("MediAdmin lab HTML missing %q", want)
		}
	}
}
