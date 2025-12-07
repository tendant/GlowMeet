package location

import "math"

// CalculateDistance returns the distance in kilometers between two points using the Haversine formula.
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// R is Earth radius. 6371 km. Convert to feet: 1 km = 3280.84 ft
	const R = 6371 * 3280.84

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
