package model

// ErrorDetail represents the error details.
type ErrorDetail struct {
	Message string `json:"message" example:"Error message"`
	Code    string `json:"code,omitempty" example:"ERR_001"`
	IsShow  bool   `json:"isShow" example:"false"`
}

// ErrorResponse represents an error response with nested error object.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}
