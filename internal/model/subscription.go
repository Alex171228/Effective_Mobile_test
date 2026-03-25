package model

import (
	"time"

	"github.com/google/uuid"
)

const MonthDateFormat = "01-2006"

type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// --- Request DTOs ---

type CreateSubscriptionRequest struct {
	ServiceName string  `json:"service_name" example:"Yandex Plus"`
	Price       int     `json:"price" example:"400"`
	UserID      string  `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string  `json:"start_date" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
}

type UpdateSubscriptionRequest struct {
	ServiceName string  `json:"service_name" example:"Yandex Plus"`
	Price       int     `json:"price" example:"500"`
	StartDate   string  `json:"start_date" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
}

// --- Response DTOs ---

type SubscriptionResponse struct {
	ID          string  `json:"id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	ServiceName string  `json:"service_name" example:"Yandex Plus"`
	Price       int     `json:"price" example:"400"`
	UserID      string  `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string  `json:"start_date" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
	CreatedAt   string  `json:"created_at" example:"2025-07-01T00:00:00Z"`
	UpdatedAt   string  `json:"updated_at" example:"2025-07-01T00:00:00Z"`
}

type ListSubscriptionsResponse struct {
	Subscriptions []SubscriptionResponse `json:"subscriptions"`
	Total         int64                  `json:"total"`
}

type CostResponse struct {
	TotalCost int64 `json:"total_cost" example:"4800"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

// --- Converters ---

func ToSubscriptionResponse(s Subscription) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:          s.ID.String(),
		ServiceName: s.ServiceName,
		Price:       s.Price,
		UserID:      s.UserID.String(),
		StartDate:   s.StartDate.Format(MonthDateFormat),
		CreatedAt:   s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
	}
	if s.EndDate != nil {
		end := s.EndDate.Format(MonthDateFormat)
		resp.EndDate = &end
	}
	return resp
}
