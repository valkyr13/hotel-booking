package owner


type Amenity string

const (
	AmenityWiFi            Amenity = "WIFI"
	AmenityPool            Amenity = "POOL"
	AmenityParking         Amenity = "PARKING"
	AmenityAirConditioning Amenity = "AC"
	AmenityTV              Amenity = "TV"
	AmenityBreakfast       Amenity = "BREAKFAST"
	AmenityGym             Amenity = "GYM"
)

func HasAmenity(amenities []Amenity, target Amenity) bool {
	for _, a := range amenities {
		if a == target {
			return true
		}
	}
	return false
}
