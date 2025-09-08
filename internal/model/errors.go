package model

// Error codes for structured error responses
const (
	// Authentication/Authorization errors
	ErrCodeEmailRequiredForTelegramDisconnect = "EMAIL_REQUIRED_FOR_TELEGRAM_DISCONNECT"
	ErrCodeTelegramRequiredForEmailDisconnect = "TELEGRAM_REQUIRED_FOR_EMAIL_DISCONNECT"
	
	// Auth service specific errors
	ErrCodeInvalidCredentials         = "INVALID_CREDENTIALS"
	ErrCodeEmployeeAlreadyExists      = "EMPLOYEE_ALREADY_EXISTS"
	ErrCodeInvalidVerificationCode    = "INVALID_VERIFICATION_CODE"
	ErrCodeVerificationCodeExpired    = "VERIFICATION_CODE_EXPIRED"
	ErrCodeEmployeeNotFound           = "EMPLOYEE_NOT_FOUND"
	ErrCodeAccountLocked              = "ACCOUNT_LOCKED"
	ErrCodeInternalServerError        = "INTERNAL_SERVER_ERROR"
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

	// Auth service predefined errors
	ErrInvalidCredentials = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "invalid credentials",
				Code:    ErrCodeInvalidCredentials,
				IsShow:  true,
			},
		},
	}

	ErrEmployeeAlreadyExists = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "employee already exists",
				Code:    ErrCodeEmployeeAlreadyExists,
				IsShow:  true,
			},
		},
	}

	ErrInvalidVerificationCode = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "invalid verification code",
				Code:    ErrCodeInvalidVerificationCode,
				IsShow:  true,
			},
		},
	}

	ErrVerificationCodeExpired = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "verification code is invalid or expired",
				Code:    ErrCodeVerificationCodeExpired,
				IsShow:  true,
			},
		},
	}

	ErrEmployeeNotFound = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "employee not found",
				Code:    ErrCodeEmployeeNotFound,
				IsShow:  false,
			},
		},
	}

	ErrAccountLocked = &ValidationError{
		Response: &ErrorResponse{
			Error: ErrorDetail{
				Message: "too many failed attempts, account temporarily locked",
				Code:    ErrCodeAccountLocked,
				IsShow:  true,
			},
		},
	}
)