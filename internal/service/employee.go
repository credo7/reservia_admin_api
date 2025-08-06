// Package employee provides employee-related use cases.
package service

import (
	"context"
	"fmt"
	"github.com/reservia/api/internal/model"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/reservia/api/internal/repository"
	"github.com/reservia/api/pkg/logger"
)

// Service handles employee-related business logic.
type EmployeeService struct {
	employeeRepo repository.EmployeeRepository
	logger       logger.Logger
}

// New creates a new employee use case.
func NewEmployeeService(employeeRepo repository.EmployeeRepository, logger logger.Logger) *EmployeeService {
	return &EmployeeService{
		employeeRepo: employeeRepo,
		logger:       logger,
	}
}

// CreateEmployee creates a new employee.
func (es *EmployeeService) CreateEmployee(ctx context.Context, req *model.CreateEmployeeRequest) (*model.Employee, error) {
	// Check if employee already exists
	if req.Email != "" {
		existingEmployee, err := es.employeeRepo.GetByEmail(ctx, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing employee: %w", err)
		}
		if existingEmployee != nil {
			return nil, fmt.Errorf("employee with email %s already exists", req.Email)
		}
	}

	// Create employee
	newEmployee := &model.Employee{
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

	if err := es.employeeRepo.Create(ctx, newEmployee); err != nil {
		return nil, fmt.Errorf("failed to create employee: %w", err)
	}

	es.logger.Info("Employee created successfully", "employee_id", newEmployee.ID.Hex())
	return newEmployee, nil
}

// GetEmployeeByID retrieves an employee by ID.
func (es *EmployeeService) GetEmployeeByID(ctx context.Context, id primitive.ObjectID) (*model.Employee, error) {
	e, err := es.employeeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee by ID: %w", err)
	}
	if e == nil {
		return nil, fmt.Errorf("employee not found")
	}
	return e, nil
}

// GetEmployeeByEmail retrieves an employee by email.
func (es *EmployeeService) GetEmployeeByEmail(ctx context.Context, email string) (*model.Employee, error) {
	e, err := es.employeeRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee by email: %w", err)
	}
	if e == nil {
		return nil, fmt.Errorf("employee not found")
	}
	return e, nil
}

// GetEmployeeByTelegramID retrieves an employee by telegram ID.
func (es *EmployeeService) GetEmployeeByTelegramID(ctx context.Context, telegramID int64) (*model.Employee, error) {
	e, err := es.employeeRepo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee by telegram ID: %w", err)
	}
	if e == nil {
		return nil, fmt.Errorf("employee not found")
	}
	return e, nil
}

// UpdateEmployee updates an employee.
func (es *EmployeeService) UpdateEmployee(ctx context.Context, id primitive.ObjectID, req *model.UpdateEmployeeRequest) (*model.Employee, error) {
	// Get existing employee
	existingEmployee, err := es.employeeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}
	if existingEmployee == nil {
		return nil, fmt.Errorf("employee not found")
	}

	// Update fields
	if req.Email != nil {
		existingEmployee.Email = *req.Email
	}
	if req.FullName != nil {
		existingEmployee.FullName = *req.FullName
	}
	if req.TelegramIsBot != nil {
		existingEmployee.TelegramIsBot = req.TelegramIsBot
	}
	if req.TelegramID != nil {
		existingEmployee.TelegramID = req.TelegramID
	}
	if req.TelegramChatID != nil {
		existingEmployee.TelegramChatID = req.TelegramChatID
	}
	if req.TelegramUsername != nil {
		existingEmployee.TelegramUsername = req.TelegramUsername
	}
	if req.TelegramLang != nil {
		existingEmployee.TelegramLang = req.TelegramLang
	}
	if req.TelegramPremium != nil {
		existingEmployee.TelegramPremium = req.TelegramPremium
	}

	existingEmployee.UpdatedAt = time.Now()

	if err := es.employeeRepo.Update(ctx, existingEmployee); err != nil {
		return nil, fmt.Errorf("failed to update employee: %w", err)
	}

	es.logger.Info("Employee updated successfully", "employee_id", existingEmployee.ID.Hex())
	return existingEmployee, nil
}

// ListEmployees retrieves employees with pagination.
func (es *EmployeeService) ListEmployees(ctx context.Context, limit, offset int) ([]*model.Employee, error) {
	employees, err := es.employeeRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}
	return employees, nil
}

// ListAllEmployees retrieves all employees without pagination.
// This is suitable for restaurant management where employee count is typically small.
func (es *EmployeeService) ListAllEmployees(ctx context.Context) ([]*model.Employee, error) {
	// Use a large limit to get all employees (restaurants rarely have more than 100 employees)
	employees, err := es.employeeRepo.List(ctx, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list all employees: %w", err)
	}
	return employees, nil
}

// ListEmployeesByRestaurantAccess retrieves employees who have access to any of the specified restaurants.
// This is more efficient than ListAllEmployees as it filters at the database level.
func (es *EmployeeService) ListEmployeesByRestaurantAccess(ctx context.Context, restaurantIDs []primitive.ObjectID) ([]*model.Employee, error) {
	if len(restaurantIDs) == 0 {
		return []*model.Employee{}, nil
	}

	employees, err := es.employeeRepo.ListByRestaurantIDs(ctx, restaurantIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees by restaurant access: %w", err)
	}
	return employees, nil
}

// UpdateEmployeeRoles updates an employee's role assignments in specified restaurants.
// This matches the Python PutEmployeeSchema functionality - admin can only manage roles, not personal info.
func (es *EmployeeService) UpdateEmployeeRoles(ctx context.Context, employeeID primitive.ObjectID, req *model.AdminUpdateEmployeeRequest) (*model.Employee, error) {
	// Get existing employee
	existingEmployee, err := es.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}
	if existingEmployee == nil {
		return nil, fmt.Errorf("employee not found")
	}

	// Validate role - prevent owner role assignment (matches Python logic)
	if req.Role == model.RoleOwner {
		return nil, fmt.Errorf("owner role cannot be assigned through this endpoint")
	}

	if !req.Role.IsValid() {
		return nil, fmt.Errorf("invalid role: %s", req.Role)
	}

	// Create a copy of the employee to modify
	updatedEmployee := *existingEmployee

	// Create a map of existing restaurant roles for faster lookup
	existingRoles := make(map[primitive.ObjectID]model.EmployeeRestaurant)
	for _, restaurant := range updatedEmployee.Restaurants {
		existingRoles[restaurant.RestaurantID] = restaurant
	}

	// Update or add roles for the specified restaurants
	var newRestaurants []model.EmployeeRestaurant
	updatedRestaurants := make(map[primitive.ObjectID]bool)

	// First, add/update roles for specified restaurants
	for _, restaurantID := range req.RestaurantsIDs {
		if existingRole, exists := existingRoles[restaurantID]; exists {
			// Don't downgrade existing owners (matches Python logic)
			if existingRole.Role == model.RoleOwner {
				es.logger.Warn("Attempted to modify owner role, keeping existing role",
					"employee_id", employeeID,
					"restaurant_id", restaurantID,
					"existing_role", existingRole.Role,
					"requested_role", req.Role)
				newRestaurants = append(newRestaurants, existingRole)
			} else {
				// Update existing role
				newRestaurants = append(newRestaurants, model.EmployeeRestaurant{
					RestaurantID: restaurantID,
					Role:         req.Role,
				})
			}
		} else {
			// Add new role
			newRestaurants = append(newRestaurants, model.EmployeeRestaurant{
				RestaurantID: restaurantID,
				Role:         req.Role,
			})
		}
		updatedRestaurants[restaurantID] = true
	}

	// Keep existing roles for restaurants not in the update request
	for _, restaurant := range updatedEmployee.Restaurants {
		if !updatedRestaurants[restaurant.RestaurantID] {
			newRestaurants = append(newRestaurants, restaurant)
		}
	}

	updatedEmployee.Restaurants = newRestaurants
	updatedEmployee.UpdatedAt = time.Now()

	if err := es.employeeRepo.Update(ctx, &updatedEmployee); err != nil {
		return nil, fmt.Errorf("failed to update employee roles: %w", err)
	}

	es.logger.Info("Employee roles updated successfully",
		"employee_id", employeeID.Hex(),
		"updated_restaurants", len(req.RestaurantsIDs),
		"new_role", req.Role)
	return &updatedEmployee, nil
}

// RemoveEmployeeFromRestaurants removes an employee's access to shared restaurants.
// This matches the expected delete behavior - removing access rather than deleting the entire employee.
func (es *EmployeeService) RemoveEmployeeFromRestaurants(ctx context.Context, employeeID primitive.ObjectID, restaurantIDs []primitive.ObjectID) (*model.Employee, error) {
	// Get existing employee
	existingEmployee, err := es.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}
	if existingEmployee == nil {
		return nil, fmt.Errorf("employee not found")
	}

	if len(restaurantIDs) == 0 {
		es.logger.Warn("No restaurants specified for removal", "employee_id", employeeID)
		return existingEmployee, nil
	}

	// Create a map for faster lookups of restaurants to remove
	restaurantsToRemove := make(map[primitive.ObjectID]bool)
	for _, id := range restaurantIDs {
		restaurantsToRemove[id] = true
	}

	// Filter out restaurants that should be removed, but preserve owner roles
	var remainingRestaurants []model.EmployeeRestaurant
	removedCount := 0
	ownerRolesKept := 0

	for _, restaurant := range existingEmployee.Restaurants {
		if restaurantsToRemove[restaurant.RestaurantID] {
			// Don't remove owner roles for safety - owners should transfer ownership first
			if restaurant.Role == model.RoleOwner {
				es.logger.Warn("Cannot remove employee from restaurant where they are owner",
					"employee_id", employeeID,
					"restaurant_id", restaurant.RestaurantID,
					"role", restaurant.Role)
				remainingRestaurants = append(remainingRestaurants, restaurant)
				ownerRolesKept++
			} else {
				// Remove non-owner roles
				removedCount++
				es.logger.Info("Removing employee access to restaurant",
					"employee_id", employeeID,
					"restaurant_id", restaurant.RestaurantID,
					"role", restaurant.Role)
			}
		} else {
			// Keep restaurants not in the removal list
			remainingRestaurants = append(remainingRestaurants, restaurant)
		}
	}

	// Update employee with remaining restaurants
	updatedEmployee := *existingEmployee
	updatedEmployee.Restaurants = remainingRestaurants
	updatedEmployee.UpdatedAt = time.Now()

	if err := es.employeeRepo.Update(ctx, &updatedEmployee); err != nil {
		return nil, fmt.Errorf("failed to update employee after removing restaurant access: %w", err)
	}

	es.logger.Info("Employee restaurant access removed successfully",
		"employee_id", employeeID.Hex(),
		"restaurants_removed", removedCount,
		"owner_roles_kept", ownerRolesKept,
		"remaining_restaurants", len(remainingRestaurants))
	
	return &updatedEmployee, nil
}
