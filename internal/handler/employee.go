// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/reservia/api/internal/middleware"
	"github.com/reservia/api/internal/model"
	"github.com/reservia/api/internal/service"
	"github.com/reservia/api/pkg/logger"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmployeeHandler handles employee-related HTTP requests.
type EmployeeHandler struct {
	employeeService *service.EmployeeService
	authService     *service.AuthService
	logger          logger.Logger
}

// NewEmployeeHandler creates a new employee handler.
func NewEmployeeHandler(employeeService *service.EmployeeService, authService *service.AuthService, logger logger.Logger) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService: employeeService,
		authService:     authService,
		logger:          logger,
	}
}

// InviteEmployee handles POST /employees/invite.
//
//	@Summary		Create employee invitation
//	@Description	Create an invitation for a new employee to join specified restaurants (matches Python POST /employees)
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			invitation	body		model.CreateEmployeeInvitationRequest	true	"Employee invitation data"
//	@Success		201			{object}	model.AuthInitResponse					"Invitation created successfully"
//	@Failure		400			{object}	map[string]string						"Invalid request body"
//	@Failure		500			{object}	map[string]string						"Internal server error"
//	@Router			/employees/invite [post]
//	@Security		BearerAuth
func (h *EmployeeHandler) InviteEmployee(w http.ResponseWriter, r *http.Request) {
	var req model.CreateEmployeeInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during invitation creation")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Check if employee has permission to manage employees for the specified restaurants
	if !employee.HasPermissionForRestaurants(model.PermissionManageEmployees, req.RestaurantsIDs) {
		h.logger.Warn("Employee attempted to invite to restaurants without permission",
			"employee_id", employee.ID,
			"permission", model.PermissionManageEmployees,
			"restaurants", req.RestaurantsIDs)
		h.writeError(w, http.StatusForbidden, "You don't have permission to invite employees to the specified restaurants")
		return
	}

	invitationResponse, err := h.authService.InviteEmployee(r.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create employee invitation", "error", err)
		if err.Error() == "invalid role" || err.Error() == "owner role not allowed for invitations" {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to create invitation")
		}
		return
	}

	h.logger.Info("Employee invitation created successfully",
		"inviter_id", employee.ID,
		"email", req.Email,
		"invitation_id", invitationResponse.CodeRequestID,
		"restaurants_count", len(req.RestaurantsIDs))
	h.writeJSON(w, http.StatusCreated, invitationResponse)
}

// GetMe handles GET /employees/me.
//
//	@Summary		Get current employee
//	@Description	Retrieve the currently authenticated employee's information
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.Employee	"Current employee information"
//	@Failure		401	{object}	map[string]string		"Unauthorized - no authenticated employee"
//	@Failure		404	{object}	map[string]string		"Employee not found"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/employees/me [get]
//	@Security		BearerAuth
func (h *EmployeeHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during GetMe request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	h.logger.Info("Successfully retrieved current employee", "employee_id", employee.ID)
	h.writeJSON(w, http.StatusOK, employee)
}

// GetEmployee handles GET /employees/{id}.
//
//	@Summary		Get employee by ID
//	@Description	Retrieve an employee by their unique ID (only if requester has access to employee's restaurants)
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string				true	"Employee ID"
//	@Success		200	{object}	model.Employee		"Employee found"
//	@Failure		400	{object}	map[string]string	"Invalid employee ID"
//	@Failure		403	{object}	map[string]string	"Access denied - no shared restaurants"
//	@Failure		404	{object}	map[string]string	"Employee not found"
//	@Router			/employees/{id} [get]
//	@Security		BearerAuth
func (h *EmployeeHandler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	requester, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during GetEmployee request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid employee ID")
		return
	}

	employee, err := h.employeeService.GetEmployeeByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get employee", "error", err)
		h.writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	// Check if requester has access to view this employee
	// They must share at least one restaurant or have manage employees permission
	requesterRestaurantIDs := requester.GetRestaurantIDs()
	employeeRestaurantIDs := employee.GetRestaurantIDs()

	// Check for shared restaurants
	hasSharedRestaurant := false
	for _, reqRestID := range requesterRestaurantIDs {
		for _, empRestID := range employeeRestaurantIDs {
			if reqRestID == empRestID {
				hasSharedRestaurant = true
				break
			}
		}
		if hasSharedRestaurant {
			break
		}
	}

	if !hasSharedRestaurant {
		h.logger.Warn("Employee attempted to access employee from different restaurants",
			"requester_id", requester.ID,
			"target_employee_id", employee.ID,
			"requester_restaurants", requesterRestaurantIDs,
			"target_restaurants", employeeRestaurantIDs)
		h.writeError(w, http.StatusForbidden, "Access denied - you don't have access to this employee")
		return
	}

	// Filter the employee's restaurants to only show those the requester has access to
	filteredEmployee := employee.FilterRestaurantsByAccess(requesterRestaurantIDs)

	h.logger.Info("Employee accessed successfully",
		"requester_id", requester.ID,
		"target_employee_id", employee.ID,
		"shared_restaurants", len(filteredEmployee.Restaurants))
	h.writeJSON(w, http.StatusOK, filteredEmployee)
}

// UpdateEmployee handles PUT /employees/{id}.
//
//	@Summary		Update employee role assignments (admin only)
//	@Description	Update an employee's role assignments in specified restaurants (admin only, matches Python PutEmployeeSchema)
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string								true	"Employee ID"
//	@Param			employee	body		model.AdminUpdateEmployeeRequest	true	"Role assignment data"
//	@Success		200			{object}	model.Employee						"Employee roles updated successfully"
//	@Failure		400			{object}	map[string]string					"Invalid request or self-update attempt"
//	@Failure		403			{object}	map[string]string					"Access denied - insufficient permissions"
//	@Failure		404			{object}	map[string]string					"Employee not found"
//	@Failure		500			{object}	map[string]string					"Internal server error"
//	@Router			/employees/{id} [put]
//	@Security		BearerAuth
func (h *EmployeeHandler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	requester, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during UpdateEmployee request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid employee ID")
		return
	}

	// First get the target employee to check access
	targetEmployee, err := h.employeeService.GetEmployeeByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get target employee for access check", "error", err)
		h.writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	var req model.AdminUpdateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate that requester has ManageEmployees permission for all specified restaurants
	if !requester.HasPermissionForRestaurants(model.PermissionManageEmployees, req.RestaurantsIDs) {
		h.logger.Warn("Employee attempted to assign roles without ManageEmployees permission",
			"requester_id", requester.ID,
			"target_employee_id", targetEmployee.ID,
			"requested_restaurants", req.RestaurantsIDs,
			"requested_role", req.Role)
		h.writeError(w, http.StatusForbidden, "Access denied - you don't have ManageEmployees permission for the specified restaurants")
		return
	}

	// Update employee roles using the new service method
	updatedEmployee, err := h.employeeService.UpdateEmployeeRoles(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update employee roles", "error", err)
		if err.Error() == "owner role cannot be assigned through this endpoint" {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to update employee roles")
		}
		return
	}

	// Filter the updated employee's restaurants to only show those the requester has access to
	requesterRestaurantIDs := requester.GetRestaurantIDs()
	filteredEmployee := updatedEmployee.FilterRestaurantsByAccess(requesterRestaurantIDs)

	h.logger.Info("Employee roles updated successfully",
		"requester_id", requester.ID,
		"target_employee_id", targetEmployee.ID,
		"updated_restaurants", len(req.RestaurantsIDs),
		"new_role", req.Role,
		"visible_restaurants", len(filteredEmployee.Restaurants))
	h.writeJSON(w, http.StatusOK, filteredEmployee)
}

// DeleteEmployee handles DELETE /employees/{id}.
//
//	@Summary		Remove employee from shared restaurants
//	@Description	Remove an employee's access to restaurants where the requester has ManageEmployees permission (matches Python DELETE /employees/{id})
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Employee ID"
//	@Param			restaurants	body		[]string				true	"Restaurant IDs to remove access from"
//	@Success		200			{object}	model.Employee			"Employee updated successfully"
//	@Failure		400			{object}	map[string]string		"Invalid employee ID or request body"
//	@Failure		403			{object}	map[string]string		"Access denied - insufficient permissions"
//	@Failure		404			{object}	map[string]string		"Employee not found"
//	@Failure		409			{object}	map[string]string		"Cannot remove owner from restaurant"
//	@Failure		500			{object}	map[string]string		"Internal server error"
//	@Router			/employees/{id} [delete]
//	@Security		BearerAuth
func (h *EmployeeHandler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	requester, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during DeleteEmployee request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid employee ID")
		return
	}

	// Parse request body for restaurant IDs to remove access from
	var requestData struct {
		RestaurantIDs []primitive.ObjectID `json:"restaurantIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(requestData.RestaurantIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "Restaurant IDs are required")
		return
	}

	// First get the target employee to check access
	targetEmployee, err := h.employeeService.GetEmployeeByID(r.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get target employee for deletion", "error", err)
		h.writeError(w, http.StatusNotFound, "Employee not found")
		return
	}

	// Validate that requester has ManageEmployees permission for all specified restaurants
	if !requester.HasPermissionForRestaurants(model.PermissionManageEmployees, requestData.RestaurantIDs) {
		h.logger.Warn("Employee attempted to remove access without ManageEmployees permission",
			"requester_id", requester.ID,
			"target_employee_id", targetEmployee.ID,
			"requested_restaurants", requestData.RestaurantIDs)
		h.writeError(w, http.StatusForbidden, "Access denied - you don't have ManageEmployees permission for the specified restaurants")
		return
	}

	// Check if trying to remove owner roles (should be blocked like Python version)
	for _, restaurantID := range requestData.RestaurantIDs {
		if targetEmployee.IsOwnerOfRestaurant(restaurantID) {
			h.logger.Warn("Attempted to remove owner from restaurant",
				"requester_id", requester.ID,
				"target_employee_id", targetEmployee.ID,
				"restaurant_id", restaurantID)
			h.writeError(w, http.StatusConflict, "Cannot remove employee from restaurant where they are owner")
			return
		}
	}

	// Remove employee from specified restaurants using the new service method
	updatedEmployee, err := h.employeeService.RemoveEmployeeFromRestaurants(r.Context(), id, requestData.RestaurantIDs)
	if err != nil {
		h.logger.Error("Failed to remove employee from restaurants", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to remove employee from restaurants")
		return
	}

	// Filter the updated employee's restaurants to only show those the requester has access to
	requesterRestaurantIDs := requester.GetRestaurantIDs()
	filteredEmployee := updatedEmployee.FilterRestaurantsByAccess(requesterRestaurantIDs)

	h.logger.Info("Employee removed from restaurants successfully",
		"requester_id", requester.ID,
		"target_employee_id", targetEmployee.ID,
		"removed_from_restaurants", len(requestData.RestaurantIDs),
		"visible_restaurants", len(filteredEmployee.Restaurants))
	h.writeJSON(w, http.StatusOK, filteredEmployee)
}

// ListEmployees handles GET /employees.
//
//	@Summary		List employees
//	@Description	Retrieve all employees from restaurants where the requester has access
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Success		200		{array}		model.Employee			"List of all employees"
//	@Failure		403		{object}	map[string]string		"Insufficient permissions"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/employees [get]
//	@Security		BearerAuth
func (h *EmployeeHandler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during ListEmployees request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Check if employee has permission to read employee data
	// They need either ReadReservations (which includes seeing staff) or ManageEmployees permission
	if !employee.HasPermission(model.PermissionManageEmployees) {
		h.logger.Warn("Employee attempted to list employees without permission",
			"employee_id", employee.ID)
		h.writeError(w, http.StatusForbidden, "You don't have permission to view employees")
		return
	}

	// Get requester's restaurant IDs for efficient database filtering
	requesterRestaurantIDs := employee.GetRestaurantIDs()

	// Use efficient database-level filtering instead of loading all employees
	employees, err := h.employeeService.ListEmployeesByRestaurantAccess(r.Context(), requesterRestaurantIDs)
	if err != nil {
		h.logger.Error("Failed to list employees by restaurant access", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to list employees")
		return
	}

	// Filter restaurant data for each employee to only show shared restaurants
	filteredEmployees := make([]*model.Employee, len(employees))
	for i, emp := range employees {
		filteredEmployees[i] = emp.FilterRestaurantsByAccess(requesterRestaurantIDs)
	}

	h.logger.Info("Listed employees", "requester_id", employee.ID, "total_count", len(filteredEmployees))
	h.writeJSON(w, http.StatusOK, filteredEmployees)
}

// GetPendingInvitations handles GET /employees/invitations/pending.
//
//	@Summary		Get pending employee invitations
//	@Description	Retrieve all pending employee invitations
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}	model.PendingInvitation	"List of pending invitations"
//	@Failure		500	{object}	map[string]string		"Internal server error"
//	@Router			/employees/invitations/pending [get]
//	@Security		BearerAuth
func (h *EmployeeHandler) GetPendingInvitations(w http.ResponseWriter, r *http.Request) {
	pendingCodes, err := h.authService.GetPendingInvitations(r.Context())
	if err != nil {
		h.logger.Error("Failed to retrieve pending invitations", "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to retrieve pending invitations")
		return
	}

	// Convert to response format
	invitations := make([]model.PendingInvitation, len(pendingCodes))
	for i, code := range pendingCodes {
		expiresAt := code.CreatedAt.Add(model.AuthRequestExpiration)
		invitations[i] = model.PendingInvitation{
			ID:          code.ID.Hex(),
			Email:       code.Email,
			FullName:    code.FullName,
			Role:        code.Role,
			Restaurants: code.RestaurantsIDs,
			CreatedAt:   code.CreatedAt,
			ExpiresAt:   expiresAt,
			Status:      "pending",
		}
	}

	h.logger.Info("Retrieved pending invitations", "count", len(invitations))
	h.writeJSON(w, http.StatusOK, invitations)
}

// DeleteInvitation handles DELETE /employees/invitations/{invitationId}.
//
//	@Summary		Cancel employee invitation
//	@Description	Cancel a pending employee invitation
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			invitationId	path		string				true	"Invitation ID"
//	@Success		204			{string}	string				"Invitation cancelled successfully"
//	@Failure		400			{object}	map[string]string	"Invalid invitation ID"
//	@Failure		404			{object}	map[string]string	"Invitation not found"
//	@Failure		500			{object}	map[string]string	"Internal server error"
//	@Router			/employees/invitations/{invitationId} [delete]
//	@Security		BearerAuth
func (h *EmployeeHandler) DeleteInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID := chi.URLParam(r, "invitationId")
	if invitationID == "" {
		h.writeError(w, http.StatusBadRequest, "Invitation ID is required")
		return
	}

	if err := h.authService.CancelInvitation(r.Context(), invitationID); err != nil {
		h.logger.Error("Failed to cancel invitation", "invitation_id", invitationID, "error", err)
		if err.Error() == "invitation not found" {
			h.writeError(w, http.StatusNotFound, "Invitation not found")
		} else if err.Error() == "invitation has already been used" || err.Error() == "invitation has already expired" {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to cancel invitation")
		}
		return
	}

	h.logger.Info("Invitation cancelled successfully", "invitation_id", invitationID)
	w.WriteHeader(http.StatusNoContent)
}

// ExtendInvitation handles POST /employees/invitations/extend.
//
//	@Summary		Extend employee invitation
//	@Description	Extend the expiration date of a pending invitation
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			request	body		map[string]interface{}	true	"Invitation extension data"
//	@Success		200		{object}	map[string]interface{}	"Invitation extended successfully"
//	@Failure		400		{object}	map[string]string		"Invalid request body"
//	@Failure		404		{object}	map[string]string		"Invitation not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/employees/invitations/extend [post]
//	@Security		BearerAuth
func (h *EmployeeHandler) ExtendInvitation(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	invitationID, ok := req["invitationId"].(string)
	if !ok || invitationID == "" {
		h.writeError(w, http.StatusBadRequest, "Invitation ID is required")
		return
	}

	extendedInvitation, err := h.authService.ExtendInvitation(r.Context(), invitationID)
	if err != nil {
		h.logger.Error("Failed to extend invitation", "invitation_id", invitationID, "error", err)
		if err.Error() == "invitation not found" {
			h.writeError(w, http.StatusNotFound, "Invitation not found")
		} else if err.Error() == "invitation has already been used" {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to extend invitation")
		}
		return
	}

	newExpiresAt := extendedInvitation.CreatedAt.Add(model.AuthRequestExpiration)
	response := map[string]interface{}{
		"invitationId": extendedInvitation.ID.Hex(),
		"newExpiresAt": newExpiresAt,
		"message":      "Invitation extended successfully",
		"email":        extendedInvitation.Email,
		"fullName":     extendedInvitation.FullName,
	}

	h.logger.Info("Invitation extended successfully", "old_invitation_id", invitationID, "new_invitation_id", extendedInvitation.ID.Hex())
	h.writeJSON(w, http.StatusOK, response)
}

// UpdateInvitation handles PATCH /employees/invitations.
//
//	@Summary		Update employee invitation
//	@Description	Update details of a pending invitation
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			request	body		map[string]interface{}	true	"Invitation update data"
//	@Success		200		{object}	map[string]interface{}	"Invitation updated successfully"
//	@Failure		400		{object}	map[string]string		"Invalid request body"
//	@Failure		404		{object}	map[string]string		"Invitation not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Router			/employees/invitations [patch]
//	@Security		BearerAuth
func (h *EmployeeHandler) UpdateInvitation(w http.ResponseWriter, r *http.Request) {
	var reqData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	invitationID, ok := reqData["invitationId"].(string)
	if !ok || invitationID == "" {
		h.writeError(w, http.StatusBadRequest, "Invitation ID is required")
		return
	}

	// Convert the map to CreateEmployeeInvitationRequest
	updates := &model.CreateEmployeeInvitationRequest{}

	if email, ok := reqData["email"].(string); ok {
		updates.Email = email
	}
	if fullName, ok := reqData["fullName"].(string); ok {
		updates.FullName = fullName
	}
	if role, ok := reqData["role"].(string); ok {
		updates.Role = role
	}
	if restaurantsData, ok := reqData["restaurantsIds"].([]interface{}); ok {
		restaurantIDs := make([]primitive.ObjectID, 0, len(restaurantsData))
		for _, id := range restaurantsData {
			if idStr, ok := id.(string); ok {
				if objID, err := primitive.ObjectIDFromHex(idStr); err == nil {
					restaurantIDs = append(restaurantIDs, objID)
				}
			}
		}
		updates.RestaurantsIDs = restaurantIDs
	}

	updatedInvitation, err := h.authService.UpdateInvitation(r.Context(), invitationID, updates)
	if err != nil {
		h.logger.Error("Failed to update invitation", "invitation_id", invitationID, "error", err)
		if err.Error() == "invitation not found" {
			h.writeError(w, http.StatusNotFound, "Invitation not found")
		} else if err.Error() == "invitation has already been used" || err.Error() == "invitation has already expired" {
			h.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			h.writeError(w, http.StatusInternalServerError, "Failed to update invitation")
		}
		return
	}

	response := map[string]interface{}{
		"invitationId": updatedInvitation.ID.Hex(),
		"email":        updatedInvitation.Email,
		"fullName":     updatedInvitation.FullName,
		"role":         updatedInvitation.Role,
		"restaurants":  updatedInvitation.RestaurantsIDs,
		"message":      "Invitation updated successfully",
		"updatedAt":    updatedInvitation.UpdatedAt,
	}

	h.logger.Info("Invitation updated successfully", "invitation_id", invitationID)
	h.writeJSON(w, http.StatusOK, response)
}

// UpdateMe handles PATCH /employees/me.
//
//	@Summary		Update current employee profile
//	@Description	Update the currently authenticated employee's profile information
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			employee	body		model.PersonalCabinetUpdateRequest	true	"Employee profile update data"
//	@Success		200			{object}	model.Employee						"Employee profile updated successfully"
//	@Failure		400			{object}	map[string]string					"Invalid request body"
//	@Failure		401			{object}	map[string]string					"Unauthorized - invalid or missing token"
//	@Failure		404			{object}	map[string]string					"Employee not found"
//	@Failure		500			{object}	map[string]string					"Internal server error"
//	@Router			/employees/me [patch]
//	@Security		BearerAuth
func (h *EmployeeHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during UpdateMe request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req model.PersonalCabinetUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update employee using the service
	updateReq := &model.UpdateEmployeeRequest{
		FullName: req.FullName,
	}

	updatedEmployee, err := h.employeeService.UpdateEmployee(r.Context(), employee.ID, updateReq)
	if err != nil {
		h.logger.Error("Failed to update employee profile", "employee_id", employee.ID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	h.logger.Info("Employee profile updated successfully", "employee_id", employee.ID)
	h.writeJSON(w, http.StatusOK, updatedEmployee)
}

// UpdateEmail handles POST /employees/me/email/update.
//
//	@Summary		Request email update
//	@Description	Initiate email update process with verification code
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.EmailUpdateRequest	true	"Email update request"
//	@Success		200		{object}	model.EmailUpdateResponse	"Email update initiated"
//	@Failure		400		{object}	map[string]string			"Invalid request body"
//	@Failure		401		{object}	map[string]string			"Unauthorized - invalid or missing token"
//	@Failure		500		{object}	map[string]string			"Internal server error"
//	@Router			/employees/me/email/update [post]
//	@Security		BearerAuth
func (h *EmployeeHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during UpdateEmail request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req model.EmailUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement email update logic in auth service
	// For now, return a mock response
	response := &model.EmailUpdateResponse{
		CodeRequestID: "mock_email_update_request_id",
	}

	h.logger.Info("Email update requested", "employee_id", employee.ID, "new_email", req.NewEmail)
	h.writeJSON(w, http.StatusOK, response)
}

// VerifyEmailUpdate handles POST /employees/me/email/verify.
//
//	@Summary		Verify email update
//	@Description	Verify new email address with verification code
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.EmailVerifyRequest	true	"Email verification request"
//	@Success		200		{object}	model.Employee				"Email updated successfully"
//	@Failure		400		{object}	map[string]string			"Invalid request body or verification code"
//	@Failure		401		{object}	map[string]string			"Unauthorized - invalid or missing token"
//	@Failure		500		{object}	map[string]string			"Internal server error"
//	@Router			/employees/me/email/verify [post]
//	@Security		BearerAuth
func (h *EmployeeHandler) VerifyEmailUpdate(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during VerifyEmailUpdate request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req model.EmailVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Implement email verification logic in auth service
	// For now, return current employee data
	h.logger.Info("Email verification completed", "employee_id", employee.ID, "code_request_id", req.CodeRequestID)
	h.writeJSON(w, http.StatusOK, employee)
}

// ConnectTelegram handles POST /employees/me/telegram/connect.
//
//	@Summary		Connect Telegram account
//	@Description	Initiate Telegram account connection process
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.TelegramConnectionResponse	"Telegram connection initiated"
//	@Failure		401	{object}	map[string]string					"Unauthorized - invalid or missing token"
//	@Failure		500	{object}	map[string]string					"Internal server error"
//	@Router			/employees/me/telegram/connect [post]
//	@Security		BearerAuth
func (h *EmployeeHandler) ConnectTelegram(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during ConnectTelegram request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Use auth service to handle Telegram connection
	response, err := h.authService.ConnectTelegram(r.Context(), employee.ID)
	if err != nil {
		h.logger.Error("Failed to initiate Telegram connection", "employee_id", employee.ID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to initiate Telegram connection")
		return
	}

	h.logger.Info("Telegram connection requested", "employee_id", employee.ID, "request_id", response.ConnectionRequestID)
	h.writeJSON(w, http.StatusOK, response)
}

// CheckTelegramConnection handles GET /employees/me/telegram/connect/{requestId}.
//
//	@Summary		Check Telegram connection status
//	@Description	Check the status of a Telegram connection request
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Param			requestId	path		string									true	"Connection request ID"
//	@Success		200			{object}	model.TelegramConnectionVerifyResponse	"Connection status"
//	@Failure		400			{object}	map[string]string						"Invalid request ID"
//	@Failure		401			{object}	map[string]string						"Unauthorized - invalid or missing token"
//	@Failure		404			{object}	map[string]string						"Connection request not found"
//	@Failure		500			{object}	map[string]string						"Internal server error"
//	@Router			/employees/me/telegram/connect/{requestId} [get]
//	@Security		BearerAuth
func (h *EmployeeHandler) CheckTelegramConnection(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during CheckTelegramConnection request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Extract request ID from URL parameter
	requestID := chi.URLParam(r, "requestId")
	if requestID == "" {
		h.writeError(w, http.StatusBadRequest, "Request ID is required")
		return
	}

	// Use auth service to check Telegram connection status
	response, err := h.authService.CheckTelegramConnection(r.Context(), requestID, employee.ID)
	if err != nil {
		h.logger.Error("Failed to check Telegram connection status", "employee_id", employee.ID, "request_id", requestID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to check Telegram connection status")
		return
	}

	h.logger.Info("Telegram connection status checked", "employee_id", employee.ID, "request_id", requestID, "is_pending", response.IsPending)
	h.writeJSON(w, http.StatusOK, response)
}

// DisconnectTelegram handles DELETE /employees/me/telegram.
//
//	@Summary		Disconnect Telegram account
//	@Description	Disconnect the current employee's Telegram account
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.Employee		"Telegram disconnected successfully"
//	@Failure		401	{object}	map[string]string	"Unauthorized - invalid or missing token"
//	@Failure		404	{object}	map[string]string	"Employee not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/employees/me/telegram [delete]
//	@Security		BearerAuth
func (h *EmployeeHandler) DisconnectTelegram(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during DisconnectTelegram request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Use auth service to disconnect Telegram
	updatedEmployee, err := h.authService.DisconnectTelegram(r.Context(), employee.ID)
	if err != nil {
		h.logger.Error("Failed to disconnect Telegram", "employee_id", employee.ID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "Failed to disconnect Telegram")
		return
	}

	h.logger.Info("Telegram disconnected successfully", "employee_id", employee.ID)
	h.writeJSON(w, http.StatusOK, updatedEmployee)
}

// DisconnectEmail handles DELETE /employees/me/email.
//
//	@Summary		Disconnect email account
//	@Description	Disconnect the current employee's email account
//	@Tags			employees
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.Employee		"Email disconnected successfully"
//	@Failure		401	{object}	map[string]string	"Unauthorized - invalid or missing token"
//	@Failure		404	{object}	map[string]string	"Employee not found"
//	@Failure		500	{object}	map[string]string	"Internal server error"
//	@Router			/employees/me/email [delete]
//	@Security		BearerAuth
func (h *EmployeeHandler) DisconnectEmail(w http.ResponseWriter, r *http.Request) {
	// Get authenticated employee from middleware context
	employee, ok := middleware.GetEmployeeFromContext(r.Context())
	if !ok {
		h.logger.Error("Employee not found in context during DisconnectEmail request")
		h.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// TODO: Implement email disconnection logic
	// For now, just return the employee data
	h.logger.Info("Email disconnected", "employee_id", employee.ID)
	h.writeJSON(w, http.StatusOK, employee)
}

// Helper methods

func (h *EmployeeHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}

func (h *EmployeeHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
		// Note: We can't call http.Error here as headers are already written
		return
	}
}
