package handler

import (
	"context"

	"github.com/alexs/subscription-service/internal/model"
)

type SubscriptionSvc interface {
	Create(ctx context.Context, req model.CreateSubscriptionRequest) (model.SubscriptionResponse, error)
	GetByID(ctx context.Context, id string) (model.SubscriptionResponse, error)
	Update(ctx context.Context, id string, req model.UpdateSubscriptionRequest) (model.SubscriptionResponse, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID, serviceName *string, limit, offset int) (model.ListSubscriptionsResponse, error)
	CalculateTotalCost(ctx context.Context, periodStart, periodEnd string, userID, serviceName *string) (model.CostResponse, error)
}
