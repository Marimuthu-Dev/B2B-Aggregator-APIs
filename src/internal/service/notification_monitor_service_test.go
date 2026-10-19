package service

import (
	"testing"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
)

func TestFormatDurationSeconds(t *testing.T) {
	tests := []struct {
		secs int64
		want string
	}{
		{-5, "0s"},
		{0, "0s"},
		{45, "45s"},
		{60, "1m"},
		{90, "1m 30s"},
		{300, "5m"},
		{350, "5m 50s"},
		{3600, "1h"},
		{3660, "1h 1m"},
		{7325, "2h 2m"},
	}

	for _, tt := range tests {
		got := formatDurationSeconds(tt.secs)
		if got != tt.want {
			t.Errorf("formatDurationSeconds(%d) = %q; want %q", tt.secs, got, tt.want)
		}
	}
}

func TestComputeHealthStatus(t *testing.T) {
	tests := []struct {
		name    string
		backlog *domain.NotificationBacklog
		want    string
	}{
		{
			name:    "nil backlog",
			backlog: nil,
			want:    "Healthy",
		},
		{
			name: "all clear",
			backlog: &domain.NotificationBacklog{
				Unsent:          5,
				OverWarn:        0,
				OverCritical:    0,
				StuckProcessing: 0,
			},
			want: "Healthy",
		},
		{
			name: "warning on over warn",
			backlog: &domain.NotificationBacklog{
				Unsent:          5,
				OverWarn:        2,
				OverCritical:    0,
				StuckProcessing: 0,
			},
			want: "Warning",
		},
		{
			name: "critical on over critical",
			backlog: &domain.NotificationBacklog{
				Unsent:          5,
				OverWarn:        3,
				OverCritical:    1,
				StuckProcessing: 0,
			},
			want: "Critical",
		},
		{
			name: "critical on stuck processing",
			backlog: &domain.NotificationBacklog{
				Unsent:          5,
				OverWarn:        0,
				OverCritical:    0,
				StuckProcessing: 2,
			},
			want: "Critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeHealthStatus(tt.backlog)
			if got != tt.want {
				t.Errorf("computeHealthStatus() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimeIST(t *testing.T) {
	if formatTimeIST(nil) != nil {
		t.Errorf("formatTimeIST(nil) should be nil")
	}
	fixed := time.Date(2026, 10, 19, 14, 30, 0, 0, time.UTC)
	// UTC 14:30 is 20:00 IST (+05:30)
	got := formatTimeIST(&fixed)
	if got == nil || *got != "2026-10-19 20:00:00" {
		t.Errorf("formatTimeIST() = %v; want 2026-10-19 20:00:00", got)
	}
}
