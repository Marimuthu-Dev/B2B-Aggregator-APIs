package repository

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func TestMapLeadToDomain_StoreFieldsOnlyOnMedLyfe(t *testing.T) {
	t.Cleanup(func() {
		persistencemodels.SetSchema(persistencemodels.DefaultSchema)
	})

	reqTime := time.Date(2026, 11, 25, 10, 30, 0, 0, time.UTC)
	persist := persistencemodels.Lead{LeadID: 1, PatientName: "Pat", RequestedAppointmentAt: &reqTime}
	names := leadJoinedNames{
		StoreName: sql.NullString{String: "Downtown Store", Valid: true},
		StoreCity: sql.NullString{String: "Chennai", Valid: true},
	}

	persistencemodels.SetSchema(persistencemodels.MedLyfeSchema)
	med := mapLeadToDomainWithOptionalJoinedNames(persist, names)
	if med.StoreName != "Downtown Store" {
		t.Errorf("MedLyfe StoreName = %q, want Downtown Store", med.StoreName)
	}
	if med.StoreCity != "Chennai" {
		t.Errorf("MedLyfe StoreCity = %q, want Chennai", med.StoreCity)
	}
	if med.RequestedAppointmentAt == nil {
		t.Errorf("MedLyfe RequestedAppointmentAt = nil, want non-nil")
	}

	persistencemodels.SetSchema("MediAdmin")
	legacy := mapLeadToDomainWithOptionalJoinedNames(persist, names)
	if legacy.StoreName != "" || legacy.StoreCity != "" {
		t.Errorf("non-MedLyfe store fields = (%q, %q), want empty", legacy.StoreName, legacy.StoreCity)
	}
	if legacy.RequestedAppointmentAt != nil {
		t.Errorf("non-MedLyfe RequestedAppointmentAt = %v, want nil", legacy.RequestedAppointmentAt)
	}

	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if _, ok := obj["StoreName"]; ok {
		t.Error("StoreName must be omitted when DB schema is not MedLyfe")
	}
	if _, ok := obj["StoreCity"]; ok {
		t.Error("StoreCity must be omitted when DB schema is not MedLyfe")
	}
	if _, ok := obj["RequestedAppointmentAt"]; ok {
		t.Error("RequestedAppointmentAt must be omitted when DB schema is not MedLyfe")
	}
}
