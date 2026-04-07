package domain

import (
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"time"
)

type Login struct {
	RecordID      int64
	UserID        int64
	Pwd           string
	UserType      string
	CreatedOn     timeutil.ISTTime
	LastUpdatedOn timeutil.ISTTime
}

// ForgotPassword represents a forgot-password reset key record
type ForgotPassword struct {
	UID                 int64
	UserID              int64
	UserType            string
	ForgetPasswordKey   string
	CreatedOn           timeutil.ISTTime
	ExpiryTimestamp     timeutil.ISTTime
	IsPasswordChanged   bool
	PasswordUpdatedOn   *time.Time
}
