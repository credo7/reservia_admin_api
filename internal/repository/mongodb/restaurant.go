// Package mongodb provides MongoDB implementations of repository interfaces.
package mongodb

import (
	"context"
	"reservia-admin-api/internal/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"reservia-admin-api/internal/repository"
)

// RestaurantRepository implements the restaurant repository interface using MongoDB.
type RestaurantRepository struct {
	collection *mongo.Collection
}

// NewRestaurantRepository creates a new restaurant repository.
func NewRestaurantRepository(db *mongo.Database) repository.RestaurantRepository {
	return &RestaurantRepository{
		collection: db.Collection("restaurants"),
	}
}

// Create creates a new restaurant.
func (r *RestaurantRepository) Create(ctx context.Context, rest *model.Restaurant) error {
	rest.ID = primitive.NewObjectID()
	rest.CreatedAt = time.Now()
	rest.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, rest)
	return err
}

// GetByID retrieves a restaurant by ID.
func (r *RestaurantRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Restaurant, error) {
	var rest model.Restaurant
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rest)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &rest, nil
}

// GetByURLName retrieves a restaurant by URL name.
func (r *RestaurantRepository) GetByURLName(ctx context.Context, urlName string) (*model.Restaurant, error) {
	var rest model.Restaurant
	err := r.collection.FindOne(ctx, bson.M{"url_name": urlName}).Decode(&rest)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &rest, nil
}

// Update updates a restaurant.
func (r *RestaurantRepository) Update(ctx context.Context, rest *model.Restaurant) error {
	rest.UpdatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": rest.ID},
		bson.M{"$set": rest},
	)
	return err
}

// Delete deletes a restaurant.
func (r *RestaurantRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// List retrieves restaurants with filtering and pagination.
func (r *RestaurantRepository) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*model.Restaurant, error) {
	// Build filter
	filter := bson.M{}
	for key, value := range filters {
		filter[key] = value
	}

	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(offset))
	opts.SetSort(bson.D{primitive.E{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var restaurants []*model.Restaurant
	for cursor.Next(ctx) {
		var rest model.Restaurant
		if err := cursor.Decode(&rest); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, &rest)
	}

	return restaurants, cursor.Err()
}

// GetByLocation retrieves restaurants by location.
func (r *RestaurantRepository) GetByLocation(ctx context.Context, latitude, longitude, radius float64) ([]*model.Restaurant, error) {
	// For now, return all restaurants - in a real implementation, you'd use geospatial queries
	return r.List(ctx, map[string]interface{}{}, 100, 0)
}

// GetByIDs retrieves restaurants by a list of IDs.
func (r *RestaurantRepository) GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*model.Restaurant, error) {
	if len(ids) == 0 {
		return []*model.Restaurant{}, nil
	}

	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var restaurants []*model.Restaurant
	for cursor.Next(ctx) {
		var rest model.Restaurant
		if err := cursor.Decode(&rest); err != nil {
			return nil, err
		}
		restaurants = append(restaurants, &rest)
	}

	return restaurants, cursor.Err()
}
