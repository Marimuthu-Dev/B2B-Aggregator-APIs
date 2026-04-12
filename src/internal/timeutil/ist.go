package timeutil

import (
	"encoding/json"
	"time"
)

const istZone = "Asia/Kolkata"

var istLoc *time.Location

func init() {
	var err error
	istLoc, err = time.LoadLocation(istZone)
	if err != nil {
		istLoc = time.FixedZone("IST", 5*60*60+30*60) // UTC+5:30
	}
}

// ISTLocation returns the Asia/Kolkata time zone for IST.
func ISTLocation() *time.Location {
	return istLoc
}

// ISTTime wraps time.Time and marshals to JSON as RFC3339 in IST for API responses.
// The underlying time is stored as-is (typically UTC from DB); only serialization uses IST.
type ISTTime struct {
	time.Time
}

// MarshalJSON implements json.Marshaler: outputs the time in IST as RFC3339 string.
// Pointer receiver so both ISTTime and *ISTTime marshal correctly; nil and zero value output "null".
func (t *ISTTime) MarshalJSON() ([]byte, error) {
	if t == nil || t.IsZero() {
		return []byte("null"), nil
	}
	s := t.Time.In(istLoc).Format(time.RFC3339)
	return json.Marshal(s)
}

// FromTime wraps a time.Time (e.g. from DB) as ISTTime for API response.
func FromTime(t time.Time) ISTTime {
	return ISTTime{Time: t}
}

// FromTimePtr wraps *time.Time as *ISTTime for API response. Returns nil if t is nil.
func FromTimePtr(t *time.Time) *ISTTime {
	if t == nil {
		return nil
	}
	ist := ISTTime{Time: *t}
	return &ist
}

// ToTime returns the underlying time.Time (e.g. for persisting to DB).
func (t ISTTime) ToTime() time.Time {
	return t.Time
}

// ToTimePtr returns *time.Time from *ISTTime for persistence. Returns nil if t is nil.
func ToTimePtr(t *ISTTime) *time.Time {
	if t == nil {
		return nil
	}
	tt := t.Time
	return &tt
}

// StoredTime wraps time.Time when the database already stores the intended wall-clock instant
// (e.g. IST). JSON marshals as RFC3339 using the time's own location — it does not shift to Asia/Kolkata.
type StoredTime struct {
	time.Time
}

// MarshalJSON outputs RFC3339 without converting zones (unlike ISTTime).
func (t *StoredTime) MarshalJSON() ([]byte, error) {
	if t == nil || t.IsZero() {
		return []byte("null"), nil
	}
	s := t.Time.Format(time.RFC3339)
	return json.Marshal(s)
}

// StoredFromTime wraps t as StoredTime (e.g. from API input before save).
func StoredFromTime(t time.Time) StoredTime {
	return StoredTime{Time: t}
}

// StoredFromTimePtr wraps *time.Time as *StoredTime. Returns nil if t is nil.
func StoredFromTimePtr(t *time.Time) *StoredTime {
	if t == nil {
		return nil
	}
	s := StoredTime{Time: *t}
	return &s
}

// StoredToTimePtr returns *time.Time from *StoredTime for persistence. Returns nil if t is nil.
func StoredToTimePtr(t *StoredTime) *time.Time {
	if t == nil {
		return nil
	}
	tt := t.Time
	return &tt
}
