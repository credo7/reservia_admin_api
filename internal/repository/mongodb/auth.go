// Package mongodb provides MongoDB implementations of repository interfaces.
package mongodb

import (
	"context"
	"fmt"
	"reservia-admin-api/internal/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"reservia-admin-api/internal/repository"
)

// AuthRepository implements the auth repository interface using MongoDB.
type AuthRepository struct {
	registerVerificationCodes  *mongo.Collection
	loginVerificationCodes     *mongo.Collection
	registerEmployeeCodes      *mongo.Collection
	tgVerificationCodes        *mongo.Collection
	emailUpdateVerificationCodes *mongo.Collection
	failedAttempts             *mongo.Collection
}

// NewAuthRepository creates a new auth repository.
func NewAuthRepository(db *mongo.Database) repository.AuthRepository {
	return &AuthRepository{
		loginVerificationCodes:       db.Collection("employees_login_verification_codes"),
		registerVerificationCodes:    db.Collection("employees_register_verification_codes"),
		registerEmployeeCodes:        db.Collection("employees_register_employee_codes"),
		tgVerificationCodes:          db.Collection("employees_tg_verification_codes"),
		emailUpdateVerificationCodes: db.Collection("employees_email_update_verification_codes"),
		failedAttempts:              db.Collection("failed_attempts"),
	}
}

// Login verification code methods

// CreateLoginVerificationCode creates a login verification code.
func (r *AuthRepository) CreateLoginVerificationCode(ctx context.Context, code *model.LoginVerificationCode) error {
	code.ID = primitive.NewObjectID()
	code.CreatedAt = time.Now()
	code.UpdatedAt = time.Now()

	_, err := r.loginVerificationCodes.InsertOne(ctx, code)
	return err
}

// GetLoginVerificationCode retrieves a login verification code by email and code.
func (r *AuthRepository) GetLoginVerificationCode(ctx context.Context, email, code string) (*model.LoginVerificationCode, error) {
	var vc model.LoginVerificationCode
	err := r.loginVerificationCodes.FindOne(ctx, bson.M{
		"email": email,
		"code":  code,
	}).Decode(&vc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &vc, nil
}

// GetLoginVerificationCodeByID retrieves a login verification code by ID.
func (r *AuthRepository) GetLoginVerificationCodeByID(ctx context.Context, id primitive.ObjectID) (*model.LoginVerificationCode, error) {
	var vc model.LoginVerificationCode
	err := r.loginVerificationCodes.FindOne(ctx, bson.M{"_id": id}).Decode(&vc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("verification code not found")
		}
		return nil, err
	}
	return &vc, nil
}

// UpdateLoginVerificationCode updates a login verification code.
func (r *AuthRepository) UpdateLoginVerificationCode(ctx context.Context, code *model.LoginVerificationCode) error {
	filter := bson.M{"_id": code.ID}
	update := bson.M{
		"$set": bson.M{
			"is_used":    code.IsUsed,
			"updated_at": code.UpdatedAt,
		},
	}
	_, err := r.loginVerificationCodes.UpdateOne(ctx, filter, update)
	return err
}

// DeleteLoginVerificationCode deletes a login verification code.
func (r *AuthRepository) DeleteLoginVerificationCode(ctx context.Context, email, code string) error {
	_, err := r.loginVerificationCodes.DeleteOne(ctx, bson.M{
		"email": email,
		"code":  code,
	})
	return err
}

// Register verification code methods

// CreateRegisterVerificationCode creates a register verification code.
func (r *AuthRepository) CreateRegisterVerificationCode(ctx context.Context, code *model.RegisterVerificationCode) error {
	code.ID = primitive.NewObjectID()
	code.CreatedAt = time.Now()
	code.UpdatedAt = time.Now()

	_, err := r.registerVerificationCodes.InsertOne(ctx, code)
	return err
}

// GetRegisterVerificationCode retrieves a register verification code by email and code.
func (r *AuthRepository) GetRegisterVerificationCode(ctx context.Context, email, code string) (*model.RegisterVerificationCode, error) {
	var vc model.RegisterVerificationCode
	err := r.registerVerificationCodes.FindOne(ctx, bson.M{
		"email": email,
		"code":  code,
	}).Decode(&vc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &vc, nil
}

// GetRegisterVerificationCodeByID retrieves a register verification code by ID.
func (r *AuthRepository) GetRegisterVerificationCodeByID(ctx context.Context, id primitive.ObjectID) (*model.RegisterVerificationCode, error) {
	var vc model.RegisterVerificationCode
	err := r.registerVerificationCodes.FindOne(ctx, bson.M{"_id": id}).Decode(&vc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("verification code not found")
		}
		return nil, err
	}
	return &vc, nil
}

// UpdateRegisterVerificationCode updates a register verification code.
func (r *AuthRepository) UpdateRegisterVerificationCode(ctx context.Context, code *model.RegisterVerificationCode) error {
	filter := bson.M{"_id": code.ID}
	update := bson.M{
		"$set": bson.M{
			"is_used":    code.IsUsed,
			"updated_at": code.UpdatedAt,
		},
	}
	_, err := r.registerVerificationCodes.UpdateOne(ctx, filter, update)
	return err
}

// DeleteRegisterVerificationCode deletes a register verification code.
func (r *AuthRepository) DeleteRegisterVerificationCode(ctx context.Context, email, code string) error {
	_, err := r.registerVerificationCodes.DeleteOne(ctx, bson.M{
		"email": email,
		"code":  code,
	})
	return err
}

// Employee registration code methods (for invitations)

// CreateEmployeeRegistrationCode creates an employee registration code.
func (r *AuthRepository) CreateEmployeeRegistrationCode(ctx context.Context, code *model.EmployeeRegistrationCode) error {
	code.ID = primitive.NewObjectID()
	code.CreatedAt = time.Now()
	code.UpdatedAt = time.Now()

	_, err := r.registerEmployeeCodes.InsertOne(ctx, code)
	return err
}

// GetEmployeeRegistrationCode retrieves an employee registration code by ID.
func (r *AuthRepository) GetEmployeeRegistrationCode(ctx context.Context, requestID string) (*model.EmployeeRegistrationCode, error) {
	// Convert string to ObjectID
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, fmt.Errorf("invalid ObjectID: %w", err)
	}

	var empRegCode model.EmployeeRegistrationCode
	err = r.registerEmployeeCodes.FindOne(ctx, bson.M{"_id": objectID}).Decode(&empRegCode)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &empRegCode, nil
}

// UpdateEmployeeRegistrationCode updates an employee registration code.
func (r *AuthRepository) UpdateEmployeeRegistrationCode(ctx context.Context, code *model.EmployeeRegistrationCode) error {
	filter := bson.M{"_id": code.ID}
	update := bson.M{
		"$set": bson.M{
			"employee_id": code.EmployeeID,
			"is_used":     code.IsUsed,
			"updated_at":  time.Now(),
		},
	}
	_, err := r.registerEmployeeCodes.UpdateOne(ctx, filter, update)
	return err
}

// GetPendingEmployeeRegistrationCodes retrieves all pending (unused and not expired) employee registration codes.
func (r *AuthRepository) GetPendingEmployeeRegistrationCodes(ctx context.Context) ([]*model.EmployeeRegistrationCode, error) {
	// Find codes that are not used and not expired (created within last 5 minutes)
	expirationTime := time.Now().Add(-model.AuthRequestExpiration)
	filter := bson.M{
		"is_used": false,
		"created_at": bson.M{
			"$gt": expirationTime,
		},
	}

	cursor, err := r.registerEmployeeCodes.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var codes []*model.EmployeeRegistrationCode
	for cursor.Next(ctx) {
		var code model.EmployeeRegistrationCode
		if err := cursor.Decode(&code); err != nil {
			return nil, err
		}
		codes = append(codes, &code)
	}

	return codes, cursor.Err()
}

// DeleteEmployeeRegistrationCode deletes an employee registration code by ID.
func (r *AuthRepository) DeleteEmployeeRegistrationCode(ctx context.Context, requestID string) error {
	// Convert string to ObjectID
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return fmt.Errorf("invalid ObjectID: %w", err)
	}

	_, err = r.registerEmployeeCodes.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// Telegram verification code methods (for admin bot auth)

// CreateTelegramVerificationCode creates a Telegram verification code.
func (r *AuthRepository) CreateTelegramVerificationCode(ctx context.Context, code *model.TelegramVerificationCode) error {
	code.ID = primitive.NewObjectID()
	code.CreatedAt = time.Now()
	code.UpdatedAt = time.Now()

	_, err := r.tgVerificationCodes.InsertOne(ctx, code)
	return err
}

// GetTelegramVerificationCode retrieves a Telegram verification code by ID.
func (r *AuthRepository) GetTelegramVerificationCode(ctx context.Context, requestID string) (*model.TelegramVerificationCode, error) {
	// Convert string to ObjectID
	objectID, err := primitive.ObjectIDFromHex(requestID)
	if err != nil {
		return nil, fmt.Errorf("invalid ObjectID: %w", err)
	}

	var tgCode model.TelegramVerificationCode
	err = r.tgVerificationCodes.FindOne(ctx, bson.M{"_id": objectID}).Decode(&tgCode)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &tgCode, nil
}

// UpdateTelegramVerificationCode updates a Telegram verification code.
func (r *AuthRepository) UpdateTelegramVerificationCode(ctx context.Context, code *model.TelegramVerificationCode) error {
	filter := bson.M{"_id": code.ID}
	update := bson.M{
		"$set": bson.M{
			"employee_id":    code.EmployeeID,
			"is_used":    code.IsUsed,
			"updated_at": time.Now(),
		},
	}
	_, err := r.tgVerificationCodes.UpdateOne(ctx, filter, update)
	return err
}

// Email update verification code methods

// CreateEmailUpdateVerificationCode creates an email update verification code.
func (r *AuthRepository) CreateEmailUpdateVerificationCode(ctx context.Context, code *model.EmailUpdateVerificationCode) error {
	code.ID = primitive.NewObjectID()
	code.CreatedAt = time.Now()
	code.UpdatedAt = time.Now()

	_, err := r.emailUpdateVerificationCodes.InsertOne(ctx, code)
	return err
}

// GetEmailUpdateVerificationCodeByID retrieves an email update verification code by ID.
func (r *AuthRepository) GetEmailUpdateVerificationCodeByID(ctx context.Context, id primitive.ObjectID) (*model.EmailUpdateVerificationCode, error) {
	var euvc model.EmailUpdateVerificationCode
	err := r.emailUpdateVerificationCodes.FindOne(ctx, bson.M{"_id": id}).Decode(&euvc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("email update verification code not found")
		}
		return nil, err
	}
	return &euvc, nil
}

// UpdateEmailUpdateVerificationCode updates an email update verification code.
func (r *AuthRepository) UpdateEmailUpdateVerificationCode(ctx context.Context, code *model.EmailUpdateVerificationCode) error {
	filter := bson.M{"_id": code.ID}
	update := bson.M{
		"$set": bson.M{
			"is_used":    code.IsUsed,
			"updated_at": code.UpdatedAt,
		},
	}
	_, err := r.emailUpdateVerificationCodes.UpdateOne(ctx, filter, update)
	return err
}

// Security improvement methods

// InvalidateActiveLoginCodes invalidates all active login verification codes for an email.
func (r *AuthRepository) InvalidateActiveLoginCodes(ctx context.Context, email string) error {
	filter := bson.M{
		"email":   email,
		"is_used": false,
		"created_at": bson.M{
			"$gt": time.Now().Add(-model.AuthRequestExpiration),
		},
	}
	update := bson.M{
		"$set": bson.M{
			"is_used":    true,
			"updated_at": time.Now(),
		},
	}
	_, err := r.loginVerificationCodes.UpdateMany(ctx, filter, update)
	return err
}

// InvalidateActiveRegisterCodes invalidates all active register verification codes for an email.
func (r *AuthRepository) InvalidateActiveRegisterCodes(ctx context.Context, email string) error {
	filter := bson.M{
		"email":   email,
		"is_used": false,
		"created_at": bson.M{
			"$gt": time.Now().Add(-model.AuthRequestExpiration),
		},
	}
	update := bson.M{
		"$set": bson.M{
			"is_used":    true,
			"updated_at": time.Now(),
		},
	}
	_, err := r.registerVerificationCodes.UpdateMany(ctx, filter, update)
	return err
}

// RecordFailedAttempt records a failed authentication attempt.
func (r *AuthRepository) RecordFailedAttempt(ctx context.Context, email, action string) error {
	now := time.Now()
	expiresAt := now.Add(model.LockoutDuration)
	
	filter := bson.M{
		"email":  email,
		"action": action,
	}
	
	// Use upsert to either update existing record or create new one
	update := bson.M{
		"$inc": bson.M{"count": 1},
		"$set": bson.M{
			"last_at":    now,
			"expires_at": expiresAt,
		},
		"$setOnInsert": bson.M{
			"_id":      primitive.NewObjectID(),
			"first_at": now,
		},
	}
	
	opts := options.Update().SetUpsert(true)
	_, err := r.failedAttempts.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetFailedAttempt retrieves a failed attempt record for an email and action.
func (r *AuthRepository) GetFailedAttempt(ctx context.Context, email, action string) (*model.FailedAttempt, error) {
	var attempt model.FailedAttempt
	
	err := r.failedAttempts.FindOne(ctx, bson.M{
		"email":  email,
		"action": action,
	}).Decode(&attempt)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	
	return &attempt, nil
}

// ClearFailedAttempts clears all failed attempts for an email.
func (r *AuthRepository) ClearFailedAttempts(ctx context.Context, email string) error {
	// Delete all failed attempt records for this email
	_, err := r.failedAttempts.DeleteMany(ctx, bson.M{"email": email})
	return err
}

// GetFailedAttemptsByEmail retrieves all failed attempt records for an email.
func (r *AuthRepository) GetFailedAttemptsByEmail(ctx context.Context, email string) ([]*model.FailedAttempt, error) {
	cursor, err := r.failedAttempts.Find(ctx, bson.M{
		"email": email,
		"action": bson.M{"$in": []string{"login", "register"}},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var attempts []*model.FailedAttempt
	for cursor.Next(ctx) {
		var attempt model.FailedAttempt
		if err := cursor.Decode(&attempt); err != nil {
			return nil, err
		}
		attempts = append(attempts, &attempt)
	}
	
	return attempts, cursor.Err()
}
