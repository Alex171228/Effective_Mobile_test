package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alexs/subscription-service/internal/model"
)

type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

func (r *SubscriptionRepository) Create(ctx context.Context, s model.Subscription) (model.Subscription, error) {
	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

	var created model.Subscription
	err := r.pool.QueryRow(ctx, query,
		s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate,
	).Scan(
		&created.ID, &created.ServiceName, &created.Price,
		&created.UserID, &created.StartDate, &created.EndDate,
		&created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return model.Subscription{}, fmt.Errorf("insert subscription: %w", err)
	}

	return created, nil
}

func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (model.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	var s model.Subscription
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.ServiceName, &s.Price,
		&s.UserID, &s.StartDate, &s.EndDate,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.Subscription{}, fmt.Errorf("subscription %s not found", id)
		}
		return model.Subscription{}, fmt.Errorf("get subscription: %w", err)
	}

	return s, nil
}

func (r *SubscriptionRepository) Update(ctx context.Context, id uuid.UUID, s model.Subscription) (model.Subscription, error) {
	query := `
		UPDATE subscriptions
		SET service_name = $1, price = $2, start_date = $3, end_date = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, service_name, price, user_id, start_date, end_date, created_at, updated_at`

	var updated model.Subscription
	err := r.pool.QueryRow(ctx, query,
		s.ServiceName, s.Price, s.StartDate, s.EndDate, id,
	).Scan(
		&updated.ID, &updated.ServiceName, &updated.Price,
		&updated.UserID, &updated.StartDate, &updated.EndDate,
		&updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.Subscription{}, fmt.Errorf("subscription %s not found", id)
		}
		return model.Subscription{}, fmt.Errorf("update subscription: %w", err)
	}

	return updated, nil
}

func (r *SubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM subscriptions WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("subscription %s not found", id)
	}

	return nil
}

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

func (r *SubscriptionRepository) List(ctx context.Context, f ListFilter) ([]model.Subscription, int64, error) {
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	if f.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *f.UserID)
		argIdx++
	}
	if f.ServiceName != nil && *f.ServiceName != "" {
		conditions = append(conditions, fmt.Sprintf("service_name ILIKE $%d", argIdx))
		args = append(args, "%"+*f.ServiceName+"%")
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscriptions %s", where)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	args = append(args, f.Limit, f.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []model.Subscription
	for rows.Next() {
		var s model.Subscription
		if err := rows.Scan(
			&s.ID, &s.ServiceName, &s.Price,
			&s.UserID, &s.StartDate, &s.EndDate,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, s)
	}

	return subscriptions, total, nil
}

func (r *SubscriptionRepository) CalculateTotalCost(
	ctx context.Context,
	periodStart, periodEnd time.Time,
	userID *uuid.UUID,
	serviceName *string,
) (int64, error) {
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	// Only subscriptions that overlap with the requested period
	conditions = append(conditions, fmt.Sprintf("start_date <= $%d", argIdx))
	args = append(args, periodEnd)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("(end_date IS NULL OR end_date >= $%d)", argIdx))
	args = append(args, periodStart)
	argIdx++

	if userID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *userID)
		argIdx++
	}
	if serviceName != nil && *serviceName != "" {
		conditions = append(conditions, fmt.Sprintf("service_name = $%d", argIdx))
		args = append(args, *serviceName)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Calculate months of overlap for each subscription within the period, then multiply by price
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(
			price * (
				(EXTRACT(YEAR FROM LEAST(COALESCE(end_date, $1), $1)) * 12
				 + EXTRACT(MONTH FROM LEAST(COALESCE(end_date, $1), $1)))
				- (EXTRACT(YEAR FROM GREATEST(start_date, $2)) * 12
				   + EXTRACT(MONTH FROM GREATEST(start_date, $2)))
				+ 1
			)
		), 0)::bigint AS total_cost
		FROM subscriptions %s`, where)

	var totalCost int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&totalCost)
	if err != nil {
		return 0, fmt.Errorf("calculate total cost: %w", err)
	}

	return totalCost, nil
}
