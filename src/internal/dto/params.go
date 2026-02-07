package dto

type IDParam struct {
	ID int64 `uri:"id" binding:"required"`
}

type PackageIDParam struct {
	ID int `uri:"id" binding:"required"`
}

type TestIDParam struct {
	ID int `uri:"id" binding:"required"`
}

type PackageMappingIDParam struct {
	ID int `uri:"id" binding:"required"`
}

type ClientIDPathParam struct {
	ClientID int64 `uri:"client_id" binding:"required"`
}

type ContactNumberQuery struct {
	ContactNumber string `form:"contactNumber" binding:"required"`
}
