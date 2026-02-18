package agent

import (
	"testing"
)

func TestValidateLocation(t *testing.T) {
	tests := []struct {
		name     string
		lat      float64
		lng      float64
		accuracy float64
		wantErr  bool
	}{
		{
			name:     "valid coordinates",
			lat:      12.9716,
			lng:      77.5946,
			accuracy: 10,
			wantErr:  false,
		},
		{
			name:     "latitude too low",
			lat:      -91,
			lng:      77.0,
			accuracy: 10,
			wantErr:  true,
		},
		{
			name:     "latitude too high",
			lat:      91,
			lng:      77.0,
			accuracy: 10,
			wantErr:  true,
		},
		{
			name:     "longitude too low",
			lat:      12.0,
			lng:      -181,
			accuracy: 10,
			wantErr:  true,
		},
		{
			name:     "longitude too high",
			lat:      12.0,
			lng:      181,
			accuracy: 10,
			wantErr:  true,
		},
		{
			name:     "accuracy too high",
			lat:      12.0,
			lng:      77.0,
			accuracy: 100,
			wantErr:  true,
		},
		{
			name:     "negative accuracy",
			lat:      12.0,
			lng:      77.0,
			accuracy: -1,
			wantErr:  true,
		},
		{
			name:     "edge case valid",
			lat:      90,
			lng:      180,
			accuracy: 99.9,
			wantErr:  false,
		},
		{
			name:     "edge case zero",
			lat:      0,
			lng:      0,
			accuracy: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocation(tt.lat, tt.lng, tt.accuracy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLocation(%f, %f, %f) error = %v, wantErr %v", tt.lat, tt.lng, tt.accuracy, err, tt.wantErr)
			}
		})
	}
}
