package handler

import (
	"encoding/json"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"reservia-admin-api/internal/model"
	"reservia-admin-api/internal/service"
	"reservia-admin-api/pkg/logger"
	"reservia-admin-api/pkg/validator"
)

type AuthHandler struct {
	authService *service.AuthService
	validator   *validator.Validator
	logger      logger.Logger
}

func NewAuthHandler(authService *service.AuthService, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
		logger:      logger,
	}
}

// Login godoc
// @Summary      User login
// @Description  Initiate login process with email verification
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request body model.LoginRequest true "Login request"
// @Success      200  {object}  model.AuthInitResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request data using struct tags
	if err := h.validator.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call auth service
	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode login response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

// Register godoc
// @Summary      User registration
// @Description  Initiate registration process with email verification
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request body model.RegisterRequest true "Registration request"
// @Success      200  {object}  model.AuthInitResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      409  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request data using struct tags
	if err := h.validator.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call auth service
	response, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode register response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

// Verify godoc
// @Summary      Verify email code
// @Description  Verify email verification code
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        request body model.VerifyEmailRequest true "Verification request"
// @Success      200  {object}  model.VerifyResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/auth/verify [post]
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request data using struct tags
	if err := h.validator.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call auth service
	response, err := h.authService.VerifyEmail(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode verify response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

// RegisterEmployee godoc
// @Summary      Complete employee registration via invitation
// @Description  Complete employee registration using invitation link with code_request_id and code
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        code_request_id query string true "Code Request ID from invitation"
// @Param        code query string true "Invitation code"
// @Success      200  {object}  model.VerifyResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      401  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/auth/register/employee [get]
func (h *AuthHandler) RegisterEmployee(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	codeRequestID := r.URL.Query().Get("code_request_id")
	code := r.URL.Query().Get("code")

	// Validate required parameters
	if codeRequestID == "" {
		http.Error(w, "code_request_id parameter is required", http.StatusBadRequest)
		return
	}
	if code == "" {
		http.Error(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	// Call auth service to complete employee registration
	response, err := h.authService.RegisterEmployee(r.Context(), codeRequestID, code)
	if err != nil {
		h.logger.Error("Failed to complete employee registration", "code_request_id", codeRequestID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode employee registration response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

// AuthorizeTelegram godoc
// @Summary      Authorize via Telegram (Admin Bot)
// @Description  Initiate Telegram authorization process for admin bot
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Success      200  {object}  model.TelegramAuthResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/auth/tg [get]
func (h *AuthHandler) AuthorizeTelegram(w http.ResponseWriter, r *http.Request) {
	// Call auth service to create Telegram authorization request
	response, err := h.authService.AuthorizeTelegram(r.Context())
	if err != nil {
		h.logger.Error("Failed to create Telegram authorization request", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

// VerifyTelegram godoc
// @Summary      Verify Telegram authorization
// @Description  Check status of Telegram authorization request from admin bot
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Param        requestID path string true "Request ID"
// @Success      200  {object}  model.PendingAuthResponse
// @Failure      404  {object}  model.ErrorResponse
// @Failure      500  {object}  model.ErrorResponse
// @Router       /api/admin/auth/tg/{requestID} [get]
func (h *AuthHandler) VerifyTelegram(w http.ResponseWriter, r *http.Request) {
	// Extract request ID from URL parameter
	requestID := chi.URLParam(r, "requestID")
	if requestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	// Call auth service to verify Telegram authorization
	response, err := h.authService.VerifyTelegram(r.Context(), requestID)
	if err != nil {
		h.logger.Error("Failed to verify Telegram authorization", "request_id", requestID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}
