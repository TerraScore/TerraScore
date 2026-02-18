package land

import (
	"encoding/json"
	"fmt"

	"github.com/terrascore/api/internal/platform"
)

// GeoJSON types for boundary validation.
type geoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// India bounding box: [68.0, 6.0] - [97.5, 37.5]
const (
	indiaBBoxMinLng = 68.0
	indiaBBoxMinLat = 6.0
	indiaBBoxMaxLng = 97.5
	indiaBBoxMaxLat = 37.5
)

// ValidateBoundaryGeoJSON checks that the GeoJSON string is a valid Polygon within India.
func ValidateBoundaryGeoJSON(geoJSONStr string) error {
	if geoJSONStr == "" {
		return platform.NewValidation("boundary is required")
	}

	var geom geoJSONGeometry
	if err := json.Unmarshal([]byte(geoJSONStr), &geom); err != nil {
		return platform.NewValidation(fmt.Sprintf("invalid GeoJSON: %s", err.Error()))
	}

	if geom.Type != "Polygon" {
		return platform.NewValidation("boundary must be a GeoJSON Polygon")
	}

	// Parse coordinates: Polygon is [][][2]float64 (array of rings, each ring is array of [lng, lat])
	var rings [][][2]float64
	if err := json.Unmarshal(geom.Coordinates, &rings); err != nil {
		return platform.NewValidation(fmt.Sprintf("invalid Polygon coordinates: %s", err.Error()))
	}

	if len(rings) == 0 {
		return platform.NewValidation("polygon must have at least one ring")
	}

	outerRing := rings[0]
	// A valid polygon ring needs at least 4 points (3 unique + closing point)
	if len(outerRing) < 4 {
		return platform.NewValidation("polygon must have at least 3 coordinate points (4 including closing point)")
	}

	// Validate all coordinates are within India bbox
	for _, coord := range outerRing {
		lng, lat := coord[0], coord[1]
		if lng < indiaBBoxMinLng || lng > indiaBBoxMaxLng || lat < indiaBBoxMinLat || lat > indiaBBoxMaxLat {
			return platform.NewValidation(fmt.Sprintf(
				"coordinates [%.4f, %.4f] are outside India bounding box [%.1f,%.1f]-[%.1f,%.1f]",
				lng, lat, indiaBBoxMinLng, indiaBBoxMinLat, indiaBBoxMaxLng, indiaBBoxMaxLat,
			))
		}
	}

	return nil
}
