// Package mongodb provides MongoDB implementations of repository interfaces.
package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/reservia/api/internal/domain/auth"
	"github.com/reservia/api/internal/repository"
)

// AuthRepository implements the auth repository interface using MongoDB.
type AuthRepository struct {
	verificationCodes *mongo.Collection
	refreshTokens     *mongo.Collection
}

// NewAuthRepository creates a new auth repository.
func NewAuthRepository(db *mongo.Database) repository.AuthRepository {
	return &AuthRepository{
		verificationCodes: db.Collection("verification_codes"),
		refreshTokens:     db.Collection("refresh_tokens"),
	}
}

// CreateVerificationCode creates a verification code.
func (r *AuthRepository) CreateVerificationCode(ctx context.Context, code *auth.VerificationCode) error {
	code.ID = primitive.NewObjectID()
	code.CreatedAt = time.Now()
	code.UpdatedAt = time.Now()

	_, err := r.verificationCodes.InsertOne(ctx, code)
	return err
}

// GetVerificationCode retrieves a verification code.
func (r *AuthRepository) GetVerificationCode(ctx context.Context, email, code string) (*auth.VerificationCode, error) {
	var vc auth.VerificationCode
	err := r.verificationCodes.FindOne(ctx, bson.M{
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

// DeleteVerificationCode deletes a verification code.
func (r *AuthRepository) DeleteVerificationCode(ctx context.Context, email, code string) error {
	_, err := r.verificationCodes.DeleteOne(ctx, bson.M{
		"email": email,
		"code":  code,
	})
	return err
}

// CreateRefreshToken creates a refresh token.
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	token.ID = primitive.NewObjectID()
	token.CreatedAt = time.Now()
	token.UpdatedAt = time.Now()

	_, err := r.refreshTokens.InsertOne(ctx, token)
	return err
}

// GetRefreshToken retrieves a refresh token.
func (r *AuthRepository) GetRefreshToken(ctx context.Context, token string) (*auth.RefreshToken, error) {
	var rt auth.RefreshToken
	err := r.refreshTokens.FindOne(ctx, bson.M{"token": token}).Decode(&rt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

// DeleteRefreshToken deletes a refresh token.
func (r *AuthRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	_, err := r.refreshTokens.DeleteOne(ctx, bson.M{"token": token})
	return err
}
