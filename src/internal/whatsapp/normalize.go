package whatsapp

import (
	"regexp"
	"strings"
)

var nonDigitRegexp = regexp.MustCompile(`\D`)

// NormalizePhoneNumber safely normalizes an Indian phone number to include the 91 country code.
// E.g. "9004535221" -> "919004535221"
// "+91 90045 35221" -> "919004535221"
// "09004535221" -> "919004535221"
func NormalizePhoneNumber(phone string) string {
	digits := nonDigitRegexp.ReplaceAllString(phone, "")
	if len(digits) == 0 {
		return ""
	}

	// If it has leading zeros, strip them
	digits = strings.TrimLeft(digits, "0")

	if len(digits) == 10 {
		return "91" + digits
	}

	// If it's already 12 digits and starts with 91, it's correct
	if len(digits) == 12 && strings.HasPrefix(digits, "91") {
		return digits
	}

	// Otherwise, leave it as is (might be international number)
	return digits
}
