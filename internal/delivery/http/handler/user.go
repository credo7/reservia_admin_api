// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/domain/user"
	userUseCase "github.com/reservia/api/internal/usecase/user"
	"github.com/reservia/api/pkg/logger"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userUseCase *userUseCase.UseCase
	logger      logger.Logger
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userUseCase *userUseCase.UseCase, logger logger.Logger) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
		logger:      logger,
	}
}

// CreateUser handles POST /users.
//
//	@Summary		Create a new user
//	@Description	Create a new user with the provided information
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		user.CreateUserRequest	true	"User creation data"
//	@Success		201		{object}	user.UserResponse		"User created successfully"
//	@Failure		400		{object}	map[string]string		"Invalid request body"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req user.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	createdUser, err := h.userUseCase.CreateUser(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create user", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	h.writeJSON(w, http.StatusCreated, createdUser.ToResponse())
}

// GetUser handles GET /users/{id}.
//
//	@Summary		Get user by ID
//	@Description	Retrieve a user by their unique ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"User ID"
//	@Success		200	{object}	user.UserResponse	"User found"
//	@Failure		400	{object}	map[string]string	"Invalid user ID"
//	@Failure		404	{object}	map[string]string	"User not found"
//	@Router			/users/{id} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userUseCase.GetUserByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err)
		h.writeError(w, http.StatusNotFound, "User not found")
		return
	}

	h.writeJSON(w, http.StatusOK, user.ToResponse())
}

// UpdateUser handles PUT /users/{id}.
//
//	@Summary		Update user
//	@Description	Update user information by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"User ID"
//	@Param			user	body		user.UpdateUserRequest	true	"User update data"
//	@Success		200		{object}	user.UserResponse		"User updated successfully"
//	@Failure		400		{object}	map[string]string		"Invalid request"
//	@Failure		404		{object}	map[string]string		"User not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/users/{id} [put]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req user.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedUser, err := h.userUseCase.UpdateUser(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update user", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	h.writeJSON(w, http.StatusOK, updatedUser.ToResponse())
}

// DeleteUser handles DELETE /users/{id}.
//
//	@Summary		Delete user
//	@Description	Delete a user by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"User ID"
//	@Success		204	{string}	string				"User deleted successfully"
//	@Failure		400	{object}	map[string]string	"Invalid user ID"
//	@Failure		404	{object}	map[string]string	"User not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/users/{id} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.userUseCase.DeleteUser(r.Context(), id); err != nil {
		h.logger.Error("Failed to delete user", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUsers handles GET /users.
//
//	@Summary		List users
//	@Description	Retrieve a list of users with pagination
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int						false	"Number of users to return (default: 10)"
//	@Param			offset	query		int						false	"Number of users to skip (default: 0)"
//	@Success		200		{object}	map[string]interface{}	"List of users with pagination info"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	users, err := h.userUseCase.ListUsers(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to list users", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	// Convert to response format
	responses := make([]*user.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"users":  responses,
		"limit":  limit,
		"offset": offset,
		"count":  len(responses),
	})
}

// Helper methods

func (h *UserHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *UserHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
