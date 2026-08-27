package repository

import (
	"database/sql"
	"encoding/json"
	"testing"

	persistencemodels "b2b-diagnostic-aggregator/apis/internal/persistence/models"
)

func TestMapLeadToDomain_StoreFieldsOnlyOnMedLyfe(t *testing.T) {
	t.Cleanup(func() {
		persistencemodels.SetSchema(persistencemodels.DefaultSchema)
	})

	persist := persistencemodels.Lead{LeadID: 1, PatientName: "Pat"}
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

	persistencemodels.SetSchema("MediAdmin")
	legacy := mapLeadToDomainWithOptionalJoinedNames(persist, names)
	if legacy.StoreName != "" || legacy.StoreCity != "" {
		t.Errorf("non-MedLyfe store fields = (%q, %q), want empty", legacy.StoreName, legacy.StoreCity)
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
}
