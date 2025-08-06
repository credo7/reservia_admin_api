// Package mongodb provides MongoDB implementations of repository interfaces.
package mongodb

import (
	"context"
	"github.com/reservia/api/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/reservia/api/internal/repository"
)

// CityRepository implements the city repository interface using MongoDB.
type CityRepository struct {
	collection *mongo.Collection
}

// NewCityRepository creates a new city repository.
func NewCityRepository(db *mongo.Database) repository.CityRepository {
	return &CityRepository{
		collection: db.Collection("cities"),
	}
}

// GetAll retrieves all cities.
func (r *CityRepository) GetAll(ctx context.Context) ([]*model.City, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var cities []*model.City
	for cursor.Next(ctx) {
		var c model.City
		if err := cursor.Decode(&c); err != nil {
			return nil, err
		}
		cities = append(cities, &c)
	}

	return cities, cursor.Err()
}

// GetByName retrieves a city by name.
func (r *CityRepository) GetByName(ctx context.Context, name string) (*model.City, error) {
	var c model.City
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// GetByCountry retrieves cities by country.
func (r *CityRepository) GetByCountry(ctx context.Context, country string) ([]*model.City, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"country": country},
			{"country_code": country},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var cities []*model.City
	for cursor.Next(ctx) {
		var c model.City
		if err := cursor.Decode(&c); err != nil {
			return nil, err
		}
		cities = append(cities, &c)
	}

	return cities, cursor.Err()
}
