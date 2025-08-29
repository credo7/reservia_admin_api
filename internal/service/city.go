package service

import (
	"context"
	"reservia-admin-api/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"reservia-admin-api/internal/repository"
	"reservia-admin-api/pkg/logger"
)

// Service handles city business logic.
type CityService struct {
	repo   repository.CityRepository
	logger logger.Logger
}

// NewCityService creates a new city use case.
func NewCityService(repo repository.CityRepository, logger logger.Logger) *CityService {
	return &CityService{
		repo:   repo,
		logger: logger,
	}
}

// GetAllCities retrieves all cities.
func (cs *CityService) GetAllCities(ctx context.Context) ([]*model.City, error) {
	cs.logger.Info("Getting all cities")

	cities, err := cs.repo.GetAll(ctx)
	if err != nil {
		cs.logger.Error("Failed to get cities", "error", err)
		return nil, err
	}

	cs.logger.Info("Successfully retrieved cities", "count", len(cities))
	return cities, nil
}

// GetCitiesByCountry retrieves cities by country.
func (cs *CityService) GetCitiesByCountry(ctx context.Context, country string) ([]*model.City, error) {
	cs.logger.Info("Getting cities by country", "country", country)

	cities, err := cs.repo.GetByCountry(ctx, country)
	if err != nil {
		cs.logger.Error("Failed to get cities by country", "country", country, "error", err)
		return nil, err
	}

	cs.logger.Info("Successfully retrieved cities by country", "country", country, "count", len(cities))
	return cities, nil
}

// GetCityByID retrieves a city by ID.
func (cs *CityService) GetCityByID(ctx context.Context, id primitive.ObjectID) (*model.City, error) {
	cs.logger.Info("Getting city by ID", "id", id.Hex())

	cityEntity, err := cs.repo.GetByID(ctx, id)
	if err != nil {
		cs.logger.Error("Failed to get city by ID", "id", id.Hex(), "error", err)
		return nil, err
	}

	if cityEntity == nil {
		cs.logger.Warn("City not found", "id", id.Hex())
		return nil, nil
	}

	cs.logger.Info("Successfully retrieved city by ID", "id", id.Hex(), "name", cityEntity.Name)
	return cityEntity, nil
}

// GetCityByName retrieves a city by name.
func (cs *CityService) GetCityByName(ctx context.Context, name string) (*model.City, error) {
	cs.logger.Info("Getting city by name", "name", name)

	cityEntity, err := cs.repo.GetByName(ctx, name)
	if err != nil {
		cs.logger.Error("Failed to get city by name", "name", name, "error", err)
		return nil, err
	}

	cs.logger.Info("Successfully retrieved city by name", "name", name)
	return cityEntity, nil
}