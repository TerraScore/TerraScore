package job

import (
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestTierAllowed(t *testing.T) {
	tests := []struct {
		name     string
		tier     string
		allowed  []string
		expected bool
	}{
		{"basic allowed for basic_check", "basic", []string{"basic", "experienced", "senior"}, true},
		{"experienced allowed for basic_check", "experienced", []string{"basic", "experienced", "senior"}, true},
		{"basic not allowed for detailed_survey", "basic", []string{"experienced", "senior"}, false},
		{"experienced allowed for detailed_survey", "experienced", []string{"experienced", "senior"}, true},
		{"basic not allowed for premium", "basic", []string{"senior"}, false},
		{"experienced not allowed for premium", "experienced", []string{"senior"}, false},
		{"senior allowed for premium", "senior", []string{"senior"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tierAllowed(tt.tier, tt.allowed)
			if result != tt.expected {
				t.Errorf("tierAllowed(%q, %v) = %v, want %v", tt.tier, tt.allowed, result, tt.expected)
			}
		})
	}
}

func TestNumericToFloat64(t *testing.T) {
	// Zero/invalid
	n := pgtype.Numeric{}
	if got := numericToFloat64(n); got != 0 {
		t.Errorf("invalid numeric = %f, want 0", got)
	}

	// Valid numeric
	n2 := pgtype.Numeric{}
	n2.Scan("4.5")
	got := numericToFloat64(n2)
	if math.Abs(got-4.5) > 0.01 {
		t.Errorf("numeric(4.5) = %f, want 4.5", got)
	}
}

func TestHoursSince(t *testing.T) {
	// Invalid timestamp should return 48
	ts := pgtype.Timestamptz{}
	if got := hoursSince(ts); got != 48 {
		t.Errorf("invalid timestamp = %f, want 48", got)
	}
}

func TestScoringWeights(t *testing.T) {
	// Verify weights sum to 1.0
	total := weightDistance + weightRating + weightCompletion + weightFreshness
	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("scoring weights sum = %f, want 1.0", total)
	}
}

func TestScoringFormula(t *testing.T) {
	// Perfect agent: 0km distance, 5.0 rating, 100% completion, 48h since last job
	distScore := (1.0 - 0.0/25.0) * weightDistance      // 0.40
	ratingScore := (5.0 / 5.0) * weightRating            // 0.30
	completionScore := 1.0 * weightCompletion             // 0.20
	freshnessScore := math.Min(48.0/48.0, 1.0) * weightFreshness // 0.10
	composite := distScore + ratingScore + completionScore + freshnessScore

	if math.Abs(composite-1.0) > 0.001 {
		t.Errorf("perfect score = %f, want 1.0", composite)
	}

	// Agent at max distance: 25km away, 2.5 rating, 50% completion, 0h since last job
	distScore2 := (1.0 - 25.0/25.0) * weightDistance      // 0.0
	ratingScore2 := (2.5 / 5.0) * weightRating            // 0.15
	completionScore2 := 0.5 * weightCompletion             // 0.10
	freshnessScore2 := math.Min(0.0/48.0, 1.0) * weightFreshness // 0.0
	composite2 := distScore2 + ratingScore2 + completionScore2 + freshnessScore2

	if math.Abs(composite2-0.25) > 0.001 {
		t.Errorf("poor score = %f, want 0.25", composite2)
	}

	// Perfect should be higher than poor
	if composite <= composite2 {
		t.Errorf("perfect (%f) should be > poor (%f)", composite, composite2)
	}
}

func TestTierMinimumMappings(t *testing.T) {
	// basic_check allows all tiers
	if !tierAllowed("basic", tierMinimum["basic_check"]) {
		t.Error("basic_check should allow basic tier")
	}

	// detailed_survey disallows basic
	if tierAllowed("basic", tierMinimum["detailed_survey"]) {
		t.Error("detailed_survey should not allow basic tier")
	}

	// premium_inspection only senior
	if tierAllowed("experienced", tierMinimum["premium_inspection"]) {
		t.Error("premium_inspection should not allow experienced tier")
	}
	if !tierAllowed("senior", tierMinimum["premium_inspection"]) {
		t.Error("premium_inspection should allow senior tier")
	}
}
