// Package city contains the city model entities and logic.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// City represents a city in the system.
type City struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name              string             `json:"name" bson:"name"`
	Country           string             `json:"country" bson:"country"`
	CountryCode       string             `json:"countryCode" bson:"country_code"`
	AverageSalary     float64            `json:"averageSalary" bson:"average_salary"`
	SalaryCoefficient float64            `json:"salaryCoefficient" bson:"salary_coefficient"`
	UTCOffset         int                `json:"utcOffset" bson:"utc_offset"`
	CreatedAt         time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updatedAt" bson:"updated_at"`
}

// CityResponse represents the response when returning city data.
type CityResponse struct {
	ID                primitive.ObjectID `json:"id"`
	Name              string             `json:"name"`
	Country           string             `json:"country"`
	CountryCode       string             `json:"countryCode"`
	AverageSalary     float64            `json:"averageSalary"`
	SalaryCoefficient float64            `json:"salaryCoefficient"`
	UTCOffset         int                `json:"utcOffset"`
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
