package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/alexs/subscription-service/internal/model"
)

type SubscriptionRepo interface {
	Create(ctx context.Context, s model.Subscription) (model.Subscription, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Subscription, error)
	Update(ctx context.Context, id uuid.UUID, s model.Subscription) (model.Subscription, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f model.ListFilter) ([]model.Subscription, int64, error)
	CalculateTotalCost(ctx context.Context, periodStart, periodEnd time.Time, userID *uuid.UUID, serviceName *string) (int64, error)
}
