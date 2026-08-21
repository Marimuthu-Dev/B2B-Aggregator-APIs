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
		"91 9036302806",
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
}
