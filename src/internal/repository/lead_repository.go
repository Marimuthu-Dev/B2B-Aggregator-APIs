package repository

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"

	"gorm.io/gorm"
)

type LeadRepository interface {
	FindAll() ([]domain.Lead, error)
	List(filter LeadListFilter) ([]domain.Lead, int64, error)
	FindByID(id int64) (*domain.Lead, error)
	ExistsByID(id int64) (bool, error)
	Create(l *domain.Lead) error
	Update(l *domain.Lead) error
	Delete(id int64) error
	// UpdateStatusForIDs sets status and audit columns; if labID is non-nil, LabID is set; if appointmentAt is non-nil, AppointmentAt is set.
	UpdateStatusForIDs(leadIDs []int64, statusID int8, lastUpdatedBy int64, labID *int64, appointmentAt *time.Time) (int64, error)
	FindByClientID(clientID int64) ([]domain.Lead, error)
	FindByStatus(statusID int8) ([]domain.Lead, error)
	FindByPackage(packageID int) ([]domain.Lead, error)
	FindByPatientID(patientID string) (*domain.Lead, error)
	FindByContactNumber(contactNumber string) ([]domain.Lead, error)
	FindByEmail(email string) ([]domain.Lead, error)
	// FindActiveLeadStatusIDByName resolves LeadStatusID from MediAdmin.tbl_LeadStatusMaster (IsActive = 1).
	FindActiveLeadStatusIDByName(name string) (int8, error)
	UpdateLeadReportURLAndStatus(leadID int64, reportURL string, statusID int8, userID int64) error
	// FindLeadsPendingFitCertification returns FIT leads pending worker processing. Does not filter on IsFitCertificateTobeGenerated (all values eligible by query).
	FindLeadsPendingFitCertification(limit int, pendingLeadStatusID int8) ([]domain.Lead, error)
	// MarkFitCertificationGenerated sets certification flags and status if the lead still matches pending criteria; logs history. Returns whether a row was updated.
	MarkFitCertificationGenerated(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (updated bool, err error)
	// MarkReportReadyToDownload advances a lead to the downloadable state without generating a certificate; logs history. Returns whether a row was updated.
	MarkReportReadyToDownload(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (updated bool, err error)
	// UpdateLeadReportApproval sets IsFit (0=hold, 1=fit, 2=unfit), download flag, IsFitCertificateTobeGenerated, remarks, FitUpdatedOn, and last-updated audit. Only rows with LeadStatusID = 8 (uploaded) match; when IsFit = 1, LeadStatusID is set to 9 (approved). Otherwise LeadStatusID is not changed. remarks nil or empty clears ApprovalRemarks. brandID nil → BrandID column unchanged.
	UpdateLeadReportApproval(leadID int64, lastUpdatedBy int64, isFit int8, allowDownload bool, isFitCertificateTobeGenerated bool, remarks *string, brandID *int64) (rowsAffected int64, err error)
}

type leadRepository struct {
	db *gorm.DB
}

// leadListScan is tbl_Leads with optional names from LEFT JOINs to lab, client, city, state, and (MedLyfe) store masters.
type leadListScan struct {
	persistencemodels.Lead
	JoinedLabName    sql.NullString `gorm:"column:joined_lab_name"`
	JoinedClientName sql.NullString `gorm:"column:joined_client_name"`
	JoinedCityName   sql.NullString `gorm:"column:joined_city_name"`
	JoinedStateName  sql.NullString `gorm:"column:joined_state_name"`
	JoinedStoreName  sql.NullString `gorm:"column:joined_store_name"`
	JoinedStoreCity  sql.NullString `gorm:"column:joined_store_city"`
}

// leadByIDLocationScan is used for FindByID city/state names (and MedLyfe store name/city); lab/client filled in service.
type leadByIDLocationScan struct {
	persistencemodels.Lead
	JoinedCityName  sql.NullString `gorm:"column:joined_city_name"`
	JoinedStateName sql.NullString `gorm:"column:joined_state_name"`
	JoinedStoreName sql.NullString `gorm:"column:joined_store_name"`
	JoinedStoreCity sql.NullString `gorm:"column:joined_store_city"`
}

func NewLeadRepository(db *gorm.DB) LeadRepository {
	return &leadRepository{db: db}
}

// gormLead omits StoreMasterID unless DB_SCHEMA is MedLyfe (column is absent on other schemas).
func gormLead(db *gorm.DB) *gorm.DB {
	if persistencemodels.HasLeadStoreMasterIDColumn() {
		return db
	}
	return db.Omit("StoreMasterID")
}

func (r *leadRepository) FindAll() ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) List(filter LeadListFilter) ([]domain.Lead, int64, error) {
	q := r.leadListJoinedQuery(filter)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := "l." + mapLeadSortColumn(filter.SortBy)
	order := normalizeSortOrder(filter.SortOrder)
	offset := (filter.Page - 1) * filter.PageSize

	var rows []leadListScan
	err := gormLead(r.leadListJoinedQuery(filter)).
		Select(leadListSelectColumns()).
		Order(sortColumn + " " + order).
		Limit(filter.PageSize).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	out := make([]domain.Lead, len(rows))
	for i := range rows {
		out[i] = mapLeadToDomainWithOptionalJoinedNames(rows[i].Lead, leadJoinedNames{
			LabName:    rows[i].JoinedLabName,
			ClientName: rows[i].JoinedClientName,
			CityName:   rows[i].JoinedCityName,
			StateName:  rows[i].JoinedStateName,
			StoreName:  rows[i].JoinedStoreName,
			StoreCity:  rows[i].JoinedStoreCity,
		})
	}
	return out, total, nil
}

func leadListSelectColumns() string {
	cols := "l.*, lm.LabName AS joined_lab_name, cm.ClientName AS joined_client_name, ctm.CityName AS joined_city_name, stm.StateName AS joined_state_name"
	if persistencemodels.HasStoreMasterTable() {
		cols += ", sm.StoreName AS joined_store_name, smctm.CityName AS joined_store_city"
	}
	return cols
}

func (r *leadRepository) leadListJoinedQuery(filter LeadListFilter) *gorm.DB {
	leadTable := persistencemodels.Lead{}.TableName()
	labTable := persistencemodels.Lab{}.TableName()
	clientTable := persistencemodels.Client{}.TableName()
	cityTable := persistencemodels.CityMaster{}.TableName()
	stateTable := persistencemodels.StateMaster{}.TableName()
	q := r.db.Table(leadTable + " AS l").
		Joins("LEFT JOIN " + labTable + " AS lm ON l.LabID = lm.LabID").
		Joins("LEFT JOIN " + clientTable + " AS cm ON l.ClientID = cm.ClientID").
		Joins("LEFT JOIN " + cityTable + " AS ctm ON l.CityID = ctm.CityID").
		Joins("LEFT JOIN " + stateTable + " AS stm ON l.StateID = stm.StateID")
	if persistencemodels.HasStoreMasterTable() {
		storeTable := persistencemodels.Store{}.TableName()
		q = q.Joins("LEFT JOIN " + storeTable + " AS sm ON l.StoreMasterID = sm.StoreID").
			Joins("LEFT JOIN " + cityTable + " AS smctm ON sm.CityID = smctm.CityID")
	}
	if filter.LeadID != nil {
		q = q.Where("l.LeadID = ?", *filter.LeadID)
	}
	if filter.ClientID != nil {
		q = q.Where("l.ClientID = ?", *filter.ClientID)
	}
	if filter.LabID != nil {
		q = q.Where("l.LabID = ?", *filter.LabID)
	}
	if filter.StatusID != nil {
		q = q.Where("l.LeadStatusID = ?", *filter.StatusID)
	}
	if filter.PackageID != nil {
		q = q.Where("l.PackageID = ?", *filter.PackageID)
	}
	if filter.CollectionType != nil && *filter.CollectionType != "" {
		q = q.Where("l.CollectionType = ?", *filter.CollectionType)
	}
	if filter.RestrictToStoreID != nil {
		sid := strconv.FormatInt(*filter.RestrictToStoreID, 10)
		if persistencemodels.HasLeadStoreMasterIDColumn() {
			q = q.Where("(l.StoreMasterID = ? OR l.StoreID = ?)", *filter.RestrictToStoreID, sid)
		} else {
			q = q.Where("l.StoreID = ?", sid)
		}
	} else {
		if filter.StoreID != nil {
			if storeID := strings.TrimSpace(*filter.StoreID); storeID != "" {
				q = q.Where("l.StoreID = ?", storeID)
			}
		}
		if filter.StoreMasterID != nil && persistencemodels.HasLeadStoreMasterIDColumn() {
			q = q.Where("l.StoreMasterID = ?", *filter.StoreMasterID)
		}
	}
	if trimmed := strings.TrimSpace(filter.Search); trimmed != "" {
		term := "%" + trimmed + "%"
		q = q.Where("(l.PatientName LIKE ? OR l.ContactNumber LIKE ? OR l.EmailID LIKE ? OR l.StoreID LIKE ?)", term, term, term, term)
	}
	if filter.AppointmentAtMin != nil {
		q = q.Where("l.AppointmentAt IS NOT NULL AND l.AppointmentAt >= ?", *filter.AppointmentAtMin)
	}
	if filter.AppointmentAtMax != nil {
		q = q.Where("l.AppointmentAt IS NOT NULL AND l.AppointmentAt <= ?", *filter.AppointmentAtMax)
	}
	switch filter.FitnessStatus {
	case domain.LeadListFitnessFilterEmpty:
		q = q.Where("l.IsFit IS NULL")
	case domain.LeadListFitnessFilterOnHold:
		q = q.Where("l.IsFit = ?", domain.LeadFitHold)
	case domain.LeadListFitnessFilterFit:
		q = q.Where("l.IsFit = ?", domain.LeadFitFit)
	case domain.LeadListFitnessFilterUnfit:
		q = q.Where("(l.IsFit = ? OR l.IsFit = ?)", domain.LeadFitUnfit, int8(-1))
	}
	return q
}

func mapLeadSortColumn(sortBy string) string {
	switch sortBy {
	case "patientName":
		return "PatientName"
	case "clientId":
		return "ClientID"
	case "labId":
		return "LabID"
	case "statusId":
		return "LeadStatusID"
	case "createdOn":
		return "CreatedOn"
	case "collectionType":
		return "CollectionType"
	case "leadId":
		return "LeadID"
	default:
		return "LeadID"
	}
}

func (r *leadRepository) FindByID(id int64) (*domain.Lead, error) {
	leadTable := persistencemodels.Lead{}.TableName()
	cityTable := persistencemodels.CityMaster{}.TableName()
	stateTable := persistencemodels.StateMaster{}.TableName()
	q := gormLead(r.db).Table(leadTable+" AS l").
		Joins("LEFT JOIN "+cityTable+" AS ctm ON l.CityID = ctm.CityID").
		Joins("LEFT JOIN "+stateTable+" AS stm ON l.StateID = stm.StateID")
	selectCols := "l.*, ctm.CityName AS joined_city_name, stm.StateName AS joined_state_name"
	if persistencemodels.HasStoreMasterTable() {
		storeTable := persistencemodels.Store{}.TableName()
		q = q.Joins("LEFT JOIN "+storeTable+" AS sm ON l.StoreMasterID = sm.StoreID").
			Joins("LEFT JOIN "+cityTable+" AS smctm ON sm.CityID = smctm.CityID")
		selectCols += ", sm.StoreName AS joined_store_name, smctm.CityName AS joined_store_city"
	}
	var row leadByIDLocationScan
	err := q.Where("l.LeadID = ?", id).Select(selectCols).First(&row).Error
	if err != nil {
		return nil, err
	}
	domainLead := mapLeadToDomainWithOptionalJoinedNames(row.Lead, leadJoinedNames{
		CityName:  row.JoinedCityName,
		StateName: row.JoinedStateName,
		StoreName: row.JoinedStoreName,
		StoreCity: row.JoinedStoreCity,
	})
	return &domainLead, nil
}

func (r *leadRepository) ExistsByID(id int64) (bool, error) {
	var count int64
	if err := gormLead(r.db).Model(&persistencemodels.Lead{}).Where("LeadID = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *leadRepository) Create(l *domain.Lead) error {
	persist := mapLeadToPersistence(*l)
	if err := gormLead(r.db).Create(&persist).Error; err != nil {
		return err
	}
	*l = mapLeadToDomain(persist)
	return nil
}

func (r *leadRepository) Update(l *domain.Lead) error {
	persist := mapLeadToPersistence(*l)
	if err := gormLead(r.db).Save(&persist).Error; err != nil {
		return err
	}
	*l = mapLeadToDomain(persist)
	return nil
}

func (r *leadRepository) Delete(id int64) error {
	return gormLead(r.db).Delete(&persistencemodels.Lead{}, id).Error
}

func (r *leadRepository) UpdateStatusForIDs(leadIDs []int64, statusID int8, lastUpdatedBy int64, labID *int64, appointmentAt *time.Time) (int64, error) {
	updates := map[string]interface{}{
		"LeadStatusID":  statusID,
		"LastUpdatedBy": lastUpdatedBy,
		"LastUpdatedOn": time.Now(),
	}
	if labID != nil {
		updates["LabID"] = *labID
	}
	if appointmentAt != nil {
		updates["AppointmentAt"] = *appointmentAt
	}
	result := gormLead(r.db).Model(&persistencemodels.Lead{}).Where("LeadID IN ?", leadIDs).Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *leadRepository) FindByClientID(clientID int64) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Where("ClientID = ?", clientID).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByStatus(statusID int8) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Where("LeadStatusID = ?", statusID).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByPackage(packageID int) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Where("PackageID = ?", packageID).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByPatientID(patientID string) (*domain.Lead, error) {
	var l persistencemodels.Lead
	err := gormLead(r.db).Where("PatientID = ?", patientID).First(&l).Error
	if err != nil {
		return nil, err
	}
	domainLead := mapLeadToDomain(l)
	return &domainLead, nil
}

func (r *leadRepository) FindByContactNumber(contactNumber string) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Where("ContactNumber = ?", contactNumber).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindByEmail(email string) ([]domain.Lead, error) {
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Where("EmailID = ?", email).Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) FindActiveLeadStatusIDByName(name string) (int8, error) {
	var statusID int8
	err := r.db.Table(persistencemodels.Table("tbl_LeadStatusMaster")).
		Select("LeadStatusID").
		Where("LeadStatusName = ? AND IsActive = ?", name, true).
		Take(&statusID).Error
	if err != nil {
		return 0, err
	}
	return statusID, nil
}

func (r *leadRepository) UpdateLeadReportURLAndStatus(leadID int64, reportURL string, statusID int8, userID int64) error {
	result := gormLead(r.db).Model(&persistencemodels.Lead{}).Where("LeadID = ?", leadID).Updates(map[string]interface{}{
		"ReportURL":     reportURL,
		"LeadStatusID":  statusID,
		"LastUpdatedBy": userID,
		"LastUpdatedOn": time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *leadRepository) FindLeadsPendingFitCertification(limit int, pendingLeadStatusID int8) ([]domain.Lead, error) {
	if limit <= 0 {
		limit = 10
	}
	var leads []persistencemodels.Lead
	err := gormLead(r.db).Where("LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?",
		pendingLeadStatusID, domain.LeadFitFit, false).
		Where("ReportURL IS NOT NULL AND LTRIM(RTRIM(ReportURL)) <> ''").
		Order("LeadID ASC").
		Limit(limit).
		Find(&leads).Error
	return mapLeadsToDomain(leads), err
}

func (r *leadRepository) MarkFitCertificationGenerated(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (bool, error) {
	var updated bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := gormLead(tx).Model(&persistencemodels.Lead{}).
			Where("LeadID = ? AND LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?",
				leadID, fromLeadStatusID, domain.LeadFitFit, false).
			Updates(map[string]interface{}{
				"IsFitCertifiedGenerated": true,
				"FitCertifiedGeneratedOn": now,
				"LeadStatusID":            toLeadStatusID,
				"LastUpdatedBy":           userID,
				"LastUpdatedOn":           now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		updated = true
		h := persistencemodels.LeadHistory{
			LeadID:    leadID,
			Action:    domain.LeadActionFitCertificationGenerated,
			CreatedBy: userID,
			CreatedOn: now,
		}
		return tx.Create(&h).Error
	})
	return updated, err
}

func (r *leadRepository) MarkReportReadyToDownload(leadID int64, userID int64, fromLeadStatusID, toLeadStatusID int8) (bool, error) {
	var updated bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := gormLead(tx).Model(&persistencemodels.Lead{}).
			Where("LeadID = ? AND LeadStatusID = ? AND IsFit = ? AND IsFitCertifiedGenerated = ?",
				leadID, fromLeadStatusID, domain.LeadFitFit, false).
			Updates(map[string]interface{}{
				"LeadStatusID":  toLeadStatusID,
				"LastUpdatedBy": userID,
				"LastUpdatedOn": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		updated = true
		h := persistencemodels.LeadHistory{
			LeadID:    leadID,
			Action:    domain.LeadActionReportReadyToDownload,
			CreatedBy: userID,
			CreatedOn: now,
		}
		return tx.Create(&h).Error
	})
	return updated, err
}

func (r *leadRepository) UpdateLeadReportApproval(leadID int64, lastUpdatedBy int64, isFit int8, allowDownload bool, isFitCertificateTobeGenerated bool, remarks *string, brandID *int64) (int64, error) {
	now := time.Now().UTC()
	var ar sql.NullString
	if remarks != nil {
		s := *remarks
		if s != "" {
			ar = sql.NullString{String: s, Valid: true}
		}
	}
	updates := map[string]interface{}{
		"IsFit":                         isFit,
		"IsReportDownloadable":          allowDownload,
		"IsFitCertificateTobeGenerated": isFitCertificateTobeGenerated,
		"ApprovalRemarks":               ar,
		"FitUpdatedOn":                  now,
		"LastUpdatedBy":                 lastUpdatedBy,
		"LastUpdatedOn":                 now,
		"LeadStatusID": gorm.Expr("CASE WHEN ? = ? THEN ? ELSE LeadStatusID END",
			isFit, domain.LeadFitFit, domain.LeadStatusIDReportApproved),
	}
	if brandID != nil {
		updates["BrandID"] = *brandID
	}
	res := gormLead(r.db).Model(&persistencemodels.Lead{}).Where("LeadID = ? AND LeadStatusID = ?", leadID, domain.LeadStatusIDReportApproval).Updates(updates)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
