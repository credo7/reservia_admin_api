package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/reservia/api/internal/domain/auth"
	"github.com/reservia/api/internal/domain/user"
	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// UseCase handles authentication business logic.
type UseCase struct {
	authRepo repository.AuthRepository
	userRepo repository.UserRepository
	logger   logger.Logger
}

// NewAuthUseCase creates a new auth use case.
func NewAuthUseCase(authRepo repository.AuthRepository, userRepo repository.UserRepository, logger logger.Logger) *UseCase {
	return &UseCase{
		authRepo: authRepo,
		userRepo: userRepo,
		logger:   logger,
	}
}

// Login initiates the passwordless login process by sending a verification code.
func (uc *UseCase) Login(ctx context.Context, req *auth.LoginRequest) (*auth.AuthResponse, error) {
	uc.logger.Info("Attempting login", "email", req.Email)

	// Find user by email
	userEntity, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		uc.logger.Error("User not found during login", "email", req.Email, "error", err)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate verification code
	verificationCode, err := uc.generateVerificationCode()
	if err != nil {
		uc.logger.Error("Failed to generate verification code", "error", err)
		return nil, fmt.Errorf("failed to generate verification code")
	}

	// Save verification code
	codeEntity := &auth.VerificationCode{
		ID:        primitive.NewObjectID(),
		Email:     req.Email,
		Code:      verificationCode,
		Purpose:   "login",
		ExpiresAt: time.Now().Add(auth.VerificationCodeExpiration),
		IsUsed:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.authRepo.CreateVerificationCode(ctx, codeEntity); err != nil {
		uc.logger.Error("Failed to save verification code", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to save verification code")
	}

	uc.logger.Info("Login verification code sent", "email", req.Email, "code", verificationCode)

	return &auth.AuthResponse{
		AccessToken:  "",
		RefreshToken: "",
		TokenType:    auth.TokenTypeBearer,
		ExpiresIn:    0,
		ExpiresAt:    time.Time{},
		User: auth.UserInfo{
			ID:          primitive.ObjectID{},
			Email:       req.Email,
			FullName:    "",
			HasEmail:    true,
			HasTelegram: userEntity.HasTelegram(),
		},
	}, nil
}

// Register creates a new user account.
func (uc *UseCase) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.AuthResponse, error) {
	uc.logger.Info("Attempting registration", "email", req.Email)

	// Check if user already exists
	existingUser, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		uc.logger.Error("User already exists", "email", req.Email)
		return nil, fmt.Errorf("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		uc.logger.Error("Failed to hash password", "error", err)
		return nil, fmt.Errorf("failed to hash password")
	}

	// Create user
	newUser := &user.User{
		ID:        primitive.NewObjectID(),
		Email:     req.Email,
		FullName:  req.FullName,
		Password:  string(hashedPassword),
		Role:      user.RoleUser,
		Status:    user.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdUser, err := uc.userRepo.Create(ctx, newUser)
	if err != nil {
		uc.logger.Error("Failed to create user", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to create user")
	}

	// Generate verification code
	verificationCode, err := uc.generateVerificationCode()
	if err != nil {
		uc.logger.Error("Failed to generate verification code", "error", err)
		return nil, fmt.Errorf("failed to generate verification code")
	}

	// Save verification code
	codeEntity := &auth.VerificationCode{
		ID:        primitive.NewObjectID(),
		Email:     req.Email,
		Code:      verificationCode,
		Purpose:   auth.PurposeEmailVerification,
		ExpiresAt: time.Now().Add(auth.VerificationCodeExpiration),
		IsUsed:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.authRepo.SaveVerificationCode(ctx, codeEntity); err != nil {
		uc.logger.Error("Failed to save verification code", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to save verification code")
	}

	uc.logger.Info("Registration successful, verification code sent", "user_id", createdUser.ID, "email", req.Email)

	// Generate tokens for immediate login
	accessToken, refreshToken, err := uc.generateTokens(createdUser.ID)
	if err != nil {
		uc.logger.Error("Failed to generate tokens after registration", "user_id", createdUser.ID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens")
	}

	// Save refresh token
	if err := uc.saveRefreshToken(ctx, refreshToken, createdUser.ID); err != nil {
		uc.logger.Error("Failed to save refresh token after registration", "user_id", createdUser.ID, "error", err)
		return nil, fmt.Errorf("failed to save refresh token")
	}

	return &auth.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    auth.TokenTypeBearer,
		ExpiresIn:    int(auth.AccessTokenExpiration.Seconds()),
		ExpiresAt:    time.Now().Add(auth.AccessTokenExpiration),
		User: auth.UserInfo{
			ID:          createdUser.ID,
			Email:       createdUser.Email,
			FullName:    createdUser.FullName,
			HasEmail:    createdUser.Email != "",
			HasTelegram: createdUser.TelegramID != 0,
		},
	}, nil
}

// VerifyEmail verifies an email address with a verification code.
func (uc *UseCase) VerifyEmail(ctx context.Context, req *auth.VerifyEmailRequest) error {
	uc.logger.Info("Attempting email verification", "email", req.Email)

	// Find verification code
	codeEntity, err := uc.authRepo.GetVerificationCode(ctx, req.Email, req.Code, auth.PurposeEmailVerification)
	if err != nil {
		uc.logger.Error("Verification code not found", "email", req.Email, "code", req.Code)
		return fmt.Errorf("invalid verification code")
	}

	// Check if code is valid
	if !codeEntity.IsValid() {
		uc.logger.Error("Verification code is invalid or expired", "email", req.Email, "code", req.Code)
		return fmt.Errorf("verification code is invalid or expired")
	}

	// Mark code as used
	codeEntity.MarkAsUsed()
	if err := uc.authRepo.UpdateVerificationCode(ctx, codeEntity); err != nil {
		uc.logger.Error("Failed to update verification code", "email", req.Email, "error", err)
		return fmt.Errorf("failed to update verification code")
	}

	// Update user's email verification status
	userEntity, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		uc.logger.Error("User not found during email verification", "email", req.Email, "error", err)
		return fmt.Errorf("user not found")
	}

	userEntity.EmailVerified = true
	userEntity.UpdatedAt = time.Now()

	if _, err := uc.userRepo.Update(ctx, userEntity); err != nil {
		uc.logger.Error("Failed to update user email verification status", "user_id", userEntity.ID, "error", err)
		return fmt.Errorf("failed to update user")
	}

	uc.logger.Info("Email verification successful", "user_id", userEntity.ID, "email", req.Email)
	return nil
}

// Helper methods

func (uc *UseCase) generateTokens(userID primitive.ObjectID) (accessToken, refreshToken string, err error) {
	// For now, generate simple tokens. In a real implementation, use JWT
	accessToken = fmt.Sprintf("access_%s_%d", userID.Hex(), time.Now().Unix())
	refreshToken = fmt.Sprintf("refresh_%s_%d", userID.Hex(), time.Now().Unix())
	return accessToken, refreshToken, nil
}

func (uc *UseCase) saveRefreshToken(ctx context.Context, token string, userID primitive.ObjectID) error {
	refreshTokenEntity := &auth.RefreshToken{
		ID:        primitive.NewObjectID(),
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(auth.RefreshTokenExpiration),
		IsRevoked: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return uc.authRepo.SaveRefreshToken(ctx, refreshTokenEntity)
}

func (uc *UseCase) generateVerificationCode() (string, error) {
	// Generate a 6-digit verification code
	code := ""
	for i := 0; i < 6; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += n.String()
	}
	return code, nil
}
