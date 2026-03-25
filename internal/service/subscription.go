package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/alexs/subscription-service/internal/model"
	"github.com/alexs/subscription-service/internal/repository"
)

type SubscriptionService struct {
	repo   *repository.SubscriptionRepository
	logger *slog.Logger
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, logger *slog.Logger) *SubscriptionService {
	return &SubscriptionService{
		repo:   repo,
		logger: logger,
	}
}

func (s *SubscriptionService) Create(ctx context.Context, req model.CreateSubscriptionRequest) (model.SubscriptionResponse, error) {
	s.logger.Info("creating subscription",
		slog.String("service_name", req.ServiceName),
		slog.String("user_id", req.UserID),
	)

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return model.SubscriptionResponse{}, fmt.Errorf("invalid user_id: %w", err)
	}

	startDate, err := time.Parse(model.MonthDateFormat, req.StartDate)
	if err != nil {
		return model.SubscriptionResponse{}, fmt.Errorf("invalid start_date (expected MM-YYYY): %w", err)
	}

	var endDate *time.Time
	if req.EndDate != nil {
		t, err := time.Parse(model.MonthDateFormat, *req.EndDate)
		if err != nil {
			return model.SubscriptionResponse{}, fmt.Errorf("invalid end_date (expected MM-YYYY): %w", err)
		}
		endDate = &t
	}

	if req.Price <= 0 {
		return model.SubscriptionResponse{}, fmt.Errorf("price must be positive")
	}

	if req.ServiceName == "" {
		return model.SubscriptionResponse{}, fmt.Errorf("service_name is required")
	}

	sub := model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      userID,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		s.logger.Error("failed to create subscription", slog.String("error", err.Error()))
		return model.SubscriptionResponse{}, fmt.Errorf("creating subscription: %w", err)
	}

	s.logger.Info("subscription created", slog.String("id", created.ID.String()))
	return model.ToSubscriptionResponse(created), nil
}

func (s *SubscriptionService) GetByID(ctx context.Context, id string) (model.SubscriptionResponse, error) {
	s.logger.Info("getting subscription", slog.String("id", id))

	uid, err := uuid.Parse(id)
	if err != nil {
		return model.SubscriptionResponse{}, fmt.Errorf("invalid id: %w", err)
	}

	sub, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		s.logger.Error("failed to get subscription", slog.String("id", id), slog.String("error", err.Error()))
		return model.SubscriptionResponse{}, err
	}

	return model.ToSubscriptionResponse(sub), nil
}

func (s *SubscriptionService) Update(ctx context.Context, id string, req model.UpdateSubscriptionRequest) (model.SubscriptionResponse, error) {
	s.logger.Info("updating subscription", slog.String("id", id))

	uid, err := uuid.Parse(id)
	if err != nil {
		return model.SubscriptionResponse{}, fmt.Errorf("invalid id: %w", err)
	}

	startDate, err := time.Parse(model.MonthDateFormat, req.StartDate)
	if err != nil {
		return model.SubscriptionResponse{}, fmt.Errorf("invalid start_date (expected MM-YYYY): %w", err)
	}

	var endDate *time.Time
	if req.EndDate != nil {
		t, err := time.Parse(model.MonthDateFormat, *req.EndDate)
		if err != nil {
			return model.SubscriptionResponse{}, fmt.Errorf("invalid end_date (expected MM-YYYY): %w", err)
		}
		endDate = &t
	}

	if req.Price <= 0 {
		return model.SubscriptionResponse{}, fmt.Errorf("price must be positive")
	}

	if req.ServiceName == "" {
		return model.SubscriptionResponse{}, fmt.Errorf("service_name is required")
	}

	sub := model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	updated, err := s.repo.Update(ctx, uid, sub)
	if err != nil {
		s.logger.Error("failed to update subscription", slog.String("id", id), slog.String("error", err.Error()))
		return model.SubscriptionResponse{}, err
	}

	s.logger.Info("subscription updated", slog.String("id", id))
	return model.ToSubscriptionResponse(updated), nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id string) error {
	s.logger.Info("deleting subscription", slog.String("id", id))

	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	if err := s.repo.Delete(ctx, uid); err != nil {
		s.logger.Error("failed to delete subscription", slog.String("id", id), slog.String("error", err.Error()))
		return err
	}

	s.logger.Info("subscription deleted", slog.String("id", id))
	return nil
}

func (s *SubscriptionService) List(ctx context.Context, userID, serviceName *string, limit, offset int) (model.ListSubscriptionsResponse, error) {
	s.logger.Info("listing subscriptions",
		slog.Int("limit", limit),
		slog.Int("offset", offset),
	)

	filter := repository.ListFilter{
		Limit:  limit,
		Offset: offset,
	}

	if userID != nil && *userID != "" {
		uid, err := uuid.Parse(*userID)
		if err != nil {
			return model.ListSubscriptionsResponse{}, fmt.Errorf("invalid user_id: %w", err)
		}
		filter.UserID = &uid
	}

	if serviceName != nil && *serviceName != "" {
		filter.ServiceName = serviceName
	}

	subs, total, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("failed to list subscriptions", slog.String("error", err.Error()))
		return model.ListSubscriptionsResponse{}, err
	}

	responses := make([]model.SubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		responses = append(responses, model.ToSubscriptionResponse(sub))
	}

	return model.ListSubscriptionsResponse{
		Subscriptions: responses,
		Total:         total,
	}, nil
}

func (s *SubscriptionService) CalculateTotalCost(
	ctx context.Context,
	periodStart, periodEnd string,
	userID, serviceName *string,
) (model.CostResponse, error) {
	s.logger.Info("calculating total cost",
		slog.String("period_start", periodStart),
		slog.String("period_end", periodEnd),
	)

	start, err := time.Parse(model.MonthDateFormat, periodStart)
	if err != nil {
		return model.CostResponse{}, fmt.Errorf("invalid period_start (expected MM-YYYY): %w", err)
	}

	end, err := time.Parse(model.MonthDateFormat, periodEnd)
	if err != nil {
		return model.CostResponse{}, fmt.Errorf("invalid period_end (expected MM-YYYY): %w", err)
	}

	if end.Before(start) {
		return model.CostResponse{}, fmt.Errorf("period_end must not be before period_start")
	}

	var uid *uuid.UUID
	if userID != nil && *userID != "" {
		parsed, err := uuid.Parse(*userID)
		if err != nil {
			return model.CostResponse{}, fmt.Errorf("invalid user_id: %w", err)
		}
		uid = &parsed
	}

	totalCost, err := s.repo.CalculateTotalCost(ctx, start, end, uid, serviceName)
	if err != nil {
		s.logger.Error("failed to calculate total cost", slog.String("error", err.Error()))
		return model.CostResponse{}, err
	}

	s.logger.Info("total cost calculated", slog.Int64("total_cost", totalCost))
	return model.CostResponse{TotalCost: totalCost}, nil
}
