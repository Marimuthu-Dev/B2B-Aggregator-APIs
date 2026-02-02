package dto

type IDParam struct {
	ID int64 `uri:"id" binding:"required"`
}

type PackageIDParam struct {
	ID int `uri:"id" binding:"required"`
}

type ContactNumberQuery struct {
	ContactNumber string `form:"contactNumber" binding:"required"`
}
