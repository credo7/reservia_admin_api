package model

// Error codes for structured error responses
const (
	// Authentication/Authorization errors
	ErrCodeEmailRequiredForTelegramDisconnect = "EMAIL_REQUIRED_FOR_TELEGRAM_DISCONNECT"
	ErrCodeTelegramRequiredForEmailDisconnect = "TELEGRAM_REQUIRED_FOR_EMAIL_DISCONNECT"
)

// ValidationError represents a business logic validation error with structured response
type ValidationError struct {
	Response *ErrorResponse
}

func (e *ValidationError) Error() string {
	return e.Response.Error.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message, code string, isShow bool) *ValidationError {
	return &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: message,
				Code:    code,
				IsShow:  isShow,
			},
		},
	}
}

// Predefined validation errors
var (
	ErrEmailRequiredForTelegramDisconnect = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "cannot disconnect Telegram when email is not connected",
				Code:    ErrCodeEmailRequiredForTelegramDisconnect,
				IsShow:  false,
			},
		},
	}

	ErrTelegramRequiredForEmailDisconnect = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "cannot disconnect email when Telegram is not connected",
				Code:    ErrCodeTelegramRequiredForEmailDisconnect,
				IsShow:  false,
			},
		},
	}
)