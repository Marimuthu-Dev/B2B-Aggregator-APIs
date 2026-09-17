package service

import (
	"testing"
	"time"

	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/pkg/utils"
)

func TestForgotPasswordSubject(t *testing.T) {
	if got := forgotPasswordSubject("MedLyfe"); got != "Reset Your MedLyfe Health Password" {
		t.Fatalf("MedLyfe subject: %q", got)
	}
	if got := forgotPasswordSubject("MediAdmin"); got != "Reset your UrMediconnect password" {
		t.Fatalf("MediAdmin subject: %q", got)
	}
}

func TestFormatLinkExpiry(t *testing.T) {
	if got := formatLinkExpiry(5 * time.Minute); got != "5 minutes" {
		t.Fatalf("got %q", got)
	}
}

func TestForgotPasswordRecipient_Store(t *testing.T) {
	st := &domain.Store{StoreName: "Koramangala Store", EmailID: "store@example.com", ContactNumber: "9000000001"}
	to, name, user := forgotPasswordRecipient(utils.UserTypeStore, st)
	if to != "store@example.com" || name != "Koramangala Store" || user != "9000000001" {
		t.Fatalf("got to=%q name=%q user=%q", to, name, user)
	}
}

func TestShouldQueueForgotPasswordEmail_MedLyfe(t *testing.T) {
	prev := persistencemodels.Schema()
	t.Cleanup(func() { persistencemodels.SetSchema(prev) })

	persistencemodels.SetSchema("MedLyfe")
	if !shouldQueueForgotPasswordEmail() {
		t.Fatal("MedLyfe should queue forgot password email")
	}

	persistencemodels.SetSchema("MediAdmin")
	if !shouldQueueForgotPasswordEmail() {
		t.Fatal("MediAdmin should still queue forgot password email")
	}
}
