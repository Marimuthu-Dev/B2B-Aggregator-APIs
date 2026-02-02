package repository

import "strings"

func normalizeSortOrder(order string) string {
	value := strings.ToLower(order)
	if value == "desc" {
		return "desc"
	}
	return "asc"
}
