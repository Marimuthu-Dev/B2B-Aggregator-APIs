package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FlexDate unmarshals JSON date-only ("2006-01-02") or RFC3339 into time.Time.
// Use *FlexDate in DTOs so omit/null is preserved; call ToTimePtr() for *time.Time.
type FlexDate time.Time

func (t *FlexDate) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}
	// Try date-only first (e.g. "2026-03-03")
	parsed, err := time.Parse("2006-01-02", s)
	if err == nil {
		*t = FlexDate(parsed)
		return nil
	}
	// Fallback to RFC3339
	parsed, err = time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD or RFC3339: %w", err)
	}
	*t = FlexDate(parsed)
	return nil
}

// ToTimePtr returns nil if receiver is nil, else *time.Time. For use in ToDomain and update getters.
func (t *FlexDate) ToTimePtr() *time.Time {
	if t == nil {
		return nil
	}
	tt := time.Time(*t)
	return &tt
}

// FlexArrayString unmarshals JSON string or array (e.g. [1,2] or ["a","b"]) into a comma-separated string.
// Use *FlexArrayString in DTOs so omit/null is preserved; call ToStringPtr() for *string.
type FlexArrayString string

func (s *FlexArrayString) UnmarshalJSON(b []byte) error {
	// Try as array first
	var arr []interface{}
	if err := json.Unmarshal(b, &arr); err == nil {
		var parts []string
		for _, v := range arr {
			parts = append(parts, fmt.Sprint(v))
		}
		*s = FlexArrayString(strings.Join(parts, ","))
		return nil
	}
	// Try as string
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return fmt.Errorf("expected string or array: %w", err)
	}
	*s = FlexArrayString(str)
	return nil
}

// ToStringPtr returns nil if receiver is nil, else *string.
func (s *FlexArrayString) ToStringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}
