package job

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/terrascore/api/db/sqlc"
)

// Candidate represents a scored agent candidate for a job.
type Candidate struct {
	AgentID        uuid.UUID
	DistanceKm     float64
	AvgRating      float64
	CompletionRate float64
	HoursSinceJob  float64
	Tier           string
	CompositeScore float64
}

// Matcher finds and scores nearby agents for survey jobs.
type Matcher struct {
	agentQ *sqlc.Queries // direct sqlc access for FindMatchableAgents
	jobRepo *Repository
	logger  *slog.Logger
}

// NewMatcher creates a matcher with agent query access and job repository.
func NewMatcher(agentQ *sqlc.Queries, jobRepo *Repository, logger *slog.Logger) *Matcher {
	return &Matcher{
		agentQ:  agentQ,
		jobRepo: jobRepo,
		logger:  logger,
	}
}

// Scoring weights.
const (
	weightDistance   = 0.40
	weightRating    = 0.30
	weightCompletion = 0.20
	weightFreshness = 0.10

	maxCandidates    = 5
	maxConcurrentJobs = 3
)

// Expansion radii in meters for PostGIS ST_DWithin.
var expansionRadii = []float64{25000, 50000, 100000} // 25km, 50km, 100km

// tierMinimum maps survey types to minimum agent tiers.
var tierMinimum = map[string][]string{
	"basic_check":       {"basic", "experienced", "senior"},
	"detailed_survey":   {"experienced", "senior"},
	"premium_inspection": {"senior"},
}

// FindCandidatesAtLocation finds candidates near a given lng/lat for a survey type.
func (m *Matcher) FindCandidatesAtLocation(ctx context.Context, lng, lat float64, surveyType string, excludeIDs []uuid.UUID) ([]Candidate, error) {
	allowedTiers := tierMinimum[surveyType]
	if allowedTiers == nil {
		allowedTiers = tierMinimum["basic_check"]
	}

	if excludeIDs == nil {
		excludeIDs = []uuid.UUID{}
	}

	for _, radiusM := range expansionRadii {
		agents, err := m.agentQ.FindMatchableAgents(ctx, sqlc.FindMatchableAgentsParams{
			StMakepoint:   lng,
			StMakepoint_2: lat,
			StDwithin:     radiusM,
			Limit:         50, // fetch more than needed, filter/score below
			Column5:       excludeIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("finding matchable agents at radius %.0fm: %w", radiusM, err)
		}

		if len(agents) == 0 {
			m.logger.Debug("no agents found at radius", "radius_m", radiusM)
			continue
		}

		candidates, err := m.scoreAndFilter(ctx, agents, allowedTiers, radiusM/1000)
		if err != nil {
			return nil, err
		}

		if len(candidates) > 0 {
			m.logger.Info("found candidates",
				"count", len(candidates),
				"radius_m", radiusM,
				"survey_type", surveyType,
			)
			return candidates, nil
		}
	}

	m.logger.Warn("no candidates found after all expansion rounds")
	return nil, nil
}

// scoreAndFilter filters by tier and concurrent load, then scores and ranks agents.
func (m *Matcher) scoreAndFilter(ctx context.Context, agents []sqlc.FindMatchableAgentsRow, allowedTiers []string, maxDistKm float64) ([]Candidate, error) {
	var candidates []Candidate

	for _, a := range agents {
		// Filter by tier
		agentTier := "basic"
		if a.Tier != nil {
			agentTier = *a.Tier
		}
		if !tierAllowed(agentTier, allowedTiers) {
			continue
		}

		// Filter by concurrent load
		activeCount, err := m.jobRepo.CountActiveJobsByAgent(ctx, a.ID)
		if err != nil {
			m.logger.Error("failed to count active jobs", "agent_id", a.ID, "error", err)
			continue
		}
		if activeCount >= maxConcurrentJobs {
			continue
		}

		// Extract numeric values
		distKm := float64(a.DistanceKm)
		avgRating := numericToFloat64(a.AvgRating)
		completionRate := numericToFloat64(a.CompletionRate)
		hoursSinceJob := hoursSince(a.LastJobCompletedAt)

		// Compute scores
		distScore := (1.0 - distKm/maxDistKm) * weightDistance
		if distScore < 0 {
			distScore = 0
		}
		ratingScore := (avgRating / 5.0) * weightRating
		completionScore := completionRate * weightCompletion
		freshnessScore := math.Min(hoursSinceJob/48.0, 1.0) * weightFreshness

		composite := distScore + ratingScore + completionScore + freshnessScore

		candidates = append(candidates, Candidate{
			AgentID:        a.ID,
			DistanceKm:     distKm,
			AvgRating:      avgRating,
			CompletionRate: completionRate,
			HoursSinceJob:  hoursSinceJob,
			Tier:           agentTier,
			CompositeScore: composite,
		})
	}

	// Sort by composite score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CompositeScore > candidates[j].CompositeScore
	})

	// Return top N
	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	return candidates, nil
}

// tierAllowed checks if agentTier is in the allowed list.
func tierAllowed(agentTier string, allowed []string) bool {
	for _, t := range allowed {
		if agentTier == t {
			return true
		}
	}
	return false
}

// numericToFloat64 converts pgtype.Numeric to float64 (defaults to 0).
func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f, _ := n.Float64Value()
	if !f.Valid {
		return 0
	}
	return f.Float64
}

// hoursSince returns hours elapsed since a timestamp (defaults to 48 if invalid).
func hoursSince(ts pgtype.Timestamptz) float64 {
	if !ts.Valid {
		return 48 // Default: treat as "long ago" for freshness score
	}
	return time.Since(ts.Time).Hours()
}
