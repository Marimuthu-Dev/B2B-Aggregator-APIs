package storage

import (
	"testing"
)

func TestValidateClientIDForMoUBlob(t *testing.T) {
	if err := validateClientIDForMoUBlob("123"); err != nil {
		t.Fatal(err)
	}
	if err := validateClientIDForMoUBlob(""); err == nil {
		t.Fatal("expected error for empty id")
	}
	if err := validateClientIDForMoUBlob("12a"); err == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestValidateAzureContainerName(t *testing.T) {
	if err := validateAzureContainerName("legal-documents"); err != nil {
		t.Fatal(err)
	}
	if err := validateAzureContainerName("ab"); err == nil {
		t.Fatal("expected error for short name")
	}
	if err := validateAzureContainerName("Bad"); err == nil {
		t.Fatal("expected error for uppercase")
	}
}

func TestClientMoUBlobObjectName(t *testing.T) {
	if got := ClientMoUBlobObjectName(" 99 "); got != "client-99-mou.pdf" {
		t.Fatalf("got %q", got)
	}
}
