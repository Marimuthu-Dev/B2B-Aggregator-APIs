package dto

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// MultipartFormMaxMemory is the max memory for parsing multipart forms (slightly above max MoU size).
const MultipartFormMaxMemory int64 = 6 << 20 // 6 MiB

// ClientCreateForm binds multipart form fields for POST /clients (same names as JSON; all text fields).
type ClientCreateForm struct {
	ClientName                string  `form:"ClientName" binding:"required"`
	Address                   string  `form:"Address" binding:"required"`
	CityID                    int8    `form:"CityID" binding:"required"`
	StateID                   int8    `form:"StateID" binding:"required"`
	Pincode                   string  `form:"Pincode" binding:"required"`
	ContactPerson1Name        string  `form:"ContactPerson1Name" binding:"required"`
	ContactPerson1Number      string  `form:"ContactPerson1Number" binding:"required"`
	ContactPerson1EmailID     string  `form:"ContactPerson1EmailID" binding:"required"`
	ContactPerson1Designation string  `form:"ContactPerson1Designation" binding:"required"`
	ContactPerson2Name        *string `form:"ContactPerson2Name"`
	ContactPerson2Number      *string `form:"ContactPerson2Number"`
	ContactPerson2EmailID     *string `form:"ContactPerson2EmailID"`
	ContactPerson2Designation *string `form:"ContactPerson2Designation"`
	CategoryID                *int8   `form:"CategoryID"`
	NatureOfBusiness          *string `form:"NatureOfBusiness"`
	GSTIN_UIN                 *string `form:"GSTIN_UIN"`
	PANNumber                 *string `form:"PANNumber"`
	BusinessVertical          string  `form:"BusinessVertical" binding:"required"`
	BillingName               *string `form:"BillingName"`
	BillingAdderss            *string `form:"BillingAdderss"`
	BillingPincode            *string `form:"BillingPincode"`
	ClientTypeID              *int8   `form:"ClientTypeID"`
	IsAcitve                  bool    `form:"IsAcitve"`
	MOUStartDate              string  `form:"MOUStartDate"`
	MOUEndDate                string  `form:"MOUEndDate"`
}

const mouFormField = "mou_document"

// ParseClientMultipartCreate parses multipart create: one text field per property (ClientName, Address, …)
// plus optional file part mou_document (PDF). No JSON blob field.
func ParseClientMultipartCreate(c *gin.Context) (ClientRequest, *multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(MultipartFormMaxMemory); err != nil {
		return ClientRequest{}, nil, fmt.Errorf("parse multipart form: %w", err)
	}
	var form ClientCreateForm
	if err := c.ShouldBind(&form); err != nil {
		return ClientRequest{}, nil, err
	}
	req, err := form.toClientRequest()
	if err != nil {
		return ClientRequest{}, nil, err
	}
	fh, err := c.FormFile(mouFormField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return req, nil, nil
		}
		return ClientRequest{}, nil, fmt.Errorf("mou_document: %w", err)
	}
	return req, fh, nil
}

func (f ClientCreateForm) toClientRequest() (ClientRequest, error) {
	req := ClientRequest{
		ClientName:                f.ClientName,
		Address:                   f.Address,
		CityID:                    f.CityID,
		StateID:                   f.StateID,
		Pincode:                   f.Pincode,
		ContactPerson1Name:        f.ContactPerson1Name,
		ContactPerson1Number:      f.ContactPerson1Number,
		ContactPerson1EmailID:     f.ContactPerson1EmailID,
		ContactPerson1Designation: f.ContactPerson1Designation,
		ContactPerson2Name:        f.ContactPerson2Name,
		ContactPerson2Number:      f.ContactPerson2Number,
		ContactPerson2EmailID:     f.ContactPerson2EmailID,
		ContactPerson2Designation: f.ContactPerson2Designation,
		CategoryID:                f.CategoryID,
		NatureOfBusiness:          f.NatureOfBusiness,
		GSTIN_UIN:                 f.GSTIN_UIN,
		PANNumber:                 f.PANNumber,
		BusinessVertical:          f.BusinessVertical,
		BillingName:               f.BillingName,
		BillingAdderss:            f.BillingAdderss,
		BillingPincode:            f.BillingPincode,
		ClientTypeID:              f.ClientTypeID,
		IsAcitve:                  f.IsAcitve,
	}
	if strings.TrimSpace(f.MOUStartDate) != "" {
		t, err := parseFlexibleMouDate(f.MOUStartDate)
		if err != nil {
			return ClientRequest{}, fmt.Errorf("MOUStartDate: %w", err)
		}
		req.MOUStartDate = t
	}
	if strings.TrimSpace(f.MOUEndDate) != "" {
		t, err := parseFlexibleMouDate(f.MOUEndDate)
		if err != nil {
			return ClientRequest{}, fmt.Errorf("MOUEndDate: %w", err)
		}
		req.MOUEndDate = t
	}
	return req, nil
}

// ParseClientMultipartUpdate parses optional per-field updates (text) and optional mou_document.
func ParseClientMultipartUpdate(c *gin.Context) (*ClientUpdateRequest, *multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(MultipartFormMaxMemory); err != nil {
		return nil, nil, fmt.Errorf("parse multipart form: %w", err)
	}
	req, err := buildClientUpdateRequestFromForm(c)
	if err != nil {
		return nil, nil, err
	}
	fh, err := c.FormFile(mouFormField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return req, nil, nil
		}
		return nil, nil, fmt.Errorf("mou_document: %w", err)
	}
	return req, fh, nil
}

func buildClientUpdateRequestFromForm(c *gin.Context) (*ClientUpdateRequest, error) {
	if c.Request.MultipartForm == nil {
		return &ClientUpdateRequest{}, nil
	}
	vals := c.Request.MultipartForm.Value
	var req ClientUpdateRequest
	if v, ok := firstFormValue(vals, "ClientName"); ok {
		req.ClientName = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "Address"); ok {
		req.Address = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "CityID"); ok {
		if n, err := strconv.ParseInt(v, 10, 8); err == nil {
			x := int8(n)
			req.CityID = &x
		}
	}
	if v, ok := firstFormValue(vals, "StateID"); ok {
		if n, err := strconv.ParseInt(v, 10, 8); err == nil {
			x := int8(n)
			req.StateID = &x
		}
	}
	if v, ok := firstFormValue(vals, "Pincode"); ok {
		req.Pincode = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson1Name"); ok {
		req.ContactPerson1Name = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson1Number"); ok {
		req.ContactPerson1Number = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson1EmailID"); ok {
		req.ContactPerson1EmailID = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson1Designation"); ok {
		req.ContactPerson1Designation = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson2Name"); ok {
		req.ContactPerson2Name = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson2Number"); ok {
		req.ContactPerson2Number = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson2EmailID"); ok {
		req.ContactPerson2EmailID = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ContactPerson2Designation"); ok {
		req.ContactPerson2Designation = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "CategoryID"); ok {
		if n, err := strconv.ParseInt(v, 10, 8); err == nil {
			x := int8(n)
			req.CategoryID = &x
		}
	}
	if v, ok := firstFormValue(vals, "NatureOfBusiness"); ok {
		req.NatureOfBusiness = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "GSTIN_UIN"); ok {
		req.GSTIN_UIN = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "PANNumber"); ok {
		req.PANNumber = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "BusinessVertical"); ok {
		req.BusinessVertical = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "BillingName"); ok {
		req.BillingName = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "BillingAdderss"); ok {
		req.BillingAdderss = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "BillingPincode"); ok {
		req.BillingPincode = stringPtr(v)
	}
	if v, ok := firstFormValue(vals, "ClientTypeID"); ok {
		if n, err := strconv.ParseInt(v, 10, 8); err == nil {
			x := int8(n)
			req.ClientTypeID = &x
		}
	}
	if v, ok := firstFormValue(vals, "IsAcitve"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			req.IsAcitve = &b
		}
	}
	if v, ok := firstFormValue(vals, "MOUStartDate"); ok && strings.TrimSpace(v) != "" {
		t, err := parseFlexibleMouDate(v)
		if err != nil {
			return nil, fmt.Errorf("MOUStartDate: %w", err)
		}
		req.MOUStartDate = t
	}
	if v, ok := firstFormValue(vals, "MOUEndDate"); ok && strings.TrimSpace(v) != "" {
		t, err := parseFlexibleMouDate(v)
		if err != nil {
			return nil, fmt.Errorf("MOUEndDate: %w", err)
		}
		req.MOUEndDate = t
	}
	return &req, nil
}

// parseFlexibleMouDate accepts RFC3339 / ISO-8601 (e.g. 2026-03-01T00:00:00.000Z) or date-only YYYY-MM-DD.
func parseFlexibleMouDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, nil
		} else {
			lastErr = err
		}
	}
	return nil, fmt.Errorf("use YYYY-MM-DD or RFC3339 datetime: %w", lastErr)
}

func firstFormValue(vals map[string][]string, key string) (string, bool) {
	a, ok := vals[key]
	if !ok || len(a) == 0 {
		return "", false
	}
	return a[0], true
}

func stringPtr(s string) *string {
	if s == "" {
		p := ""
		return &p
	}
	return &s
}
