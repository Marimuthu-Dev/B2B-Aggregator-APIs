package dto

import "strings"

type PaginationQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=200"`
	SortBy   string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc ASC DESC"`
}

func (q PaginationQuery) Normalize(defaultSortBy string) PaginationQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.SortBy == "" {
		q.SortBy = defaultSortBy
	}
	q.SortOrder = strings.ToLower(q.SortOrder)
	if q.SortOrder == "" {
		q.SortOrder = "asc"
	}
	return q
}
