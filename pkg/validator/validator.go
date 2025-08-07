// Package validator provides validation utilities for the API.
package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the go-playground validator.
type Validator struct {
	validate *validator.Validate
}

// New creates a new validator instance.
func New() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

// Struct validates a struct and returns user-friendly error messages.
func (v *Validator) Struct(s interface{}) error {
	err := v.validate.Struct(s)
	if err != nil {
		// Convert validation errors to user-friendly messages
		var messages []string
		for _, err := range err.(validator.ValidationErrors) {
			messages = append(messages, v.getErrorMessage(err))
		}
		return fmt.Errorf(strings.Join(messages, ", "))
	}
	return nil
}

// getErrorMessage converts a validation error to a user-friendly message.
func (v *Validator) getErrorMessage(err validator.FieldError) string {
	field := strings.ToLower(err.Field())
	
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, err.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}