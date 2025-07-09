// Package auth contains the authentication domain entities and logic.
package auth

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VerificationCode represents a verification code for email/phone verification.
type VerificationCode struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email     string             `json:"email" bson:"email"`
	Code      string             `json:"code" bson:"code"`
	Purpose   string             `json:"purpose" bson:"purpose"` // "email_verification", "password_reset", etc.
	ExpiresAt time.Time          `json:"expires_at" bson:"expires_at"`
	IsUsed    bool               `json:"is_used" bson:"is_used"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// RefreshToken represents a refresh token for JWT authentication.
type RefreshToken struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Token     string             `json:"token" bson:"token"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	ExpiresAt time.Time          `json:"expires_at" bson:"expires_at"`
	IsRevoked bool               `json:"is_revoked" bson:"is_revoked"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Password string `json:"password" validate:"required,min=6"`
}

// VerifyEmailRequest represents an email verification request.
type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
}

// ForgotPasswordRequest represents a forgot password request.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a password reset request.
type ResetPasswordRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Code     string `json:"code" validate:"required,len=6"`
	Password string `json:"new_password" validate:"required,min=6"`
}

// TelegramAuthRequest represents a Telegram authentication request.
type TelegramAuthRequest struct {
	TelegramID       int64  `json:"tg_id" validate:"required"`
	TelegramUsername string `json:"tg_username,omitempty"`
	TelegramChatID   int64  `json:"tg_chat_id" validate:"required"`
	FullName         string `json:"full_name" validate:"required,min=2,max=100"`
}

// AuthResponse represents an authentication response.
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserInfo  `json:"user"`
}

// UserInfo represents user information in auth response.
type UserInfo struct {
	ID          primitive.ObjectID `json:"id"`
	Email       string             `json:"email,omitempty"`
	FullName    string             `json:"full_name"`
	HasEmail    bool               `json:"has_email"`
	HasTelegram bool               `json:"has_telegram"`
}

// RefreshTokenRequest represents a refresh token request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// EmployeeRegistrationRequest represents an employee registration request.
type EmployeeRegistrationRequest struct {
	RestaurantID primitive.ObjectID `json:"restaurant_id" validate:"required"`
	Email        string             `json:"email" validate:"required,email"`
	FullName     string             `json:"full_name" validate:"required,min=2,max=100"`
	Role         string             `json:"role" validate:"required,oneof=manager employee"`
}

// Business logic methods

// IsExpired checks if the verification code has expired.
func (vc *VerificationCode) IsExpired() bool {
	return time.Now().After(vc.ExpiresAt)
}

// IsValid checks if the verification code is valid (not expired and not used).
func (vc *VerificationCode) IsValid() bool {
	return !vc.IsExpired() && !vc.IsUsed
}

// MarkAsUsed marks the verification code as used.
func (vc *VerificationCode) MarkAsUsed() {
	vc.IsUsed = true
	vc.UpdatedAt = time.Now()
}

// IsExpired checks if the refresh token has expired.
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the refresh token is valid (not expired and not revoked).
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired() && !rt.IsRevoked
}

// Revoke revokes the refresh token.
func (rt *RefreshToken) Revoke() {
	rt.IsRevoked = true
	rt.UpdatedAt = time.Now()
}

// Constants for verification code purposes
const (
	PurposeEmailVerification = "email_verification"
	PurposePasswordReset     = "password_reset"
	PurposeEmailUpdate       = "email_update"
)

// Constants for token types
const (
	TokenTypeBearer = "Bearer"
)

// Constants for token expiration
const (
	AccessTokenExpiration      = 24 * time.Hour
	RefreshTokenExpiration     = 7 * 24 * time.Hour
	VerificationCodeExpiration = 15 * time.Minute
)
