package service

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildClientResetPasswordURL(t *testing.T) {
	key := "m3wJb4EFAtw4Bs//AkW7GCo/jBM+rDEm="
	got := buildClientResetPasswordURL("https://client.urmediconnect.com/", key)
	wantPrefix := "https://client.urmediconnect.com/reset-password?"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("prefix: got %q", got)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Query().Get("token") != key {
		t.Fatalf("token query: got %q", u.Query().Get("token"))
	}
	if strings.Contains(u.RawQuery, "+") {
		t.Fatalf("raw query should percent-encode +: %q", u.RawQuery)
	}
}
