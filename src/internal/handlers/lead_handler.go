package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/apperrors"
	"b2b-diagnostic-aggregator/apis/internal/domain"
	"b2b-diagnostic-aggregator/apis/internal/dto"
	"b2b-diagnostic-aggregator/apis/internal/middleware"
	"b2b-diagnostic-aggregator/apis/internal/repository"
	"b2b-diagnostic-aggregator/apis/internal/service"
	"b2b-diagnostic-aggregator/apis/internal/timeutil"
	"b2b-diagnostic-aggregator/apis/pkg/utils"

	"github.com/gin-gonic/gin"
)

type LeadHandler struct {
	svc service.LeadService
}

func NewLeadHandler(svc service.LeadService) *LeadHandler {
	return &LeadHandler{svc: svc}
}

func (h *LeadHandler) GetAll(c *gin.Context) {
	var query dto.LeadListQuery
	if !middleware.BindQuery(c, &query) {
		return
	}
	if err := enrichLeadListQueryFromPascalCaseKeys(c, &query); err != nil {
		respondError(c, err)
		return
	}
	applyLeadListScopeFromJWT(c, &query)
	if err := requireLeadListJWTSatisfied(c, &query); err != nil {
		respondError(c, err)
		return
	}
	page := query.PaginationQuery.Normalize("createdOn", 0)
	fitnessFilter := domain.LeadListFitnessFilterNone
	if raw := strings.TrimSpace(query.FitnessStatus); raw != "" {
		var err error
		fitnessFilter, err = domain.ParseLeadListFitnessFilter(raw)
		if err != nil {
			respondError(c, apperrors.NewBadRequest(err.Error(), err))
			return
		}
	}
	apptMin, apptMax, err := parseLeadListAppointmentAtRange(&query)
	if err != nil {
		respondError(c, err)
		return
	}
	filter := repository.LeadListFilter{
		Page:           page.Page,
		PageSize:       page.PageSize,
		SortBy:         page.SortBy,
		SortOrder:      page.SortOrder,
		LeadID:         query.LeadID,
		ClientID:       query.ClientID,
		LabID:          query.LabID,
		StatusID:       query.StatusID,
		PackageID:      query.PackageID,
		CollectionType: query.CollectionType,
		StoreID:          query.StoreID,
		Search:                    query.Search,
		FitnessStatus:             fitnessFilter,
		AppointmentAtMin: apptMin,
		AppointmentAtMax: apptMax,
	}

	data, total, err := h.svc.ListLeads(filter)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, data, "Success", gin.H{
		"Count":    len(data),
		"Page":     filter.Page,
		"PageSize": filter.PageSize,
		"Total":    total,
	})
}

func (h *LeadHandler) GetByID(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	data, err := h.svc.GetLeadByID(params.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	if !leadDetailAccessibleByJWT(c, data) {
		respondError(c, apperrors.NewNotFound("Lead not found", nil))
		return
	}
	respondData(c, http.StatusOK, data, "Success", nil)
}

func (h *LeadHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.LeadRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	lead := req.ToDomain()
	if err := h.svc.CreateLead(&lead, userID); err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, lead, "Lead created successfully", nil)
}

func (h *LeadHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.LeadUpdateRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if !req.HasAtLeastOneField() {
		respondError(c, apperrors.NewBadRequest("At least one field is required in the payload to update", nil))
		return
	}
	lead, err := h.svc.UpdateLead(params.ID, &req, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, lead, "Lead updated successfully", nil)
}

func (h *LeadHandler) Delete(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	actorID, _ := middleware.GetUserID(c)
	if actorID == 0 {
		actorID = 1
	}
	if err := h.svc.DeleteLead(params.ID, actorID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Lead deleted successfully")
}

func (h *LeadHandler) BulkUpdateStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	var req dto.BulkUpdateLeadStatusRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	count, err := h.svc.BulkUpdateLeadStatus(req.LeadIDs, req.LeadStatusID, userID, req.LabID, req.AppointmentAt)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, gin.H{"UpdatedCount": count}, "Lead statuses updated successfully", nil)
}

// UploadReport handles POST /api/v1/leads/{id}/reports/upload (multipart field "file", PDF).
func (h *LeadHandler) UploadReport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	userType, typeOK := middleware.GetUserType(c)
	if !typeOK {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType != utils.UserTypeEmployee && userType != utils.UserTypeLab {
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		respondError(c, apperrors.NewBadRequest("PDF file is required", err))
		return
	}

	reportURL, err := h.svc.UploadBloodTestReport(c.Request.Context(), params.ID, userID, fh)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, gin.H{"ReportURL": reportURL}, "Report uploaded successfully", nil)
}

// ApproveReport updates lead report approval (fit / unfit / hold), download flag, and remarks.
//
// OpenAPI 3.0 (informal):
//
//	post:
//	  summary: Approve lead report status
//	  operationId: approveLeadReport
//	  path: /api/v1/leads/{id}/reports/approve
//	  tags: [Leads]
//	  security: [{ bearerAuth: [] }]
//	  parameters:
//	    - name: id
//	      in: path
//	      required: true
//	      schema: { type: integer, format: int64 }
//	  requestBody:
//	    required: true
//	    content:
//	      application/json:
//	        schema:
//	          type: object
//	          required: [status, isFitCertificateToBeGenerated]
//	          properties:
//	            status: { type: string, enum: [fit, unfit, hold] }
//	            remarks: { type: string, maxLength: 250 }
//	            allowDownload: { type: boolean }
//	            isFitCertificateToBeGenerated: { type: boolean, description: "Persisted to IsFitCertificateTobeGenerated; whether the fitness certificate pipeline should run for this lead." }
//	            BrandID: { type: integer, format: int64, nullable: true, description: "Client brand mapping UID (tbl_ClientBrandMapping). null or omitted → BrandID column not updated." }
//	  responses:
//	    "200":
//	      description: OK
//	      content:
//	        application/json:
//	          schema:
//	            type: object
//	            properties:
//	              success: { type: boolean, example: true }
//	              message: { type: string }
//	              timestamp: { type: string, format: date-time }
func (h *LeadHandler) ApproveReport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	userType, typeOK := middleware.GetUserType(c)
	if !typeOK {
		respondError(c, apperrors.NewUnauthorized("Authentication required", nil))
		return
	}
	if userType != utils.UserTypeEmployee {
		respondError(c, apperrors.NewForbidden("You are not authorized for this activity.", nil))
		return
	}
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	var req dto.ApproveLeadRequest
	if !middleware.BindJSON(c, &req) {
		return
	}
	if err := h.svc.ApproveLeadReport(params.ID, &req, userID); err != nil {
		respondError(c, err)
		return
	}
	respondMessage(c, http.StatusOK, "Lead report status updated successfully")
}

// GetReportDownloadURL handles GET /api/v1/leads/{id}/reports/download-url (short-lived SAS URL for the report PDF).
func (h *LeadHandler) GetReportDownloadURL(c *gin.Context) {
	var params dto.IDParam
	if !middleware.BindUri(c, &params) {
		return
	}
	if !middleware.RequirePositiveID(c, params.ID) {
		return
	}
	userType := 0
	if ut, ok := middleware.GetUserType(c); ok {
		userType = ut
	}
	userID, _ := middleware.GetUserID(c)
	downloadURL, expiresAt, err := h.svc.GetLeadReportDownloadURL(c.Request.Context(), params.ID, userType, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusOK, gin.H{
		"DownloadURL": downloadURL,
		"ExpiresAt":   expiresAt.UTC().Format(time.RFC3339),
	}, "Success", nil)
}

func (h *LeadHandler) BulkImportCsv(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil || file == nil {
		respondError(c, apperrors.NewBadRequest("CSV file is required", err))
		return
	}
	clientIDStr := c.PostForm("ClientID")
	packageIDStr := c.PostForm("PackageID")
	if clientIDStr == "" || packageIDStr == "" {
		respondError(c, apperrors.NewBadRequest("ClientID and PackageID are required in the request body", nil))
		return
	}
	clientID, err1 := strconv.ParseInt(clientIDStr, 10, 64)
	packageID, err2 := strconv.Atoi(packageIDStr)
	if err1 != nil || err2 != nil || clientID <= 0 || packageID <= 0 {
		respondError(c, apperrors.NewBadRequest("ClientID and PackageID must be positive integers", nil))
		return
	}

	f, err := file.Open()
	if err != nil {
		respondError(c, apperrors.NewBadRequest("Failed to read file", err))
		return
	}
	defer f.Close()
	buf := make([]byte, file.Size+1)
	n, _ := f.Read(buf)
	buf = buf[:n]

	userID, _ := middleware.GetUserID(c)
	if userID == 0 {
		userID = 1
	}
	inserted, err := h.svc.BulkImportFromCSV(buf, clientID, packageID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, http.StatusCreated, gin.H{"InsertedCount": inserted}, "Leads imported successfully", nil)
}

// applyLeadListScopeFromJWT forces client/lab list scope from JWT: userType 2 → ClientID = userId;
// userType 3 → LabID = userId (matches login token payload). Employees are unchanged; query filters
// from the URL cannot widen scope for client/lab users.
func applyLeadListScopeFromJWT(c *gin.Context, q *dto.LeadListQuery) {
	userID, ok := middleware.GetUserID(c)
	if !ok || userID <= 0 {
		return
	}
	userType, ok := middleware.GetUserType(c)
	if !ok {
		return
	}
	switch userType {
	case utils.UserTypeClient:
		q.ClientID = &userID
	case utils.UserTypeLab:
		q.LabID = &userID
	}
}

// requireLeadListJWTSatisfied ensures client/lab callers always have a forced filter (no unscoped list).
func requireLeadListJWTSatisfied(c *gin.Context, q *dto.LeadListQuery) error {
	userType, ok := middleware.GetUserType(c)
	if !ok {
		return nil
	}
	switch userType {
	case utils.UserTypeClient:
		if q.ClientID == nil {
			return apperrors.NewUnauthorized("Authentication required", nil)
		}
	case utils.UserTypeLab:
		if q.LabID == nil {
			return apperrors.NewUnauthorized("Authentication required", nil)
		}
	}
	return nil
}

// leadDetailAccessibleByJWT enforces the same scope for single-lead reads (client / lab).
func leadDetailAccessibleByJWT(c *gin.Context, d *domain.LeadDetail) bool {
	if d == nil {
		return false
	}
	userID, idOK := middleware.GetUserID(c)
	userType, typeOK := middleware.GetUserType(c)
	if !idOK || userID <= 0 || !typeOK {
		return false
	}
	switch userType {
	case utils.UserTypeEmployee:
		return true
	case utils.UserTypeClient:
		return d.ClientID == userID
	case utils.UserTypeLab:
		if d.LabID == nil {
			return false
		}
		return *d.LabID == userID
	default:
		return false
	}
}

// parseLeadListAppointmentAtRange builds IST bounds for l.AppointmentAt: from at 00:00:00, to at 23:59:59.999999999 (inclusive).
// Dates must be YYYY-MM-DD. Either bound may be omitted.
func parseLeadListAppointmentAtRange(q *dto.LeadListQuery) (min *time.Time, maxInclusive *time.Time, err error) {
	fromStr := strings.TrimSpace(q.AppointmentAtFrom)
	toStr := strings.TrimSpace(q.AppointmentAtTo)
	if fromStr == "" && toStr == "" {
		return nil, nil, nil
	}
	loc := timeutil.ISTLocation()
	const layout = "2006-01-02"
	if fromStr != "" {
		d, e := time.ParseInLocation(layout, fromStr, loc)
		if e != nil {
			return nil, nil, apperrors.NewBadRequest("Invalid appointmentAtFrom: use YYYY-MM-DD (calendar date in India Standard Time)", e)
		}
		start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		min = &start
	}
	if toStr != "" {
		d, e := time.ParseInLocation(layout, toStr, loc)
		if e != nil {
			return nil, nil, apperrors.NewBadRequest("Invalid appointmentAtTo: use YYYY-MM-DD (calendar date in India Standard Time)", e)
		}
		end := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 999999999, loc)
		maxInclusive = &end
	}
	if min != nil && maxInclusive != nil && min.After(*maxInclusive) {
		return nil, nil, apperrors.NewBadRequest("appointmentAtFrom must be on or before appointmentAtTo", nil)
	}
	return min, maxInclusive, nil
}

// enrichLeadListQueryFromPascalCaseKeys fills filters from alternate query spellings (PascalCase, lowercase)
// when Gin did not bind the camelCase form tag. Also normalizes collectionType / CollectionType (Home | Center | Camp).
func enrichLeadListQueryFromPascalCaseKeys(c *gin.Context, q *dto.LeadListQuery) error {
	if err := mergePositiveInt64QueryMulti(c, &q.LeadID, "LeadID", "leadid"); err != nil {
		return err
	}
	if err := mergePositiveInt64QueryMulti(c, &q.ClientID, "ClientID", "clientid"); err != nil {
		return err
	}
	if err := mergePositiveInt64QueryMulti(c, &q.LabID, "LabID", "labid"); err != nil {
		return err
	}
	if err := mergePositiveIntQueryMulti(c, &q.PackageID, "PackageID", "packageid"); err != nil {
		return err
	}
	if strings.TrimSpace(q.Search) == "" {
		if s := strings.TrimSpace(c.Query("Search")); s != "" {
			q.Search = s
		}
	}
	if q.StoreID == nil {
		if s := strings.TrimSpace(c.Query("StoreID")); s != "" {
			q.StoreID = &s
		} else if s := strings.TrimSpace(c.Query("storeid")); s != "" {
			q.StoreID = &s
		}
	}
	if strings.TrimSpace(q.FitnessStatus) == "" {
		if s := strings.TrimSpace(c.Query("FitnessStatus")); s != "" {
			q.FitnessStatus = s
		}
	}
	if strings.TrimSpace(q.AppointmentAtFrom) == "" {
		if s := strings.TrimSpace(c.Query("AppointmentAtFrom")); s != "" {
			q.AppointmentAtFrom = s
		}
	}
	if strings.TrimSpace(q.AppointmentAtTo) == "" {
		if s := strings.TrimSpace(c.Query("AppointmentAtTo")); s != "" {
			q.AppointmentAtTo = s
		}
	}
	return mergeLeadCollectionTypeQueryParam(c, q)
}

func mergeLeadCollectionTypeQueryParam(c *gin.Context, q *dto.LeadListQuery) error {
	raw := ""
	if q.CollectionType != nil {
		raw = strings.TrimSpace(*q.CollectionType)
	}
	if raw == "" {
		raw = strings.TrimSpace(c.Query("collectionType"))
	}
	if raw == "" {
		raw = strings.TrimSpace(c.Query("CollectionType"))
	}
	if raw == "" {
		q.CollectionType = nil
		return nil
	}
	norm, err := domain.ParseLeadCollectionType(raw)
	if err != nil {
		return apperrors.NewBadRequest("Invalid query parameter collectionType: use Home, Center, or Camp", err)
	}
	q.CollectionType = &norm
	return nil
}

// mergePositiveInt64QueryMulti sets *dest from the first non-empty query key among keys (order preserved).
func mergePositiveInt64QueryMulti(c *gin.Context, dest **int64, keys ...string) error {
	if *dest != nil {
		return nil
	}
	for _, key := range keys {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" {
			continue
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id < 1 {
			return apperrors.NewBadRequest("Invalid query parameter "+key+": must be a positive integer", err)
		}
		*dest = &id
		return nil
	}
	return nil
}

func mergePositiveIntQueryMulti(c *gin.Context, dest **int, keys ...string) error {
	if *dest != nil {
		return nil
	}
	for _, key := range keys {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" {
			continue
		}
		n64, err := strconv.ParseInt(raw, 10, 0)
		if err != nil || n64 < 1 {
			return apperrors.NewBadRequest("Invalid query parameter "+key+": must be a positive integer", err)
		}
		v := int(n64)
		*dest = &v
		return nil
	}
	return nil
}
