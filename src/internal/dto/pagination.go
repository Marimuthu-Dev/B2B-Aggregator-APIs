package dto

import "strings"

type PaginationQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"pageSize" binding:"omitempty,min=1,max=1000"`
	SortBy   string `form:"sortBy" binding:"omitempty"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc ASC DESC"`
}

const DefaultPageSize = 20

// Normalize sets defaults for pagination. defaultPageSizeOverride: if > 0, use as default when client did not send pageSize (e.g. 500 for client/lab "get all"); 0 means use DefaultPageSize (20).
func (q PaginationQuery) Normalize(defaultSortBy string, defaultPageSizeOverride int) PaginationQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		if defaultPageSizeOverride > 0 {
			q.PageSize = defaultPageSizeOverride
		} else {
			q.PageSize = DefaultPageSize
		}
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
