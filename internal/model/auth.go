// Package auth contains the authentication model entities and logic.
package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// LoginVerificationCode represents a verification code for email/phone verification.
type LoginVerificationCode struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email     string             `json:"email" bson:"email"`
	Code      string             `json:"code" bson:"code"`
	IsUsed    bool               `json:"isUsed" bson:"is_used"`
	CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
}

// LoginRequest represents a passwordless login request (matches Python LoginBodySchema).
type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// RegisterVerificationCode represents a verification code for email/phone verification.
type RegisterVerificationCode struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FullName  string             `json:"fullName" bson:"full_name"`
	Email     string             `json:"email" bson:"email"`
	Code      string             `json:"code" bson:"code"`
	IsUsed    bool               `json:"isUsed" bson:"is_used"`
	CreatedAt time.Time          `json:"createdAt" bson:"created_at"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updated_at"`
}

// RegisterRequest represents a passwordless registration request (matches Python RegisterBodySchema).
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	FullName string `json:"fullName" validate:"required,min=2,max=100"`
}

// VerifyEmailRequest represents an email verification request (matches Python VerifyBodySchema).
type VerifyEmailRequest struct {
	CodeRequestID string `json:"codeRequestId" validate:"required"`
	Code          string `json:"code" validate:"required,len=6"`
}

// ForgotPasswordRequest represents a forgot password request.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents a password reset request.
type ResetPasswordRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Code     string `json:"code" validate:"required,len=6"`
	Password string `json:"newPassword" validate:"required,min=6"`
}

// TelegramAuthRequest represents a Telegram authentication request.
type TelegramAuthRequest struct {
	TelegramID       int64  `json:"tgId" validate:"required"`
	TelegramUsername string `json:"tgUsername,omitempty"`
	TelegramChatID   int64  `json:"tgChatId" validate:"required"`
	FullName         string `json:"full_name" validate:"required,min=2,max=100"`
}

// AuthInitResponse represents initial auth response for login/register (matches Python AuthResponseSchema).
type AuthInitResponse struct {
	CodeRequestID string `json:"codeRequestId"`
}

// VerifyResponse represents verification response (matches Python VerifyResponseSchema).
type VerifyResponse struct {
	AccessToken string `json:"accessToken"`
}

// PendingAuthResponse represents pending auth status (matches Python VerifyWithPendingFieldSchema).
type PendingAuthResponse struct {
	IsPending   bool   `json:"isPending"`
	AccessToken string `json:"accessToken,omitempty"`
}

// TelegramAuthResponse represents Telegram auth initiation (matches Python AuthTGResponseSchema).
type TelegramAuthResponse struct {
	AuthRequestID string `json:"authRequestId"`
	TgUrl         string `json:"tgUrl"`
}

// UserInfo represents user information in auth response.
type UserInfo struct {
	ID          primitive.ObjectID `json:"id"`
	Email       string             `json:"email,omitempty"`
	FullName    string             `json:"fullName"`
	HasEmail    bool               `json:"hasEmail"`
	HasTelegram bool               `json:"hasTelegram"`
}

// CreateEmployeeInvitationRequest represents a request to create an employee invitation (matches Python CreateEmployeeSchema).
type CreateEmployeeInvitationRequest struct {
	Email        string             `json:"email" validate:"required,email"`
	FullName     string             `json:"fullName" validate:"required,min=2,max=100"`
	RestaurantID primitive.ObjectID `json:"restaurantId" validate:"required"`
	Role         string             `json:"role" validate:"required,oneof=admin employee owner"`
}

// EmployeeRegistrationResponse represents employee registration information.
type EmployeeRegistrationResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// User profile management requests/responses (for /user/me endpoints)

// UpdateMeRequest represents a request to update current user's profile (matches Python PersonalCabinetUpdateSchema).
type UpdateMeRequest struct {
	FullName string `json:"fullName,omitempty" validate:"omitempty,min=2,max=100"`
}

// EmailUpdateRequest represents a request to update email (matches Python EmailUpdateRequestSchema).
type EmailUpdateRequest struct {
	NewEmail string `json:"newEmail" validate:"required,email"`
}

// EmailUpdateResponse represents response for email update initiation (matches Python AuthResponseSchema).
type EmailUpdateResponse struct {
	CodeRequestID string `json:"codeRequestId"`
}

// EmailVerifyRequest represents a request to verify new email (matches Python VerifyBodySchema).
type EmailVerifyRequest struct {
	CodeRequestID string `json:"codeRequestId" validate:"required"`
	Code          string `json:"code" validate:"required,len=6"`
}

// TelegramConnectResponse represents response for Telegram connection initiation (matches Python TelegramConnectionResponseSchema).
type TelegramConnectResponse struct {
	ConnectionRequestID string `json:"connectionRequestId"`
	TgUrl               string `json:"tgUrl"`
}

// TelegramConnectStatusResponse represents Telegram connection status check (matches Python VerifyWithPendingFieldSchema).
type TelegramConnectStatusResponse struct {
	IsPending bool `json:"isPending"`
	Success   bool `json:"success,omitempty"`
}

// Business logic methods

// IsExpired checks if the verification code has expired.
func (vc *LoginVerificationCode) IsExpired() bool {
	return time.Now().After(vc.CreatedAt.Add(AuthRequestExpiration))
}

// IsValid checks if the verification code is valid (not expired and not used).
func (vc *LoginVerificationCode) IsValid() bool {
	return !vc.IsExpired() && !vc.IsUsed
}

// MarkAsUsed marks the verification code as used.
func (vc *LoginVerificationCode) MarkAsUsed() {
	vc.IsUsed = true
	vc.UpdatedAt = time.Now()
}

// EmployeeRegistrationCode represents an employee invitation code for registration.
type EmployeeRegistrationCode struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	EmployeeID   *primitive.ObjectID `json:"employeeId,omitempty" bson:"employee_id,omitempty"`
	Email        string              `json:"email" bson:"email"`
	FullName     string              `json:"fullName" bson:"full_name"`
	RestaurantID primitive.ObjectID  `json:"restaurantId" bson:"restaurant_id"`
	Role         string              `json:"role" bson:"role"`
	Code         string              `json:"code" bson:"code"`
	CreatedAt    time.Time           `json:"createdAt" bson:"created_at"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updated_at"`
	IsUsed       bool                `json:"isUsed" bson:"is_used"`
}

// TelegramVerificationCode represents a Telegram authentication request code.
type TelegramVerificationCode struct {
	ID         primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	EmployeeID *primitive.ObjectID `json:"employeeId,omitempty" bson:"employee_id,omitempty"`
	CreatedAt  time.Time           `json:"createdAt" bson:"created_at"`
	UpdatedAt  time.Time           `json:"updatedAt" bson:"updated_at"`
	IsUsed     bool                `json:"isUsed" bson:"is_used"`
}

// Business logic methods for specific models

// IsExpired checks if the employee registration code has expired.
func (erc *EmployeeRegistrationCode) IsExpired() bool {
	return time.Now().After(erc.CreatedAt.Add(AuthRequestExpiration))
}

// IsCompleted checks if the employee registration code has been completed (employee_id is set).
func (erc *EmployeeRegistrationCode) IsCompleted() bool {
	return erc.EmployeeID != nil
}

// IsExpired checks if the Telegram verification code has expired.
func (tvc *TelegramVerificationCode) IsExpired() bool {
	return time.Now().After(tvc.CreatedAt.Add(AuthRequestExpiration))
}

// IsCompleted checks if the Telegram verification code has been completed (employee_id is set).
func (tvc *TelegramVerificationCode) IsCompleted() bool {
	return tvc.EmployeeID != nil
}

// IsExpired checks if the action request has expired.
func (ar *ActionRequest) IsExpired() bool {
	return time.Now().After(ar.CreatedAt.Add(AuthRequestExpiration))
}

// IsCompleted checks if the action request has been completed (employee_id is set).
func (ar *ActionRequest) IsCompleted() bool {
	return ar.EmployeeID != nil
}

// ActionRequest represents a general action request (adapted from Python ActionRequestSchema for Go/MongoDB).
type ActionRequest struct {
	ID             primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	EmployeeID     *primitive.ObjectID  `json:"employeeId,omitempty" bson:"employee_id,omitempty"`
	Action         string               `json:"action" bson:"action"`
	Email          string               `json:"email,omitempty" bson:"email,omitempty"`
	FullName       string               `json:"fullName,omitempty" bson:"full_name,omitempty"`
	ReservationID  *primitive.ObjectID  `json:"reservationId,omitempty" bson:"reservation_id,omitempty"`
	RestaurantID   *primitive.ObjectID `json:"restaurantId,omitempty" bson:"restaurant_id,omitempty"`
	Role           string               `json:"role,omitempty" bson:"role,omitempty"`
	Code           string               `json:"code,omitempty" bson:"code,omitempty"`
	CreatedAt      time.Time            `json:"createdAt" bson:"created_at"`
	UpdatedAt      time.Time            `json:"updatedAt" bson:"updated_at"`
	IsUsed         bool                 `json:"isUsed" bson:"is_used"`
}

// Constants for action types (matches Python CodeAction enum)
const (
	ActionRegister         = "REGISTER"
	ActionRegisterEmployee = "REGISTER_EMPLOYEE"
	ActionLogin            = "LOGIN"
	ActionTGAuth           = "TG_AUTH"
	ActionEmailBooking     = "EMAIL_BOOKING"
	ActionConnectTelegram  = "CONNECT_TELEGRAM"
	ActionUpdateEmail      = "UPDATE_EMAIL"
)

// UpdateEmployeeInvitationRequest represents a request to update an employee invitation.
type UpdateEmployeeInvitationRequest struct {
	InvitationID primitive.ObjectID  `json:"invitationId" validate:"required"`
	Email        string              `json:"email,omitempty" validate:"omitempty,email"`
	FullName     string              `json:"fullName,omitempty" validate:"omitempty,min=2,max=100"`
	RestaurantID *primitive.ObjectID `json:"restaurantId,omitempty"`
	Role         string              `json:"role,omitempty" validate:"omitempty,oneof=admin employee"`
}

// Constants for token expiration
const (
	AccessTokenExpiration = 24 * time.Hour
	AuthRequestExpiration = 5 * time.Minute // Matches Python AUTH_TIMEOUT_MINUTES
)
