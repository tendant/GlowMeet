package location

import (
	"testing"
)

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name string
		lat1 float64
		lon1 float64
		lat2 float64
		lon2 float64
		want float64 // Expected distance in feet
		tol  float64 // Tolerance
	}{
		{
			name: "Same location",
			lat1: 37.7749, lon1: -122.4194,
			lat2: 37.7749, lon2: -122.4194,
			want: 0,
			tol:  1.0,
		},
		{
			name: "Approx 1 degree latitude (111.19 km)",
			lat1: 0, lon1: 0,
			lat2: 1, lon2: 0,
			// 111.19 km * 3280.84 = 364816.5 ft
			want: 364816.5,
			tol:  500.0, // Accept some variance due to R precision
		},
		{
			name: "SF to LA (approx 347 miles / 1.83M ft)",
			lat1: 37.7749, lon1: -122.4194,
			lat2: 34.0522, lon2: -118.2437,
			// Google says ~559 km = 1,834,000 ft straight line
			// Haversine calc might differ slightly from geodesic
			want: 1834160,
			tol:  10000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if diff := abs(got - tt.want); diff > tt.tol {
				t.Errorf("CalculateDistance() = %v, want %v (diff %v > tol %v)", got, tt.want, diff, tt.tol)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
