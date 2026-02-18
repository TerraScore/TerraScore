package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/terrascore/api/internal/platform"
)

const (
	locationKeyPrefix = "agent:loc:"
	locationTTL       = 5 * time.Minute
	flushInterval     = 5 * time.Minute
)

// LocationData stored in Redis.
type LocationData struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"ts"`
}

// ValidateLocation checks lat/lng/accuracy bounds.
func ValidateLocation(lat, lng, accuracy float64) error {
	if lat < -90 || lat > 90 {
		return platform.NewValidation("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return platform.NewValidation("longitude must be between -180 and 180")
	}
	if accuracy < 0 || accuracy >= 100 {
		return platform.NewValidation("accuracy must be between 0 and 100 meters")
	}
	return nil
}

// UpdateLocationRedis writes agent location to Redis and publishes for real-time clients.
func UpdateLocationRedis(ctx context.Context, rdb *redis.Client, agentID uuid.UUID, lat, lng, accuracy float64) error {
	data := LocationData{
		Lat:       lat,
		Lng:       lng,
		Accuracy:  accuracy,
		Timestamp: time.Now().Unix(),
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling location data: %w", err)
	}

	key := locationKeyPrefix + agentID.String()

	// SETEX with 5 minute TTL
	if err := rdb.Set(ctx, key, jsonBytes, locationTTL).Err(); err != nil {
		return fmt.Errorf("storing location in redis: %w", err)
	}

	// PUBLISH for real-time WebSocket clients
	channel := fmt.Sprintf("agent:%s:location", agentID.String())
	rdb.Publish(ctx, channel, jsonBytes)

	return nil
}

// LocationFlusher periodically flushes agent locations from Redis to PostGIS.
type LocationFlusher struct {
	rdb    *redis.Client
	repo   *Repository
	logger *slog.Logger
}

// NewLocationFlusher creates a new location flusher.
func NewLocationFlusher(rdb *redis.Client, repo *Repository, logger *slog.Logger) *LocationFlusher {
	return &LocationFlusher{
		rdb:    rdb,
		repo:   repo,
		logger: logger,
	}
}

// Start begins the periodic flush loop. Call in a goroutine.
func (lf *LocationFlusher) Start(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	lf.logger.Info("location flusher started", "interval", flushInterval)

	for {
		select {
		case <-ctx.Done():
			lf.logger.Info("location flusher stopped")
			return
		case <-ticker.C:
			lf.flush(ctx)
		}
	}
}

func (lf *LocationFlusher) flush(ctx context.Context) {
	var cursor uint64
	var flushed int

	for {
		keys, nextCursor, err := lf.rdb.Scan(ctx, cursor, locationKeyPrefix+"*", 100).Result()
		if err != nil {
			lf.logger.Error("scanning redis for location keys", "error", err)
			return
		}

		for _, key := range keys {
			val, err := lf.rdb.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			var loc LocationData
			if err := json.Unmarshal([]byte(val), &loc); err != nil {
				lf.logger.Warn("invalid location data in redis", "key", key, "error", err)
				continue
			}

			// Extract agent ID from key: "agent:loc:{uuid}"
			agentIDStr := key[len(locationKeyPrefix):]
			agentID, err := uuid.Parse(agentIDStr)
			if err != nil {
				lf.logger.Warn("invalid agent ID in redis key", "key", key, "error", err)
				continue
			}

			if err := lf.repo.UpdateAgentLocation(ctx, agentID, loc.Lng, loc.Lat); err != nil {
				lf.logger.Error("flushing location to PostGIS", "agent_id", agentID, "error", err)
				continue
			}
			flushed++
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if flushed > 0 {
		lf.logger.Info("flushed agent locations to PostGIS", "count", flushed)
	}
}
