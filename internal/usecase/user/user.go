// Package user provides user-related use cases.
package user

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/domain/user"
	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
)

// UseCase handles user-related business logic.
type UseCase struct {
	userRepo repository.UserRepository
	logger   logger.Logger
}

// New creates a new user use case.
func New(userRepo repository.UserRepository, logger logger.Logger) *UseCase {
	return &UseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreateUser creates a new user.
func (uc *UseCase) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.User, error) {
	// Check if user already exists
	if req.Email != "" {
		existingUser, err := uc.userRepo.GetByEmail(ctx, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing user: %w", err)
		}
		if existingUser != nil {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
	}

	// Create user
	newUser := &user.User{
		Email:            req.Email,
		FullName:         req.FullName,
		TelegramIsBot:    req.TelegramIsBot,
		TelegramID:       req.TelegramID,
		TelegramChatID:   req.TelegramChatID,
		TelegramUsername: req.TelegramUsername,
		TelegramLang:     req.TelegramLang,
		TelegramPremium:  req.TelegramPremium,
		Restaurants:      req.Restaurants,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	uc.logger.Info("User created successfully", "user_id", newUser.ID.Hex())
	return newUser, nil
}

// GetUserByID retrieves a user by ID.
func (uc *UseCase) GetUserByID(ctx context.Context, id primitive.ObjectID) (*user.User, error) {
	u, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	if u == nil {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

// GetUserByEmail retrieves a user by email.
func (uc *UseCase) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	u, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	if u == nil {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

// GetUserByTelegramID retrieves a user by telegram ID.
func (uc *UseCase) GetUserByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	u, err := uc.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by telegram ID: %w", err)
	}
	if u == nil {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

// UpdateUser updates a user.
func (uc *UseCase) UpdateUser(ctx context.Context, id primitive.ObjectID, req *user.UpdateUserRequest) (*user.User, error) {
	// Get existing user
	existingUser, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Update fields
	if req.Email != nil {
		existingUser.Email = *req.Email
	}
	if req.FullName != nil {
		existingUser.FullName = *req.FullName
	}
	if req.TelegramIsBot != nil {
		existingUser.TelegramIsBot = req.TelegramIsBot
	}
	if req.TelegramID != nil {
		existingUser.TelegramID = req.TelegramID
	}
	if req.TelegramChatID != nil {
		existingUser.TelegramChatID = req.TelegramChatID
	}
	if req.TelegramUsername != nil {
		existingUser.TelegramUsername = req.TelegramUsername
	}
	if req.TelegramLang != nil {
		existingUser.TelegramLang = req.TelegramLang
	}
	if req.TelegramPremium != nil {
		existingUser.TelegramPremium = req.TelegramPremium
	}

	existingUser.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, existingUser); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	uc.logger.Info("User updated successfully", "user_id", existingUser.ID.Hex())
	return existingUser, nil
}

// DeleteUser deletes a user.
func (uc *UseCase) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	// Check if user exists
	existingUser, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if existingUser == nil {
		return fmt.Errorf("user not found")
	}

	if err := uc.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	uc.logger.Info("User deleted successfully", "user_id", id.Hex())
	return nil
}

// ListUsers retrieves users with pagination.
func (uc *UseCase) ListUsers(ctx context.Context, limit, offset int) ([]*user.User, error) {
	users, err := uc.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}
