package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PendingInvitation represents a single pending invitation
type PendingInvitation struct {
	ID          string                 `json:"id" example:"507f1f77bcf86cd799439011"`
	Email       string                 `json:"email" example:"john.doe@example.com"`
	FullName    string                 `json:"fullName" example:"John Doe"`
	Role        string                 `json:"role" example:"admin"`
	RestaurantID primitive.ObjectID    `json:"restaurantId"`
	CreatedAt   time.Time              `json:"createdAt" example:"2023-01-01T00:00:00Z"`
	ExpiresAt   time.Time              `json:"expiresAt" example:"2023-01-02T00:00:00Z"`
	Status      string                 `json:"status" example:"pending"`
}