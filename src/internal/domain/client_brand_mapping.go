package domain

// ClientBrandMappingItem is an active row from MediAdmin.tbl_ClientBrandMapping (UID + BrandName).
type ClientBrandMappingItem struct {
	UID       int64
	BrandName string
}
