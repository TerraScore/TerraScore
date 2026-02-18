package job

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

const (
	offerTimeout   = 30 * time.Minute
	maxRounds      = 3
	agentsPerRound = 5
)

// Dispatcher handles cascade dispatch of job offers to ranked agents.
type Dispatcher struct {
	matcher  *Matcher
	jobRepo  *Repository
	rdb      *redis.Client
	eventBus *platform.EventBus
	logger   *slog.Logger
}

// NewDispatcher creates a cascade dispatcher.
func NewDispatcher(matcher *Matcher, jobRepo *Repository, rdb *redis.Client, eventBus *platform.EventBus, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		matcher:  matcher,
		jobRepo:  jobRepo,
		rdb:      rdb,
		eventBus: eventBus,
		logger:   logger,
	}
}

// HandleJobCreated is the EventBus handler for "job.created" events.
// It spawns a goroutine to run the cascade dispatch for the job.
func (d *Dispatcher) HandleJobCreated(ctx context.Context, event platform.Event) {
	job, ok := event.Payload.(*sqlc.SurveyJob)
	if !ok {
		d.logger.Error("dispatcher: invalid event payload type")
		return
	}

	// Spawn cascade goroutine per job
	go d.dispatchJob(ctx, job)
}

// dispatchJob runs the full cascade dispatch flow for a single job.
func (d *Dispatcher) dispatchJob(ctx context.Context, job *sqlc.SurveyJob) {
	d.logger.Info("dispatcher: starting cascade", "job_id", job.ID)

	// We need the parcel centroid to find nearby agents.
	// The parcel boundary centroid is stored in parcels.centroid (PostGIS computed column).
	// For simplicity, we'll pass lng=0, lat=0 as a fallback.
	// In production, we'd query the parcel to get its centroid.
	// For now, get parcel centroid from the parcel table via a simple query.
	lng, lat, err := d.getParcelCentroid(ctx, job.ParcelID)
	if err != nil {
		d.logger.Error("dispatcher: failed to get parcel centroid",
			"job_id", job.ID,
			"parcel_id", job.ParcelID,
			"error", err,
		)
		d.markUnassigned(ctx, job.ID)
		return
	}

	excludeIDs := []uuid.UUID{}
	totalOffersSent := int32(0)

	for round := int32(1); round <= maxRounds; round++ {
		candidates, err := d.matcher.FindCandidatesAtLocation(ctx, lng, lat, job.SurveyType, excludeIDs)
		if err != nil {
			d.logger.Error("dispatcher: matching failed",
				"job_id", job.ID,
				"round", round,
				"error", err,
			)
			continue
		}

		if len(candidates) == 0 {
			d.logger.Info("dispatcher: no candidates in round",
				"job_id", job.ID,
				"round", round,
			)
			continue
		}

		for rank, candidate := range candidates {
			// Create offer
			distKm := float32(candidate.DistanceKm)
			matchScore := pgtype.Numeric{}
			matchScore.Scan(fmt.Sprintf("%.4f", candidate.CompositeScore))

			offer, err := d.jobRepo.CreateOffer(ctx, sqlc.CreateJobOfferParams{
				JobID:        job.ID,
				AgentID:      candidate.AgentID,
				CascadeRound: round,
				OfferRank:    int32(rank + 1),
				DistanceKm:   &distKm,
				MatchScore:   matchScore,
				ExpiresAt:    time.Now().Add(offerTimeout),
			})
			if err != nil {
				d.logger.Error("dispatcher: failed to create offer",
					"job_id", job.ID,
					"agent_id", candidate.AgentID,
					"error", err,
				)
				continue
			}

			totalOffersSent++

			// Update job status to offered
			d.jobRepo.UpdateJobStatus(ctx, job.ID, "offered")

			// Publish to Redis for real-time notification to agent
			d.publishOfferToAgent(ctx, candidate.AgentID, offer)

			// FCM push notification placeholder (Phase 1: log only)
			d.logger.Info("dispatcher: FCM push placeholder",
				"agent_id", candidate.AgentID,
				"job_id", job.ID,
				"offer_id", offer.ID,
			)

			// Wait for agent response via Redis pub/sub
			accepted := d.waitForResponse(ctx, offer.ID)

			if accepted {
				// Assign agent to job
				_, err := d.jobRepo.AssignAgent(ctx, sqlc.AssignAgentParams{
					ID:              job.ID,
					AssignedAgentID: pgtype.UUID{Bytes: candidate.AgentID, Valid: true},
					CascadeRound:    &round,
					TotalOffersSent: &totalOffersSent,
				})
				if err != nil {
					d.logger.Error("dispatcher: failed to assign agent",
						"job_id", job.ID,
						"agent_id", candidate.AgentID,
						"error", err,
					)
					continue
				}

				d.eventBus.Publish(platform.Event{
					Type:    "job.assigned",
					Payload: job,
				})

				d.logger.Info("dispatcher: job assigned",
					"job_id", job.ID,
					"agent_id", candidate.AgentID,
					"round", round,
					"rank", rank+1,
				)
				return
			}

			// Agent declined or timed out; try next candidate
			excludeIDs = append(excludeIDs, candidate.AgentID)
		}
	}

	// All rounds exhausted
	d.markUnassigned(ctx, job.ID)
	d.logger.Warn("dispatcher: all rounds exhausted, job unassigned", "job_id", job.ID)
}

// waitForResponse waits for an agent's accept/decline on a Redis channel.
// Returns true if accepted, false if declined or timed out.
func (d *Dispatcher) waitForResponse(ctx context.Context, offerID uuid.UUID) bool {
	channel := fmt.Sprintf("offer:%s:response", offerID)

	ctx, cancel := context.WithTimeout(ctx, offerTimeout)
	defer cancel()

	sub := d.rdb.Subscribe(ctx, channel)
	defer sub.Close()

	ch := sub.Channel()
	select {
	case msg := <-ch:
		if msg == nil {
			return false
		}
		d.logger.Info("dispatcher: received response",
			"offer_id", offerID,
			"response", msg.Payload,
		)
		return msg.Payload == "accepted"
	case <-ctx.Done():
		// Timeout — expire the offer
		d.logger.Info("dispatcher: offer timed out", "offer_id", offerID)
		expired := "expired"
		d.jobRepo.UpdateOfferStatus(ctx, offerID, expired, nil)
		return false
	}
}

// publishOfferToAgent sends an offer notification to the agent's Redis channel.
func (d *Dispatcher) publishOfferToAgent(ctx context.Context, agentID uuid.UUID, offer *sqlc.JobOffer) {
	channel := fmt.Sprintf("agent:%s:offers", agentID)
	payload, _ := json.Marshal(map[string]interface{}{
		"offer_id":   offer.ID,
		"job_id":     offer.JobID,
		"expires_at": offer.ExpiresAt,
	})
	d.rdb.Publish(ctx, channel, payload)
}

// getParcelCentroid fetches the centroid coordinates for a parcel.
// Uses a direct query since we need centroid as lng/lat.
func (d *Dispatcher) getParcelCentroid(ctx context.Context, parcelID uuid.UUID) (lng, lat float64, err error) {
	var lngVal, latVal *float64
	row := d.jobRepo.db.QueryRow(ctx,
		"SELECT ST_X(centroid), ST_Y(centroid) FROM parcels WHERE id = $1",
		parcelID,
	)
	err = row.Scan(&lngVal, &latVal)
	if err != nil {
		return 0, 0, fmt.Errorf("getting parcel centroid: %w", err)
	}
	if lngVal == nil || latVal == nil {
		return 0, 0, fmt.Errorf("parcel %s has no centroid", parcelID)
	}
	return *lngVal, *latVal, nil
}

// markUnassigned sets job status to unassigned.
func (d *Dispatcher) markUnassigned(ctx context.Context, jobID uuid.UUID) {
	_, err := d.jobRepo.UpdateJobStatus(ctx, jobID, "unassigned")
	if err != nil {
		d.logger.Error("dispatcher: failed to mark unassigned", "job_id", jobID, "error", err)
	}
}
