package model

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Message string `json:"message" example:"Error message"`
	Code    string `json:"code,omitempty" example:"ERR_001"`
	Details string `json:"details,omitempty" example:"Additional error details"`
}

// SuccessResponse represents a success response.
type SuccessResponse struct {
	Message string      `json:"message" example:"Operation successful"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total" example:"100"`
	Page       int         `json:"page" example:"1"`
	PageSize   int         `json:"page_size" example:"10"`
	TotalPages int         `json:"total_pages" example:"10"`
}