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

// EmployeeRepository implements the employee repository interface using MongoDB.
type EmployeeRepository struct {
	collection *mongo.Collection
}

// NewEmployeeRepository creates a new employee repository.
func NewEmployeeRepository(db *mongo.Database) repository.EmployeeRepository {
	return &EmployeeRepository{
		collection: db.Collection("employees"),
	}
}

// Create creates a new employee.
func (r *EmployeeRepository) Create(ctx context.Context, e *model.Employee) error {
	e.ID = primitive.NewObjectID()
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, e)
	return err
}

// GetByID retrieves an employee by ID.
func (r *EmployeeRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Employee, error) {
	var e model.Employee
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&e)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

// GetByEmail retrieves an employee by email.
func (r *EmployeeRepository) GetByEmail(ctx context.Context, email string) (*model.Employee, error) {
	var e model.Employee
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&e)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

// GetByTelegramID retrieves an employee by telegram ID.
func (r *EmployeeRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*model.Employee, error) {
	var e model.Employee
	err := r.collection.FindOne(ctx, bson.M{"tg_id": telegramID}).Decode(&e)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

// Update updates an employee.
func (r *EmployeeRepository) Update(ctx context.Context, e *model.Employee) error {
	e.UpdatedAt = time.Now()

	// Build update document with both $set and $unset operations
	setFields := bson.M{
		"full_name":   e.FullName,
		"restaurants": e.Restaurants,
		"created_at":  e.CreatedAt,
		"updated_at":  e.UpdatedAt,
	}
	
	// Handle email field - only set if not empty
	if e.Email != "" {
		setFields["email"] = e.Email
	}

	unsetFields := bson.M{}

	// Handle Telegram fields - if nil, unset them; otherwise set them
	if e.TelegramIsBot != nil {
		setFields["tg_is_bot"] = e.TelegramIsBot
	} else {
		unsetFields["tg_is_bot"] = ""
	}

	if e.TelegramID != nil {
		setFields["tg_id"] = e.TelegramID
	} else {
		unsetFields["tg_id"] = ""
	}

	if e.TelegramChatID != nil {
		setFields["tg_chat_id"] = e.TelegramChatID
	} else {
		unsetFields["tg_chat_id"] = ""
	}

	if e.TelegramUsername != nil {
		setFields["tg_username"] = e.TelegramUsername
	} else {
		unsetFields["tg_username"] = ""
	}

	if e.TelegramLang != nil {
		setFields["tg_lang"] = e.TelegramLang
	} else {
		unsetFields["tg_lang"] = ""
	}

	if e.TelegramPremium != nil {
		setFields["tg_is_premium"] = e.TelegramPremium
	} else {
		unsetFields["tg_is_premium"] = ""
	}

	// Build the update document
	update := bson.M{"$set": setFields}
	if len(unsetFields) > 0 {
		update["$unset"] = unsetFields
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": e.ID},
		update,
	)
	return err
}

// Delete deletes an employee.
func (r *EmployeeRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// List retrieves employees with pagination.
func (r *EmployeeRepository) List(ctx context.Context, limit, offset int) ([]*model.Employee, error) {
	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(offset))
	opts.SetSort(bson.D{primitive.E{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var employees []*model.Employee
	for cursor.Next(ctx) {
		var e model.Employee
		if err := cursor.Decode(&e); err != nil {
			return nil, err
		}
		employees = append(employees, &e)
	}

	return employees, cursor.Err()
}

// ListByRestaurantIDs retrieves employees who have access to any of the specified restaurants.
func (r *EmployeeRepository) ListByRestaurantIDs(ctx context.Context, restaurantIDs []primitive.ObjectID) ([]*model.Employee, error) {
	// Use MongoDB query to filter employees by restaurant access at the database level
	filter := bson.M{
		"restaurants.restaurant_id": bson.M{
			"$in": restaurantIDs,
		},
	}

	opts := options.Find()
	opts.SetSort(bson.D{primitive.E{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var employees []*model.Employee
	for cursor.Next(ctx) {
		var e model.Employee
		if err := cursor.Decode(&e); err != nil {
			return nil, err
		}
		employees = append(employees, &e)
	}

	return employees, cursor.Err()
}
