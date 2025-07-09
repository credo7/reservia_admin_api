// Package mongodb provides MongoDB implementations of repository interfaces.
package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/reservia/api/internal/domain/user"
	"github.com/reservia/api/internal/repository"
)

// UserRepository implements the user repository interface using MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *mongo.Database) repository.UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

// Create creates a new user.
func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, u)
	return err
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*user.User, error) {
	var u user.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetByTelegramID retrieves a user by telegram ID.
func (r *UserRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	var u user.User
	err := r.collection.FindOne(ctx, bson.M{"tg_id": telegramID}).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// Update updates a user.
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	u.UpdatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": u.ID},
		bson.M{"$set": u},
	)
	return err
}

// Delete deletes a user.
func (r *UserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// List retrieves users with pagination.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*user.User, error) {
	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(offset))
	opts.SetSort(bson.D{{"created_at", -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []*user.User
	for cursor.Next(ctx) {
		var u user.User
		if err := cursor.Decode(&u); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}

	return users, cursor.Err()
}
