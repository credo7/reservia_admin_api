package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/config"
	"github.com/reservia/api/pkg/logger"
	"github.com/reservia/api/pkg/rabbitmq"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthService handles authentication business logic.
type AuthService struct {
	authRepo      repository.AuthRepository
	employeeRepo  repository.EmployeeRepository
	emailProducer *rabbitmq.Producer
	queueName     string
	config        *config.Config
	logger        logger.Logger
}

// NewAuthService creates a new auth service.
func NewAuthService(authRepo repository.AuthRepository, employeeRepo repository.EmployeeRepository, emailProducer *rabbitmq.Producer, queueName string, cfg *config.Config, logger logger.Logger) *AuthService {
	return &AuthService{
		authRepo:      authRepo,
		employeeRepo:  employeeRepo,
		emailProducer: emailProducer,
		queueName:     queueName,
		config:        cfg,
		logger:        logger,
	}
}

// Login initiates the passwordless login process for employees by sending a verification code.
func (as *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthInitResponse, error) {
	as.logger.Info("Attempting login", "email", req.Email)

	// Find employee by email
	_, err := as.employeeRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		as.logger.Error("Employee not found during login", "email", req.Email, "error", err)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate verification code
	verificationCode, err := as.generateVerificationCode()
	if err != nil {
		as.logger.Error("Failed to generate verification code", "error", err)
		return nil, fmt.Errorf("failed to generate verification code")
	}

	// Save verification code
	codeEntity := &model.LoginVerificationCode{
		ID:        primitive.NewObjectID(),
		Email:     req.Email,
		Code:      verificationCode,
		IsUsed:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := as.authRepo.CreateLoginVerificationCode(ctx, codeEntity); err != nil {
		as.logger.Error("Failed to save verification code", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to save verification code")
	}

	// Send email verification code via RabbitMQ
	if as.emailProducer != nil {
		if err := as.emailProducer.SendAuthEmail(as.queueName, req.Email, verificationCode); err != nil {
			as.logger.Error("Failed to send email verification", "email", req.Email, "error", err)
			// Note: We don't return error here as the verification code is saved and user can still verify manually
		} else {
			as.logger.Info("Email verification task sent", "email", req.Email)
		}
	}

	as.logger.Info("Login verification code generated", "email", req.Email, "code_request_id", codeEntity.ID.Hex())

	return &model.AuthInitResponse{
		CodeRequestID: codeEntity.ID.Hex(),
	}, nil
}

// Register creates a new employee account.
func (as *AuthService) Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthInitResponse, error) {
	as.logger.Info("Attempting registration", "email", req.Email)

	// Check if employee already exists
	existingEmployee, err := as.employeeRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingEmployee != nil {
		as.logger.Error("Employee already exists", "email", req.Email)
		return nil, fmt.Errorf("employee already exists")
	}

	// Generate verification code
	verificationCode, err := as.generateVerificationCode()
	if err != nil {
		as.logger.Error("Failed to generate verification code", "error", err)
		return nil, fmt.Errorf("failed to generate verification code")
	}

	// Save verification code
	codeEntity := &model.RegisterVerificationCode{
		ID:        primitive.NewObjectID(),
		FullName:  req.FullName,
		Email:     req.Email,
		Code:      verificationCode,
		IsUsed:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := as.authRepo.CreateRegisterVerificationCode(ctx, codeEntity); err != nil {
		as.logger.Error("Failed to save verification code", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to save verification code")
	}

	// Send employee registration email via RabbitMQ
	if as.emailProducer != nil {
		if err := as.emailProducer.SendEmployeeRegisterEmail(as.queueName, req.Email, req.FullName, "", verificationCode); err != nil {
			as.logger.Error("Failed to send employee registration email", "email", req.Email, "error", err)
			// Note: We don't return error here as the verification code is saved and user can still verify manually
		} else {
			as.logger.Info("Employee registration email task sent", "email", req.Email)
		}
	}

	as.logger.Info("Registration verification code generated", "email", req.Email, "code_request_id", codeEntity.ID.Hex())

	return &model.AuthInitResponse{
		CodeRequestID: codeEntity.ID.Hex(),
	}, nil
}

// VerifyEmail verifies an employee's email address with a verification code and returns access token.
func (as *AuthService) VerifyEmail(ctx context.Context, req *model.VerifyEmailRequest) (*model.VerifyResponse, error) {
	as.logger.Info("Attempting email verification", "code_request_id", req.CodeRequestID)

	// Convert CodeRequestID to ObjectID
	codeRequestID, err := primitive.ObjectIDFromHex(req.CodeRequestID)
	if err != nil {
		as.logger.Error("Invalid code request ID format", "code_request_id", req.CodeRequestID, "error", err)
		return nil, fmt.Errorf("invalid code request ID")
	}

	// Try to find as LoginVerificationCode first
	loginCode, err := as.authRepo.GetLoginVerificationCodeByID(ctx, codeRequestID)
	if err == nil {
		// Handle login verification
		return as.handleLoginVerification(ctx, loginCode, req.Code)
	}

	// Try to find as RegisterVerificationCode
	registerCode, err := as.authRepo.GetRegisterVerificationCodeByID(ctx, codeRequestID)
	if err == nil {
		// Handle registration verification
		return as.handleRegisterVerification(ctx, registerCode, req.Code)
	}

	as.logger.Error("Verification code not found", "code_request_id", req.CodeRequestID)
	return nil, fmt.Errorf("invalid verification code")
}

func (as *AuthService) handleLoginVerification(ctx context.Context, codeEntity *model.LoginVerificationCode, providedCode string) (*model.VerifyResponse, error) {
	// Verify the provided code matches
	if codeEntity.Code != providedCode {
		as.logger.Error("Verification code mismatch", "code_request_id", codeEntity.ID.Hex())
		return nil, fmt.Errorf("invalid verification code")
	}

	// Check if code is valid
	if !codeEntity.IsValid() {
		as.logger.Error("Verification code is invalid or expired", "code_request_id", codeEntity.ID.Hex())
		return nil, fmt.Errorf("verification code is invalid or expired")
	}

	// Mark code as used
	codeEntity.MarkAsUsed()
	if err := as.authRepo.UpdateLoginVerificationCode(ctx, codeEntity); err != nil {
		as.logger.Error("Failed to mark verification code as used", "code_request_id", codeEntity.ID.Hex(), "error", err)
		return nil, fmt.Errorf("failed to process verification code")
	}

	// Get employee by email
	employeeEntity, err := as.employeeRepo.GetByEmail(ctx, codeEntity.Email)
	if err != nil {
		as.logger.Error("Employee not found during email verification", "email", codeEntity.Email, "error", err)
		return nil, fmt.Errorf("employee not found")
	}

	// Update employee's updated timestamp
	employeeEntity.UpdatedAt = time.Now()
	if err := as.employeeRepo.Update(ctx, employeeEntity); err != nil {
		as.logger.Error("Failed to update employee", "employee_id", employeeEntity.ID, "error", err)
		return nil, fmt.Errorf("failed to update employee")
	}

	// Generate access token after successful verification
	accessToken, err := as.generateToken(employeeEntity.ID)
	if err != nil {
		as.logger.Error("Failed to generate tokens after verification", "employee_id", employeeEntity.ID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens")
	}

	as.logger.Info("Login verification successful", "employee_id", employeeEntity.ID, "email", codeEntity.Email)
	return &model.VerifyResponse{
		AccessToken: accessToken,
	}, nil
}

func (as *AuthService) handleRegisterVerification(ctx context.Context, codeEntity *model.RegisterVerificationCode, providedCode string) (*model.VerifyResponse, error) {
	// Verify the provided code matches
	if codeEntity.Code != providedCode {
		as.logger.Error("Verification code mismatch", "code_request_id", codeEntity.ID.Hex())
		return nil, fmt.Errorf("invalid verification code")
	}

	// Check if code is already used
	if codeEntity.IsUsed {
		as.logger.Error("Verification code already used", "code_request_id", codeEntity.ID.Hex())
		return nil, fmt.Errorf("verification code already used")
	}

	// Mark code as used
	codeEntity.IsUsed = true
	codeEntity.UpdatedAt = time.Now()
	if err := as.authRepo.UpdateRegisterVerificationCode(ctx, codeEntity); err != nil {
		as.logger.Error("Failed to mark verification code as used", "code_request_id", codeEntity.ID.Hex(), "error", err)
		return nil, fmt.Errorf("failed to process verification code")
	}

	// Create new employee
	newEmployee := &model.Employee{
		ID:        primitive.NewObjectID(),
		FullName:  codeEntity.FullName,
		Email:     codeEntity.Email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := as.employeeRepo.Create(ctx, newEmployee); err != nil {
		as.logger.Error("Failed to create employee during registration", "email", codeEntity.Email, "error", err)
		return nil, fmt.Errorf("failed to create employee")
	}

	// Generate access token after successful registration
	accessToken, err := as.generateToken(newEmployee.ID)
	if err != nil {
		as.logger.Error("Failed to generate tokens after registration", "employee_id", newEmployee.ID, "error", err)
		return nil, fmt.Errorf("failed to generate tokens")
	}

	as.logger.Info("Registration verification successful", "employee_id", newEmployee.ID, "email", codeEntity.Email)
	return &model.VerifyResponse{
		AccessToken: accessToken,
	}, nil
}

// AuthorizeTelegram creates a Telegram authorization request (matches Python authorize_tg).
func (as *AuthService) AuthorizeTelegram(ctx context.Context) (*model.TelegramAuthResponse, error) {
	as.logger.Info("Creating Telegram authorization request")

	// Create Telegram verification code
	tgCode := &model.TelegramVerificationCode{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsUsed:    false,
	}

	// Save Telegram verification code to database
	if err := as.authRepo.CreateTelegramVerificationCode(ctx, tgCode); err != nil {
		as.logger.Error("Failed to create Telegram verification code", "error", err)
		return nil, fmt.Errorf("failed to create Telegram auth request")
	}

	// Generate Telegram bot URL (matches Python implementation)
	tgUrl := fmt.Sprintf("https://t.me/%s?start=action_request_id=%s",
		as.config.Telegram.AdminBotUsername, tgCode.ID.Hex())

	as.logger.Info("Telegram authorization request created", "request_id", tgCode.ID.Hex())
	as.logger.Debug("Telegram authorization URL", "url", tgUrl)

	return &model.TelegramAuthResponse{
		AuthRequestID: tgCode.ID.Hex(),
		TgUrl:         tgUrl,
	}, nil
}

// VerifyTelegram verifies the status of a Telegram authorization request (matches Python authorize_tg_verify).
func (as *AuthService) VerifyTelegram(ctx context.Context, requestID string) (*model.PendingAuthResponse, error) {
	as.logger.Info("Verifying Telegram authorization request", "request_id", requestID)

	// Get Telegram verification code by ID
	tgCode, err := as.authRepo.GetTelegramVerificationCode(ctx, requestID)
	if err != nil {
		as.logger.Error("Telegram verification code not found", "request_id", requestID, "error", err)
		return nil, fmt.Errorf("invalid request ID")
	}

	// Note: No action type validation needed for TelegramVerificationCode as it's specific to Telegram auth

	// Check if request has expired
	if tgCode.IsExpired() {
		as.logger.Warn("Telegram auth request has expired", "request_id", requestID)
		return nil, fmt.Errorf("authorization request has expired")
	}

	// Check if user has completed the authorization (employee_id is set)
	if !tgCode.IsCompleted() {
		as.logger.Info("Telegram auth request is still pending", "request_id", requestID)
		return &model.PendingAuthResponse{
			IsPending: true,
		}, nil
	}

	// Prevent reuse - check if already used
	if tgCode.IsUsed {
		as.logger.Warn("Telegram auth request already used", "request_id", requestID)
		return nil, fmt.Errorf("authorization request was already used")
	}

	// Mark as used to prevent reuse
	tgCode.IsUsed = true
	tgCode.UpdatedAt = time.Now()
	if err := as.authRepo.UpdateTelegramVerificationCode(ctx, tgCode); err != nil {
		as.logger.Error("Failed to mark Telegram verification code as used", "request_id", requestID, "error", err)
		return nil, fmt.Errorf("failed to complete authorization")
	}

	// Generate access token
	accessToken, err := as.generateToken(*tgCode.EmployeeID)
	if err != nil {
		as.logger.Error("Failed to generate tokens after Telegram auth", "employee_id", tgCode.EmployeeID, "error", err)
		return nil, fmt.Errorf("failed to generate access token")
	}

	as.logger.Info("Telegram authorization completed successfully", "request_id", requestID, "employee_id", tgCode.EmployeeID)

	return &model.PendingAuthResponse{
		IsPending:   false,
		AccessToken: accessToken,
	}, nil
}

// Helper methods

func (as *AuthService) generateToken(employeeID primitive.ObjectID) (accessToken string, err error) {
	now := time.Now()

	// Generate access token (JWT)
	accessClaims := jwt.MapClaims{
		"sub":  employeeID.Hex(),
		"iat":  now.Unix(),
		"exp":  now.Add(as.config.Auth.TokenExpiry).Unix(),
		"type": "access",
	}

	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString([]byte(as.config.Auth.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return accessToken, nil
}

func (as *AuthService) generateVerificationCode() (string, error) {
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

// RegisterEmployee handles employee registration via invitation link (matches Python register_employee).
func (as *AuthService) RegisterEmployee(ctx context.Context, codeRequestID, code string) (*model.VerifyResponse, error) {
	as.logger.Info("Attempting employee registration via invitation", "code_request_id", codeRequestID)

	// Get employee registration code by ID (matches Python ActionRequestService.get)
	empRegCode, err := as.authRepo.GetEmployeeRegistrationCode(ctx, codeRequestID)
	if err != nil {
		as.logger.Error("Employee registration code not found", "code_request_id", codeRequestID, "error", err)
		return nil, fmt.Errorf("invalid invitation link")
	}
	if empRegCode == nil {
		as.logger.Error("Employee registration code is nil", "code_request_id", codeRequestID)
		return nil, fmt.Errorf("invalid invitation link")
	}

	// Verify the provided code matches (matches Python code_request.code != code)
	if empRegCode.Code != code {
		as.logger.Warn("Wrong employee registration code", "code_request_id", codeRequestID)
		return nil, fmt.Errorf("invalid invitation code")
	}

	// Check if request has expired (matches Python cls._check_code_request_expiration)
	if empRegCode.IsExpired() {
		as.logger.Warn("Employee registration invitation has expired", "code_request_id", codeRequestID)
		return nil, fmt.Errorf("invitation has expired")
	}

	// Check if already used (implicit in Python through expiration/validation)
	if empRegCode.IsUsed {
		as.logger.Warn("Employee registration invitation already used", "code_request_id", codeRequestID)
		return nil, fmt.Errorf("invitation was already used")
	}

	// Create restaurant associations from invitation (matches Python UserRestaurantSchema creation)
	newRestaurants := make([]model.EmployeeRestaurant, 0, len(empRegCode.RestaurantsIDs))
	for _, restaurantID := range empRegCode.RestaurantsIDs {
		newRestaurants = append(newRestaurants, model.EmployeeRestaurant{
			RestaurantID: restaurantID,
			Role:         model.Role(empRegCode.Role),
		})
	}

	// Check if employee already exists
	existingEmployee, err := as.employeeRepo.GetByEmail(ctx, empRegCode.Email)
	var targetEmployee *model.Employee

	if err == nil && existingEmployee != nil {
		// Employee exists - add new restaurants to existing employee
		as.logger.Info("Adding restaurants to existing employee", "email", empRegCode.Email, "employee_id", existingEmployee.ID)

		// Merge existing restaurants with new ones (avoid duplicates)
		existingRestaurantMap := make(map[primitive.ObjectID]bool)
		for _, existingRest := range existingEmployee.Restaurants {
			existingRestaurantMap[existingRest.RestaurantID] = true
		}

		// Add only new restaurants that don't already exist
		for _, newRest := range newRestaurants {
			if !existingRestaurantMap[newRest.RestaurantID] {
				existingEmployee.Restaurants = append(existingEmployee.Restaurants, newRest)
			}
		}

		// Update employee with new restaurants
		existingEmployee.UpdatedAt = time.Now()
		if err := as.employeeRepo.Update(ctx, existingEmployee); err != nil {
			as.logger.Error("Failed to update existing employee with new restaurants", "email", empRegCode.Email, "error", err)
			return nil, fmt.Errorf("failed to update employee")
		}

		targetEmployee = existingEmployee
		as.logger.Info("Successfully added restaurants to existing employee", "employee_id", existingEmployee.ID, "email", empRegCode.Email)
	} else {
		// Employee doesn't exist - create new employee
		as.logger.Info("Creating new employee from invitation", "email", empRegCode.Email)

		newEmployee := &model.Employee{
			ID:          primitive.NewObjectID(),
			FullName:    empRegCode.FullName,
			Email:       empRegCode.Email,
			Restaurants: newRestaurants,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := as.employeeRepo.Create(ctx, newEmployee); err != nil {
			as.logger.Error("Failed to create employee during invitation registration", "email", empRegCode.Email, "error", err)
			return nil, fmt.Errorf("failed to create employee")
		}

		targetEmployee = newEmployee
		as.logger.Info("Successfully created new employee", "employee_id", newEmployee.ID, "email", empRegCode.Email)
	}

	// Mark invitation as used (matches Python ActionRequestService.set_employee_id_and_mark_as_used)
	empRegCode.IsUsed = true
	empRegCode.EmployeeID = &targetEmployee.ID
	empRegCode.UpdatedAt = time.Now()
	if err := as.authRepo.UpdateEmployeeRegistrationCode(ctx, empRegCode); err != nil {
		as.logger.Error("Failed to mark invitation as used", "code_request_id", codeRequestID, "error", err)
		return nil, fmt.Errorf("failed to complete registration")
	}

	// Generate access token for immediate login (matches Python create_access_token)
	accessToken, err := as.generateToken(targetEmployee.ID)
	if err != nil {
		as.logger.Error("Failed to generate tokens after employee registration", "employee_id", targetEmployee.ID, "error", err)
		return nil, fmt.Errorf("failed to generate access token")
	}

	as.logger.Info("Employee registration successful", "employee_id", targetEmployee.ID, "email", empRegCode.Email)
	return &model.VerifyResponse{
		AccessToken: accessToken,
	}, nil
}

// ValidateToken validates and extracts employee ID from JWT access token
func (as *AuthService) ValidateToken(tokenString string) (primitive.ObjectID, error) {
	// Parse JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(as.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid token: %w", err)
	}

	// Validate token and extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Verify token type
		if tokenType, exists := claims["type"]; !exists || tokenType != "access" {
			return primitive.NilObjectID, fmt.Errorf("invalid token type")
		}

		// Extract employee ID
		employeeIDStr, ok := claims["sub"].(string)
		if !ok {
			return primitive.NilObjectID, fmt.Errorf("invalid employee ID in token")
		}

		employeeID, err := primitive.ObjectIDFromHex(employeeIDStr)
		if err != nil {
			return primitive.NilObjectID, fmt.Errorf("invalid employee ID format: %w", err)
		}

		// Verify employee exists
		_, err = as.employeeRepo.GetByID(context.Background(), employeeID)
		if err != nil {
			return primitive.NilObjectID, fmt.Errorf("employee not found")
		}

		return employeeID, nil
	}

	return primitive.NilObjectID, fmt.Errorf("invalid token claims")
}

// InviteEmployee creates an employee invitation (matches Python EmployeeService.add -> _send_employee_invitation).
func (as *AuthService) InviteEmployee(ctx context.Context, req *model.CreateEmployeeInvitationRequest) (*model.AuthInitResponse, error) {
	as.logger.Info("Creating employee invitation", "email", req.Email, "restaurants", req.RestaurantsIDs)

	// Validate role
	role := model.Role(req.Role)
	if !role.IsValid() {
		as.logger.Error("Invalid role for employee invitation", "role", req.Role)
		return nil, fmt.Errorf("invalid role")
	}

	// Prevent OWNER role assignment (matches Python validation)
	if role == model.RoleOwner {
		as.logger.Error("Cannot assign owner role through invitation", "email", req.Email)
		return nil, fmt.Errorf("owner role not allowed for invitations")
	}

	// Generate invitation code (matches Python generate_code(16))
	invitationCode, err := as.generateInvitationCode()
	if err != nil {
		as.logger.Error("Failed to generate invitation code", "error", err)
		return nil, fmt.Errorf("failed to generate invitation code")
	}

	// Create employee registration code (matches Python CreateActionRequestSchema with REGISTER_EMPLOYEE action)
	empRegCode := &model.EmployeeRegistrationCode{
		ID:             primitive.NewObjectID(),
		Email:          req.Email,
		FullName:       req.FullName,
		RestaurantsIDs: req.RestaurantsIDs,
		Role:           req.Role,
		Code:           invitationCode,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		IsUsed:         false,
	}

	// Save invitation to database
	if err := as.authRepo.CreateEmployeeRegistrationCode(ctx, empRegCode); err != nil {
		as.logger.Error("Failed to save employee invitation", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to create invitation")
	}

	// Send invitation email via RabbitMQ (matches Python send_employee_register_email_task.delay)
	if as.emailProducer != nil {
		if err := as.emailProducer.SendEmployeeRegisterEmail(as.queueName, req.Email, req.FullName, empRegCode.ID.Hex(), invitationCode); err != nil {
			as.logger.Error("Failed to send employee invitation email", "email", req.Email, "error", err)
			// Note: We don't return error here as the invitation is saved and can be resent
		} else {
			as.logger.Info("Employee invitation email task sent", "email", req.Email)
		}
	}

	as.logger.Info("Employee invitation created successfully", "email", req.Email, "invitation_id", empRegCode.ID.Hex())

	return &model.AuthInitResponse{
		CodeRequestID: empRegCode.ID.Hex(),
	}, nil
}

// generateInvitationCode generates a 16-character invitation code (matches Python generate_code(16)).
func (as *AuthService) generateInvitationCode() (string, error) {
	// Characters to use for invitation code (alphanumeric)
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	code := ""

	for i := 0; i < 16; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		code += string(chars[n.Int64()])
	}

	return code, nil
}

// ConnectTelegram creates a Telegram connection request for an employee.
func (as *AuthService) ConnectTelegram(ctx context.Context, employeeID primitive.ObjectID) (*model.TelegramConnectionResponse, error) {
	as.logger.Info("Creating Telegram connection request", "employee_id", employeeID)

	// Create Telegram verification code for connection
	tgCode := &model.TelegramVerificationCode{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsUsed:    false,
	}

	// Save Telegram verification code to database
	if err := as.authRepo.CreateTelegramVerificationCode(ctx, tgCode); err != nil {
		as.logger.Error("Failed to create Telegram verification code for connection", "error", err)
		return nil, fmt.Errorf("failed to create Telegram connection request")
	}

	// Generate Telegram bot URL for connection (similar to auth but for connection)
	tgUrl := fmt.Sprintf("https://t.me/%s?start=connect_request_id=%s",
		as.config.Telegram.AdminBotUsername, tgCode.ID.Hex())

	as.logger.Info("Telegram connection request created", "request_id", tgCode.ID.Hex(), "employee_id", employeeID)

	return &model.TelegramConnectionResponse{
		ConnectionRequestID: tgCode.ID.Hex(),
		TelegramURL:         tgUrl,
	}, nil
}

// CheckTelegramConnection checks the status of a Telegram connection request.
func (as *AuthService) CheckTelegramConnection(ctx context.Context, requestID string, employeeID primitive.ObjectID) (*model.TelegramConnectionVerifyResponse, error) {
	as.logger.Info("Checking Telegram connection status", "request_id", requestID, "employee_id", employeeID)

	// Get Telegram verification code by ID
	tgCode, err := as.authRepo.GetTelegramVerificationCode(ctx, requestID)
	if err != nil {
		as.logger.Error("Telegram connection request not found", "request_id", requestID, "error", err)
		return nil, fmt.Errorf("invalid connection request ID")
	}

	// Check if request has expired
	if tgCode.IsExpired() {
		as.logger.Warn("Telegram connection request has expired", "request_id", requestID)
		return &model.TelegramConnectionVerifyResponse{
			IsPending: false,
			Success:   false,
		}, nil
	}

	// Check if connection has been completed (employee_id is set)
	if !tgCode.IsCompleted() {
		as.logger.Info("Telegram connection request is still pending", "request_id", requestID)
		return &model.TelegramConnectionVerifyResponse{
			IsPending: true,
			Success:   false,
		}, nil
	}

	// Check if already used
	if tgCode.IsUsed {
		as.logger.Warn("Telegram connection request already used", "request_id", requestID)
		return &model.TelegramConnectionVerifyResponse{
			IsPending: false,
			Success:   false,
		}, nil
	}

	// Connection completed successfully - get both employees
	targetEmployee, err := as.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		as.logger.Error("Failed to get target employee for Telegram connection", "employee_id", employeeID, "error", err)
		return nil, fmt.Errorf("employee not found")
	}

	// Get the Telegram employee (the one who clicked the bot link)
	telegramEmployee, err := as.employeeRepo.GetByID(ctx, *tgCode.EmployeeID)
	if err != nil {
		as.logger.Error("Failed to get Telegram employee", "telegram_employee_id", *tgCode.EmployeeID, "error", err)
		return nil, fmt.Errorf("Telegram employee not found")
	}

	// Transfer Telegram data from bot employee to target employee
	targetEmployee.TelegramIsBot = telegramEmployee.TelegramIsBot
	targetEmployee.TelegramID = telegramEmployee.TelegramID
	targetEmployee.TelegramChatID = telegramEmployee.TelegramChatID
	targetEmployee.TelegramUsername = telegramEmployee.TelegramUsername
	targetEmployee.TelegramLang = telegramEmployee.TelegramLang
	targetEmployee.TelegramPremium = telegramEmployee.TelegramPremium
	targetEmployee.UpdatedAt = time.Now()

	if err := as.employeeRepo.Update(ctx, targetEmployee); err != nil {
		as.logger.Error("Failed to update target employee with Telegram data", "employee_id", employeeID, "error", err)
		return nil, fmt.Errorf("failed to connect Telegram account")
	}

	// Mark verification code as used
	tgCode.IsUsed = true
	tgCode.UpdatedAt = time.Now()
	if err := as.authRepo.UpdateTelegramVerificationCode(ctx, tgCode); err != nil {
		as.logger.Error("Failed to mark Telegram connection code as used", "request_id", requestID, "error", err)
		// Don't return error here as the connection was successful
	}

	as.logger.Info("Telegram connection completed successfully", "request_id", requestID, "target_employee_id", employeeID, "telegram_employee_id", *tgCode.EmployeeID, "telegram_id", *telegramEmployee.TelegramID)

	return &model.TelegramConnectionVerifyResponse{
		IsPending: false,
		Success:   true,
	}, nil
}

// DisconnectTelegram disconnects Telegram from an employee account.
func (as *AuthService) DisconnectTelegram(ctx context.Context, employeeID primitive.ObjectID) (*model.Employee, error) {
	as.logger.Info("Disconnecting Telegram account", "employee_id", employeeID)

	// Get employee
	employee, err := as.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		as.logger.Error("Failed to get employee for Telegram disconnection", "employee_id", employeeID, "error", err)
		return nil, fmt.Errorf("employee not found")
	}

	// Remove Telegram ID
	employee.TelegramID = nil
	employee.UpdatedAt = time.Now()
	if err := as.employeeRepo.Update(ctx, employee); err != nil {
		as.logger.Error("Failed to disconnect Telegram from employee", "employee_id", employeeID, "error", err)
		return nil, fmt.Errorf("failed to disconnect Telegram account")
	}

	as.logger.Info("Telegram account disconnected successfully", "employee_id", employeeID)
	return employee, nil
}

// GetPendingInvitations retrieves all pending employee invitations.
func (as *AuthService) GetPendingInvitations(ctx context.Context) ([]*model.EmployeeRegistrationCode, error) {
	as.logger.Info("Retrieving pending employee invitations")

	pendingCodes, err := as.authRepo.GetPendingEmployeeRegistrationCodes(ctx)
	if err != nil {
		as.logger.Error("Failed to retrieve pending invitations", "error", err)
		return nil, fmt.Errorf("failed to retrieve pending invitations")
	}

	as.logger.Info("Retrieved pending invitations", "count", len(pendingCodes))
	return pendingCodes, nil
}

// CancelInvitation cancels a pending employee invitation.
func (as *AuthService) CancelInvitation(ctx context.Context, invitationID string) error {
	as.logger.Info("Cancelling employee invitation", "invitation_id", invitationID)

	// Check if invitation exists and is still pending
	invitation, err := as.authRepo.GetEmployeeRegistrationCode(ctx, invitationID)
	if err != nil {
		as.logger.Error("Failed to retrieve invitation for cancellation", "invitation_id", invitationID, "error", err)
		return fmt.Errorf("invitation not found")
	}
	if invitation == nil {
		as.logger.Warn("Invitation not found for cancellation", "invitation_id", invitationID)
		return fmt.Errorf("invitation not found")
	}

	// Check if already used or expired
	if invitation.IsUsed {
		as.logger.Warn("Cannot cancel used invitation", "invitation_id", invitationID)
		return fmt.Errorf("invitation has already been used")
	}
	if invitation.IsExpired() {
		as.logger.Warn("Cannot cancel expired invitation", "invitation_id", invitationID)
		return fmt.Errorf("invitation has already expired")
	}

	// Delete the invitation
	if err := as.authRepo.DeleteEmployeeRegistrationCode(ctx, invitationID); err != nil {
		as.logger.Error("Failed to delete invitation", "invitation_id", invitationID, "error", err)
		return fmt.Errorf("failed to cancel invitation")
	}

	as.logger.Info("Successfully cancelled invitation", "invitation_id", invitationID, "email", invitation.Email)
	return nil
}

// ExtendInvitation extends the expiration of a pending invitation by regenerating it.
func (as *AuthService) ExtendInvitation(ctx context.Context, invitationID string) (*model.EmployeeRegistrationCode, error) {
	as.logger.Info("Extending employee invitation", "invitation_id", invitationID)

	// Get the existing invitation
	oldInvitation, err := as.authRepo.GetEmployeeRegistrationCode(ctx, invitationID)
	if err != nil {
		as.logger.Error("Failed to retrieve invitation for extension", "invitation_id", invitationID, "error", err)
		return nil, fmt.Errorf("invitation not found")
	}
	if oldInvitation == nil {
		as.logger.Warn("Invitation not found for extension", "invitation_id", invitationID)
		return nil, fmt.Errorf("invitation not found")
	}

	// Check if already used
	if oldInvitation.IsUsed {
		as.logger.Warn("Cannot extend used invitation", "invitation_id", invitationID)
		return nil, fmt.Errorf("invitation has already been used")
	}

	// Generate new invitation code
	newInvitationCode, err := as.generateInvitationCode()
	if err != nil {
		as.logger.Error("Failed to generate new invitation code for extension", "error", err)
		return nil, fmt.Errorf("failed to generate new invitation code")
	}

	// Create new invitation with extended time
	newInvitation := &model.EmployeeRegistrationCode{
		ID:             primitive.NewObjectID(),
		Email:          oldInvitation.Email,
		FullName:       oldInvitation.FullName,
		RestaurantsIDs: oldInvitation.RestaurantsIDs,
		Role:           oldInvitation.Role,
		Code:           newInvitationCode,
		CreatedAt:      time.Now(), // This extends the expiration
		UpdatedAt:      time.Now(),
		IsUsed:         false,
	}

	// Save the new invitation
	if err := as.authRepo.CreateEmployeeRegistrationCode(ctx, newInvitation); err != nil {
		as.logger.Error("Failed to save extended invitation", "email", oldInvitation.Email, "error", err)
		return nil, fmt.Errorf("failed to create extended invitation")
	}

	// Delete the old invitation
	if err := as.authRepo.DeleteEmployeeRegistrationCode(ctx, invitationID); err != nil {
		as.logger.Error("Failed to delete old invitation during extension", "invitation_id", invitationID, "error", err)
		// Don't return error here as the new invitation was created successfully
		as.logger.Warn("Old invitation was not deleted, but new invitation was created successfully")
	}

	// Send new invitation email via RabbitMQ
	if as.emailProducer != nil {
		if err := as.emailProducer.SendEmployeeRegisterEmail(as.queueName, oldInvitation.Email, oldInvitation.FullName, newInvitation.ID.Hex(), newInvitationCode); err != nil {
			as.logger.Error("Failed to send extended invitation email", "email", oldInvitation.Email, "error", err)
			// Note: We don't return error here as the invitation is saved and can be resent
		} else {
			as.logger.Info("Extended invitation email task sent", "email", oldInvitation.Email)
		}
	}

	as.logger.Info("Successfully extended invitation", "old_invitation_id", invitationID, "new_invitation_id", newInvitation.ID.Hex(), "email", oldInvitation.Email)
	return newInvitation, nil
}

// UpdateInvitation updates the details of a pending invitation.
func (as *AuthService) UpdateInvitation(ctx context.Context, invitationID string, updates *model.CreateEmployeeInvitationRequest) (*model.EmployeeRegistrationCode, error) {
	as.logger.Info("Updating employee invitation", "invitation_id", invitationID)

	// Get the existing invitation
	existingInvitation, err := as.authRepo.GetEmployeeRegistrationCode(ctx, invitationID)
	if err != nil {
		as.logger.Error("Failed to retrieve invitation for update", "invitation_id", invitationID, "error", err)
		return nil, fmt.Errorf("invitation not found")
	}
	if existingInvitation == nil {
		as.logger.Warn("Invitation not found for update", "invitation_id", invitationID)
		return nil, fmt.Errorf("invitation not found")
	}

	// Check if already used
	if existingInvitation.IsUsed {
		as.logger.Warn("Cannot update used invitation", "invitation_id", invitationID)
		return nil, fmt.Errorf("invitation has already been used")
	}

	// Check if expired
	if existingInvitation.IsExpired() {
		as.logger.Warn("Cannot update expired invitation", "invitation_id", invitationID)
		return nil, fmt.Errorf("invitation has already expired")
	}

	// Validate role if provided
	if updates.Role != "" {
		role := model.Role(updates.Role)
		if !role.IsValid() {
			as.logger.Error("Invalid role for invitation update", "role", updates.Role)
			return nil, fmt.Errorf("invalid role")
		}
		if role == model.RoleOwner {
			as.logger.Error("Cannot assign owner role through invitation update", "invitation_id", invitationID)
			return nil, fmt.Errorf("owner role not allowed for invitations")
		}
	}

	// Update the invitation fields
	existingInvitation.UpdatedAt = time.Now()
	if updates.Email != "" && updates.Email != existingInvitation.Email {
		existingInvitation.Email = updates.Email
	}
	if updates.FullName != "" && updates.FullName != existingInvitation.FullName {
		existingInvitation.FullName = updates.FullName
	}
	if updates.Role != "" && updates.Role != existingInvitation.Role {
		existingInvitation.Role = updates.Role
	}
	if len(updates.RestaurantsIDs) > 0 {
		existingInvitation.RestaurantsIDs = updates.RestaurantsIDs
	}

	// Update in database
	if err := as.authRepo.UpdateEmployeeRegistrationCode(ctx, existingInvitation); err != nil {
		as.logger.Error("Failed to update invitation", "invitation_id", invitationID, "error", err)
		return nil, fmt.Errorf("failed to update invitation")
	}

	as.logger.Info("Successfully updated invitation", "invitation_id", invitationID, "email", existingInvitation.Email)
	return existingInvitation, nil
}
