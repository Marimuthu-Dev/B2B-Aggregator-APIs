package models

import "testing"

func TestHasLeadStoreMasterIDColumn(t *testing.T) {
	t.Cleanup(func() {
		SetSchema(DefaultSchema)
	})

	cases := []struct {
		schema string
		want   bool
	}{
		{schema: "MedLyfe", want: true},
		{schema: "medlyfe", want: true},
		{schema: "MediAdmin", want: false},
		{schema: DefaultSchema, want: false},
	}
	for _, tc := range cases {
		SetSchema(tc.schema)
		if got := HasLeadStoreMasterIDColumn(); got != tc.want {
			t.Errorf("schema %q: HasLeadStoreMasterIDColumn() = %v, want %v", tc.schema, got, tc.want)
		}
		if got := HasStoreMasterTable(); got != tc.want {
			t.Errorf("schema %q: HasStoreMasterTable() = %v, want %v", tc.schema, got, tc.want)
		}
	}
}
