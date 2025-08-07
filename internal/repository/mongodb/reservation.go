// Package mongodb provides MongoDB implementations of repository interfaces.
package mongodb

import (
	"context"
	"github.com/reservia/api/internal/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/reservia/api/internal/repository"
)

// ReservationRepository implements the reservation repository interface using MongoDB.
type ReservationRepository struct {
	collection *mongo.Collection
}

// NewReservationRepository creates a new reservation repository.
func NewReservationRepository(db *mongo.Database) repository.ReservationRepository {
	return &ReservationRepository{
		collection: db.Collection("reservations"),
	}
}

// Create creates a new reservation.
func (r *ReservationRepository) Create(ctx context.Context, res *model.Reservation) error {
	res.ID = primitive.NewObjectID()
	res.CreatedAt = time.Now()
	res.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, res)
	return err
}

// GetByID retrieves a reservation by ID.
func (r *ReservationRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Reservation, error) {
	var res model.Reservation
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&res)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

// Update updates a reservation.
func (r *ReservationRepository) Update(ctx context.Context, res *model.Reservation) error {
	res.UpdatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": res.ID},
		bson.M{"$set": res},
	)
	return err
}

// Delete deletes a reservation.
func (r *ReservationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetByRestaurantID retrieves reservations by restaurant ID.
func (r *ReservationRepository) GetByRestaurantID(ctx context.Context, restaurantID primitive.ObjectID, limit, offset int) ([]*model.Reservation, error) {
	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(offset))
	opts.SetSort(bson.D{primitive.E{Key: "start_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, bson.M{"restaurant_id": restaurantID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reservations []*model.Reservation
	for cursor.Next(ctx) {
		var res model.Reservation
		if err := cursor.Decode(&res); err != nil {
			return nil, err
		}
		reservations = append(reservations, &res)
	}

	return reservations, cursor.Err()
}

// GetByDateRange retrieves reservations within a date range.
func (r *ReservationRepository) GetByDateRange(ctx context.Context, restaurantID primitive.ObjectID, startDate, endDate string) ([]*model.Reservation, error) {
	// Parse dates
	startTime, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, err
	}
	endTime, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, err
	}

	// Set end time to end of day
	endTime = endTime.Add(24*time.Hour - time.Nanosecond)

	filter := bson.M{
		"restaurant_id": restaurantID,
		"start_at": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	opts := options.Find()
	opts.SetSort(bson.D{primitive.E{Key: "start_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reservations []*model.Reservation
	for cursor.Next(ctx) {
		var res model.Reservation
		if err := cursor.Decode(&res); err != nil {
			return nil, err
		}
		reservations = append(reservations, &res)
	}

	return reservations, cursor.Err()
}

// CheckAvailability checks if a time slot is available.
func (r *ReservationRepository) CheckAvailability(ctx context.Context, restaurantID, roomID primitive.ObjectID, startTime, endTime string) (bool, error) {
	// Parse times
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return false, err
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return false, err
	}

	// Check for overlapping reservations
	filter := bson.M{
		"restaurant_id": restaurantID,
		"room_id":       roomID.Hex(),
		"status": bson.M{
			"$in": []model.Status{
				model.StatusPending,
				model.StatusConfirmed,
				model.StatusArrived,
			},
		},
		"$or": []bson.M{
			{
				"start_at": bson.M{
					"$lt": end,
				},
				"end_at": bson.M{
					"$gt": start,
				},
			},
		},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count == 0, nil
}
