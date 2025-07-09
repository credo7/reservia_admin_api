// Package city contains the city domain entities and logic.
package city

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// City represents a city in the system.
type City struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name              string             `json:"name" bson:"name"`
	Country           string             `json:"country" bson:"country"`
	CountryCode       string             `json:"country_code" bson:"country_code"`
	AverageSalary     float64            `json:"average_salary" bson:"average_salary"`
	SalaryCoefficient float64            `json:"salary_coefficient" bson:"salary_coefficient"`
	UTCOffset         int                `json:"utc_offset" bson:"utc_offset"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" bson:"updated_at"`
}

// CityResponse represents the response when returning city data.
type CityResponse struct {
	ID                primitive.ObjectID `json:"id"`
	Name              string             `json:"name"`
	Country           string             `json:"country"`
	CountryCode       string             `json:"country_code"`
	AverageSalary     float64            `json:"average_salary"`
	SalaryCoefficient float64            `json:"salary_coefficient"`
	UTCOffset         int                `json:"utc_offset"`
}

// ToResponse converts a City to CityResponse.
func (c *City) ToResponse() *CityResponse {
	return &CityResponse{
		ID:                c.ID,
		Name:              c.Name,
		Country:           c.Country,
		CountryCode:       c.CountryCode,
		AverageSalary:     c.AverageSalary,
		SalaryCoefficient: c.SalaryCoefficient,
		UTCOffset:         c.UTCOffset,
	}
}

// GetTimezone returns the timezone string for the city.
func (c *City) GetTimezone() string {
	if c.UTCOffset >= 0 {
		return "UTC+" + string(rune(c.UTCOffset))
	}
	return "UTC" + string(rune(c.UTCOffset))
}

// IsInCountry checks if the city is in the specified country.
func (c *City) IsInCountry(country string) bool {
	return c.Country == country || c.CountryCode == country
}
