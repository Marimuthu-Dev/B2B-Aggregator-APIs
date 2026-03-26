package repository

import (
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/timeutil"

	"gorm.io/gorm"
)

const mouExpiringSoonDays = 15

// ApplyMouStatusOrFilter restricts rows whose MOU matches any of the given statuses (OR).
// Same rules for Client and Lab (MOUStartDate / MOUEndDate). IST calendar date.
func ApplyMouStatusOrFilter(query *gorm.DB, statuses []string) *gorm.DB {
	if len(statuses) == 0 {
		return query
	}
	ist := timeutil.ISTLocation()
	now := time.Now().In(ist)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)
	soonEnd := today.AddDate(0, 0, mouExpiringSoonDays)

	var ors []string
	var args []interface{}
	for _, s := range statuses {
		switch s {
		case "active":
			ors = append(ors, "(MOUStartDate IS NOT NULL AND MOUEndDate IS NOT NULL AND MOUStartDate <= ? AND MOUEndDate >= ?)")
			args = append(args, today, today)
		case "expired":
			ors = append(ors, "(MOUEndDate IS NOT NULL AND MOUEndDate < ?)")
			args = append(args, today)
		case "expiringSoon":
			ors = append(ors, "(MOUEndDate IS NOT NULL AND MOUEndDate >= ? AND MOUEndDate <= ?)")
			args = append(args, today, soonEnd)
		}
	}
	if len(ors) == 0 {
		return query
	}
	return query.Where(strings.Join(ors, " OR "), args...)
}
