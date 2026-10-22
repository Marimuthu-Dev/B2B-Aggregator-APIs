package domain

import (
	"testing"
)

func TestFitnessStatusFromLead(t *testing.T) {
	fit := LeadFitFit
	unfit := LeadFitUnfit
	legacyUnfit := int8(-1)
	hold := LeadFitHold

	tests := []struct {
		name         string
		isFit        *int8
		leadStatusID int8
		want         string
	}{
		{"Fit status", &fit, 9, "Fit"},
		{"Fit with uploaded status", &fit, 8, "Fit"},
		{"Fit with ready status", &fit, 10, "Fit"},
		{"Unfit status", &unfit, 8, "UnFit"},
		{"Legacy unfit (-1)", &legacyUnfit, 8, "UnFit"},
		{"Hold after report uploaded", &hold, 8, "On Hold"},
		{"Hold at status 9", &hold, 9, "On Hold"},
		{"Hold before report uploaded", &hold, 4, "Yet to Reported"},
		{"Nil before report uploaded (status 1)", nil, 1, "Yet to Reported"},
		{"Nil before report uploaded (status 4)", nil, 4, "Yet to Reported"},
		{"Nil before report uploaded (status 6)", nil, 6, "Yet to Reported"},
		{"Nil at report uploaded (status 8)", nil, 8, "On Hold"},
		{"Nil at report approved (status 9)", nil, 9, "On Hold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FitnessStatusFromLead(tt.isFit, tt.leadStatusID)
			if got != tt.want {
				t.Errorf("FitnessStatusFromLead(%v, %d) = %q; want %q", tt.isFit, tt.leadStatusID, got, tt.want)
			}
		})
	}
}

func TestParseLeadListFitnessFilter(t *testing.T) {
	tests := []struct {
		input string
		want  LeadListFitnessFilter
	}{
		{"", LeadListFitnessFilterNone},
		{"   ", LeadListFitnessFilterNone},
		{"fit", LeadListFitnessFilterFit},
		{"Fit", LeadListFitnessFilterFit},
		{"unfit", LeadListFitnessFilterUnfit},
		{"UnFit", LeadListFitnessFilterUnfit},
		{"on hold", LeadListFitnessFilterOnHold},
		{"On Hold", LeadListFitnessFilterOnHold},
		{"hold", LeadListFitnessFilterOnHold},
		{"yet to report", LeadListFitnessFilterYetToReport},
		{"Yet to Reported", LeadListFitnessFilterYetToReport},
		{"yettoreport", LeadListFitnessFilterYetToReport},
		{"yettoreported", LeadListFitnessFilterYetToReport},
		{"Empty", LeadListFitnessFilterEmpty},
		{"Not Assessed", LeadListFitnessFilterEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLeadListFitnessFilter(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseLeadListFitnessFilter(%q) = %d; want %d", tt.input, got, tt.want)
			}
		})
	}
}
