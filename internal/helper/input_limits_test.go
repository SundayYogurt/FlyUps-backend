package helper

import (
	"flyup/internal/dto"
	"math"
	"strings"
	"testing"
)

func TestInputLimits(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   any
		invalid bool
	}{
		{"thai title boundary", dto.UpdateMilestoneRequest{Title: stringPointer(strings.Repeat("ก", 50))}, false},
		{"thai title too long", dto.UpdateMilestoneRequest{Title: stringPointer(strings.Repeat("ก", 51))}, true},
		{"patch omitted", dto.UpdateProjectRequest{}, false},
		{"project description", dto.UpdateProjectRequest{Description: stringPointer(strings.Repeat("a", 41))}, true},
		{"infinity", math.Inf(1), true},
		{"negative infinity", math.Inf(-1), true},
		{"nan", math.NaN(), true},
		{"money boundary", MaxMoney, false},
		{"money overflow", MaxMoney + 1, true},
		{"nested nan", []dto.CreateInvestmentRequest{{Amount: math.NaN()}}, true},
		{"attachment count", dto.SubmitMilestoneRequest{Attachments: make([]string, 6)}, true},
		{"unsafe link", dto.SubmitMilestoneRequest{Links: []string{"javascript:alert(1)"}}, true},
		{"valid link", dto.SubmitMilestoneRequest{Links: []string{"https://example.com/evidence"}}, false},
		{"phase too high", dto.CreateMilestoneRequest{PhaseNo: 5}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateInputLimits(tc.input); (err != nil) != tc.invalid {
				t.Fatalf("error = %v, want invalid %v", err, tc.invalid)
			}
		})
	}
}

func stringPointer(s string) *string { return &s }

func TestComplaintResolutionNoteBoundaries(t *testing.T) {
	for _, tc := range []struct {
		note  string
		valid bool
	}{
		{"   ", false}, {" กก ", false}, {"กกก", true}, {strings.Repeat("ก", 2000), true}, {strings.Repeat("ก", 2001), false},
	} {
		if valid := ValidateInputLimits(dto.ResolveComplaintRequest{AdminNote: tc.note}) == nil; valid != tc.valid {
			t.Errorf("note length %d: valid=%v want %v", len(tc.note), valid, tc.valid)
		}
	}
}
