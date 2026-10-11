package utility

import (
	"os"
	"testing"
)

func TestBaseEncryptAndDecrypt(t *testing.T) {
	testCases := []struct {
		name          string
		encryptionKey string
		plainText     string
	}{
		{
			name:          "DefaultFallbackKey",
			encryptionKey: "",
			plainText:     "1052",
		},
		{
			name:          "ShortKeyHashedWithSHA256",
			encryptionKey: "short_key",
			plainText:     "99999",
		},
		{
			name:          "Full32ByteKey",
			encryptionKey: "12345678901234567890123456789012",
			plainText:     "42",
		},
		{
			name:          "LongKeyTruncatedTo32",
			encryptionKey: "this_is_a_very_long_encryption_key_that_exceeds_32_bytes_easily!",
			plainText:     "lead_id_102030",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.encryptionKey != "" {
				t.Setenv("ENCRYPTION_KEY", tc.encryptionKey)
			} else {
				os.Unsetenv("ENCRYPTION_KEY")
			}

			encrypted := BaseEncryptUrlSafe(tc.plainText)
			if encrypted == "" {
				t.Fatalf("expected non-empty encrypted string, got empty")
			}

			decrypted := BaseDecryptUrlSafe(encrypted)
			if decrypted != tc.plainText {
				t.Fatalf("expected decrypted text %q, got %q", tc.plainText, decrypted)
			}
		})
	}
}

func TestBaseEncryptUrlSafe_EmptyInput(t *testing.T) {
	if got := BaseEncryptUrlSafe(""); got != "" {
		t.Fatalf("expected empty string for empty input, got %q", got)
	}
	if got := BaseDecryptUrlSafe(""); got != "" {
		t.Fatalf("expected empty string for empty input, got %q", got)
	}
}
