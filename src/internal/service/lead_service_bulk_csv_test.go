package service

import (
	"testing"

	"b2b-diagnostic-aggregator/apis/internal/domain"
	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
	"b2b-diagnostic-aggregator/apis/internal/repository"

	"gorm.io/gorm"
)

type mockLeadRepoForBulkCSV struct {
	repository.LeadRepository
	created []*domain.Lead
}

func (m *mockLeadRepoForBulkCSV) Create(lead *domain.Lead) error {
	cp := *lead
	cp.LeadID = int64(len(m.created) + 1)
	lead.LeadID = cp.LeadID
	m.created = append(m.created, &cp)
	return nil
}

type mockClientRepoForBulkCSV struct {
	repository.ClientRepository
	clients map[int64]*domain.Client
}

func (m *mockClientRepoForBulkCSV) FindByID(id int64) (*domain.Client, error) {
	if c, ok := m.clients[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}

type mockStoreRepoForBulkCSV struct {
	repository.StoreRepository
	stores map[int64]*domain.Store
}

func (m *mockStoreRepoForBulkCSV) FindByID(id int64) (*domain.Store, error) {
	if s, ok := m.stores[id]; ok {
		return s, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockStoreRepoForBulkCSV) List(filter repository.StoreListFilter) ([]domain.Store, int64, error) {
	var out []domain.Store
	for _, s := range m.stores {
		if filter.ClientID != nil && s.ClientID != *filter.ClientID {
			continue
		}
		if filter.IsActive != nil && s.IsActive != *filter.IsActive {
			continue
		}
		out = append(out, *s)
	}
	return out, int64(len(out)), nil
}

func TestBulkImportFromCSV_MedLyfeSchema_EmpIDAndEmploymentTypeOnly(t *testing.T) {
	prevSchema := persistencemodels.Schema()
	t.Cleanup(func() { persistencemodels.SetSchema(prevSchema) })

	for _, schema := range []string{"MedLyfe", "MedLyfeDB"} {
		t.Run(schema, func(t *testing.T) {
			persistencemodels.SetSchema(schema)

			leadRepo := &mockLeadRepoForBulkCSV{}
			histRepo := &mockLeadHistoryRepoForUpload{}
			uow := &mockLeadUowForUpload{leadRepo: leadRepo, historyRepo: histRepo}
			svc := &leadService{repo: leadRepo, uow: uow}

			csvData := "\uFEFFPatientName,Age,Gender,CollectionType,ContactNumber,Emailid,Address,CityID,StateID,Pincode,EmpID,EmploymentType,StoreID\n" +
				"Rahul Sharma,28,M,Home,9876543210,rahul@example.com,221B Baker Street,10,1,400001,EMP001,New,STORE-SHOULD-IGNORE\n" +
				"Anita Verma,32,F,Center,9123456789,anita@example.com,14 MG Road,11,1,500001,,Existing,STORE-999\n"

			inserted, err := svc.BulkImportFromCSV([]byte(csvData), 101, 5, nil, 1)
			if err != nil {
				t.Fatalf("unexpected error for schema %s: %v", schema, err)
			}
			if inserted != 2 || len(leadRepo.created) != 2 {
				t.Fatalf("expected 2 inserted rows, got inserted=%d len=%d", inserted, len(leadRepo.created))
			}

			r1 := leadRepo.created[0]
			if r1.EmpID != "EMP001" {
				t.Errorf("row 1 EmpID = %q, want EMP001", r1.EmpID)
			}
			if r1.EmploymentTypeID == nil || *r1.EmploymentTypeID != 1 {
				t.Errorf("row 1 EmploymentTypeID = %v, want 1", r1.EmploymentTypeID)
			}
			if r1.StoreID != "" {
				t.Errorf("row 1 StoreID = %q, want empty on %s", r1.StoreID, schema)
			}

			r2 := leadRepo.created[1]
			if r2.EmpID != "" {
				t.Errorf("row 2 optional blank EmpID = %q, want empty", r2.EmpID)
			}
			if r2.EmploymentTypeID == nil || *r2.EmploymentTypeID != 2 {
				t.Errorf("row 2 EmploymentTypeID = %v, want 2", r2.EmploymentTypeID)
			}
			if r2.StoreID != "" {
				t.Errorf("row 2 StoreID = %q, want empty on %s", r2.StoreID, schema)
			}
		})
	}
}

func TestBulkImportFromCSV_MedLyfeSchema_StoreMasterIDMappingAndValidation(t *testing.T) {
	prevSchema := persistencemodels.Schema()
	t.Cleanup(func() { persistencemodels.SetSchema(prevSchema) })

	for _, schema := range []string{"MedLyfe", "MedLyfeDB"} {
		t.Run(schema, func(t *testing.T) {
			persistencemodels.SetSchema(schema)

			clientRepo := &mockClientRepoForBulkCSV{
				clients: map[int64]*domain.Client{
					101: {ClientID: 101, ClientName: "Client With Stores", IsStoreLoginEnabled: true, IsAcitve: true},
					102: {ClientID: 102, ClientName: "Client Without Stores", IsStoreLoginEnabled: false, IsAcitve: true},
				},
			}
			storeRepo := &mockStoreRepoForBulkCSV{
				stores: map[int64]*domain.Store{
					55: {StoreID: 55, ClientID: 101, StoreName: "Chennai Main Store", IsActive: true},
					99: {StoreID: 99, ClientID: 999, StoreName: "Other Client Store", IsActive: true},
				},
			}

			csvData := "PatientName,Age,Gender,CollectionType,ContactNumber,Emailid,Address,CityID,StateID,Pincode,EmpID,EmploymentType\n" +
				"Rahul Sharma,28,M,Home,9876543210,rahul@example.com,221B Baker Street,10,1,400001,EMP001,New\n" +
				"Anita Verma,32,F,Center,9123456789,anita@example.com,14 MG Road,11,1,500001,EMP002,Existing\n"

			// 1. Client 101 has active stores: omitting StoreMasterID must fail.
			leadRepo1 := &mockLeadRepoForBulkCSV{}
			svc1 := &leadService{
				repo:       leadRepo1,
				uow:        &mockLeadUowForUpload{leadRepo: leadRepo1, historyRepo: &mockLeadHistoryRepoForUpload{}},
				clientRepo: clientRepo,
				storeRepo:  storeRepo,
			}
			if _, err := svc1.BulkImportFromCSV([]byte(csvData), 101, 5, nil, 1); err == nil {
				t.Fatalf("expected error when client has stores and StoreMasterID is omitted")
			}

			// 2. Store belonging to a different client must fail.
			wrongStore := int64(99)
			if _, err := svc1.BulkImportFromCSV([]byte(csvData), 101, 5, &wrongStore, 1); err == nil {
				t.Fatalf("expected error when StoreMasterID belongs to another client")
			}

			// 3. Valid StoreMasterID maps onto ALL imported leads.
			validStore := int64(55)
			inserted, err := svc1.BulkImportFromCSV([]byte(csvData), 101, 5, &validStore, 1)
			if err != nil {
				t.Fatalf("unexpected error with valid StoreMasterID: %v", err)
			}
			if inserted != 2 || len(leadRepo1.created) != 2 {
				t.Fatalf("expected 2 leads created, got %d", len(leadRepo1.created))
			}
			for idx, created := range leadRepo1.created {
				if created.StoreMasterID == nil || *created.StoreMasterID != 55 {
					t.Errorf("row %d StoreMasterID = %v, want 55", idx+1, created.StoreMasterID)
				}
			}

			// 4. Client 102 has no stores: succeeds without StoreMasterID (NULL).
			leadRepo2 := &mockLeadRepoForBulkCSV{}
			svc2 := &leadService{
				repo:       leadRepo2,
				uow:        &mockLeadUowForUpload{leadRepo: leadRepo2, historyRepo: &mockLeadHistoryRepoForUpload{}},
				clientRepo: clientRepo,
				storeRepo:  storeRepo,
			}
			inserted2, err2 := svc2.BulkImportFromCSV([]byte(csvData), 102, 5, nil, 1)
			if err2 != nil {
				t.Fatalf("unexpected error for client without stores: %v", err2)
			}
			if inserted2 != 2 || leadRepo2.created[0].StoreMasterID != nil {
				t.Errorf("expected nil StoreMasterID for client without stores, got %v", leadRepo2.created[0].StoreMasterID)
			}
		})
	}
}

func TestBulkImportFromCSV_NonMedLyfeSchema_EmpIDAndStoreIDOnly(t *testing.T) {
	prevSchema := persistencemodels.Schema()
	t.Cleanup(func() { persistencemodels.SetSchema(prevSchema) })
	persistencemodels.SetSchema("MediAdmin")

	leadRepo := &mockLeadRepoForBulkCSV{}
	histRepo := &mockLeadHistoryRepoForUpload{}
	uow := &mockLeadUowForUpload{leadRepo: leadRepo, historyRepo: histRepo}
	svc := &leadService{repo: leadRepo, uow: uow}

	csvData := "PatientName,Age,Gender,CollectionType,ContactNumber,Emailid,Address,CityID,StateID,Pincode,EmpID,StoreID,EmploymentType\n" +
		"Rahul Sharma,28,M,Home,9876543210,rahul@example.com,221B Baker Street,10,1,400001,EMP100,STORE-001,New\n" +
		"Anita Verma,32,F,Center,9123456789,anita@example.com,14 MG Road,11,1,500001,,,\n"

	ignoredStoreMaster := int64(55)
	inserted, err := svc.BulkImportFromCSV([]byte(csvData), 101, 5, &ignoredStoreMaster, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inserted != 2 || len(leadRepo.created) != 2 {
		t.Fatalf("expected 2 inserted rows, got inserted=%d len=%d", inserted, len(leadRepo.created))
	}

	r1 := leadRepo.created[0]
	if r1.EmpID != "EMP100" {
		t.Errorf("row 1 EmpID = %q, want EMP100", r1.EmpID)
	}
	if r1.StoreID != "STORE-001" {
		t.Errorf("row 1 StoreID = %q, want STORE-001", r1.StoreID)
	}
	if r1.StoreMasterID != nil {
		t.Errorf("row 1 StoreMasterID = %v, want nil on non-MedLyfe", r1.StoreMasterID)
	}
	if r1.EmploymentTypeID != nil {
		t.Errorf("row 1 EmploymentTypeID = %v, want nil on non-MedLyfe", r1.EmploymentTypeID)
	}

	r2 := leadRepo.created[1]
	if r2.EmpID != "" || r2.StoreID != "" || r2.StoreMasterID != nil || r2.EmploymentTypeID != nil {
		t.Errorf("row 2 optional blank fields = EmpID:%q StoreID:%q StoreMasterID:%v EmploymentTypeID:%v", r2.EmpID, r2.StoreID, r2.StoreMasterID, r2.EmploymentTypeID)
	}
}
