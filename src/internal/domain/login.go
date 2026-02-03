package domain

import "time"

type Login struct {
	RecordID      int64
	UserID        int64
	Pwd           string
	UserType      string
	CreatedOn     time.Time
	LastUpdatedOn time.Time
}

// ForgotPassword represents a forgot-password reset key record
type ForgotPassword struct {
	Uid                 int64
	UserID              int64
	UserType            string
	ForgetPasswordKey   string
	CreatedOn           time.Time
	ExpiryTimestamp     time.Time
	IsPasswordChanged   bool
	IsPasswordUpdatedOn *time.Time
}
