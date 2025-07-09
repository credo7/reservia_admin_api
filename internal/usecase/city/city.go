package city

import (
	"context"

	"github.com/reservia/api/internal/domain/city"
	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
)

// UseCase handles city business logic.
type UseCase struct {
	repo   repository.CityRepository
	logger logger.Logger
}

// NewCityUseCase creates a new city use case.
func NewCityUseCase(repo repository.CityRepository, logger logger.Logger) *UseCase {
	return &UseCase{
		repo:   repo,
		logger: logger,
	}
}

// GetAllCities retrieves all cities.
func (uc *UseCase) GetAllCities(ctx context.Context) ([]*city.City, error) {
	uc.logger.Info("Getting all cities")

	cities, err := uc.repo.GetAll(ctx)
	if err != nil {
		uc.logger.Error("Failed to get cities", "error", err)
		return nil, err
	}

	uc.logger.Info("Successfully retrieved cities", "count", len(cities))
	return cities, nil
}

// GetCitiesByCountry retrieves cities by country.
func (uc *UseCase) GetCitiesByCountry(ctx context.Context, country string) ([]*city.City, error) {
	uc.logger.Info("Getting cities by country", "country", country)

	cities, err := uc.repo.GetByCountry(ctx, country)
	if err != nil {
		uc.logger.Error("Failed to get cities by country", "country", country, "error", err)
		return nil, err
	}

	uc.logger.Info("Successfully retrieved cities by country", "country", country, "count", len(cities))
	return cities, nil
}

// GetCityByName retrieves a city by name.
func (uc *UseCase) GetCityByName(ctx context.Context, name string) (*city.City, error) {
	uc.logger.Info("Getting city by name", "name", name)

	cityEntity, err := uc.repo.GetByName(ctx, name)
	if err != nil {
		uc.logger.Error("Failed to get city by name", "name", name, "error", err)
		return nil, err
	}

	uc.logger.Info("Successfully retrieved city by name", "name", name)
	return cityEntity, nil
}
