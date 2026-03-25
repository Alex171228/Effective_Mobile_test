package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexs/subscription-service/internal/model"
)

type mockRepo struct {
	createFn             func(ctx context.Context, s model.Subscription) (model.Subscription, error)
	getByIDFn            func(ctx context.Context, id uuid.UUID) (model.Subscription, error)
	updateFn             func(ctx context.Context, id uuid.UUID, s model.Subscription) (model.Subscription, error)
	deleteFn             func(ctx context.Context, id uuid.UUID) error
	listFn               func(ctx context.Context, f model.ListFilter) ([]model.Subscription, int64, error)
	calculateTotalCostFn func(ctx context.Context, periodStart, periodEnd time.Time, userID *uuid.UUID, serviceName *string) (int64, error)
}

func (m *mockRepo) Create(ctx context.Context, s model.Subscription) (model.Subscription, error) {
	return m.createFn(ctx, s)
}
func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (model.Subscription, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockRepo) Update(ctx context.Context, id uuid.UUID, s model.Subscription) (model.Subscription, error) {
	return m.updateFn(ctx, id, s)
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}
func (m *mockRepo) List(ctx context.Context, f model.ListFilter) ([]model.Subscription, int64, error) {
	return m.listFn(ctx, f)
}
func (m *mockRepo) CalculateTotalCost(ctx context.Context, periodStart, periodEnd time.Time, userID *uuid.UUID, serviceName *string) (int64, error) {
	return m.calculateTotalCostFn(ctx, periodStart, periodEnd, userID, serviceName)
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreate_Success(t *testing.T) {
	id := uuid.New()
	userID := uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba")
	now := time.Now()

	repo := &mockRepo{
		createFn: func(_ context.Context, s model.Subscription) (model.Subscription, error) {
			s.ID = id
			s.CreatedAt = now
			s.UpdatedAt = now
			return s, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	resp, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      userID.String(),
		StartDate:   "07-2025",
	})

	require.NoError(t, err)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "Yandex Plus", resp.ServiceName)
	assert.Equal(t, 400, resp.Price)
	assert.Equal(t, userID.String(), resp.UserID)
	assert.Equal(t, "07-2025", resp.StartDate)
	assert.Nil(t, resp.EndDate)
}

func TestCreate_WithEndDate(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, s model.Subscription) (model.Subscription, error) {
			s.ID = uuid.New()
			s.CreatedAt = time.Now()
			s.UpdatedAt = time.Now()
			return s, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	endDate := "12-2025"
	resp, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		Price:       600,
		UserID:      uuid.New().String(),
		StartDate:   "01-2025",
		EndDate:     &endDate,
	})

	require.NoError(t, err)
	assert.NotNil(t, resp.EndDate)
	assert.Equal(t, "12-2025", *resp.EndDate)
}

func TestCreate_InvalidUserID(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		UserID:      "not-a-uuid",
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
}

func TestCreate_InvalidStartDate(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		UserID:      uuid.New().String(),
		StartDate:   "2025-07",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start_date")
}

func TestCreate_InvalidEndDate(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())
	badEnd := "not-a-date"

	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		UserID:      uuid.New().String(),
		StartDate:   "01-2025",
		EndDate:     &badEnd,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid end_date")
}

func TestCreate_NegativePrice(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       -100,
		UserID:      uuid.New().String(),
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "price must be positive")
}

func TestCreate_ZeroPrice(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       0,
		UserID:      uuid.New().String(),
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "price must be positive")
}

func TestCreate_EmptyServiceName(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "",
		Price:       100,
		UserID:      uuid.New().String(),
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service_name is required")
}

func TestCreate_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ model.Subscription) (model.Subscription, error) {
			return model.Subscription{}, fmt.Errorf("db connection lost")
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	_, err := svc.Create(context.Background(), model.CreateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		UserID:      uuid.New().String(),
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating subscription")
}

func TestGetByID_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, uid uuid.UUID) (model.Subscription, error) {
			return model.Subscription{
				ID:          uid,
				ServiceName: "Yandex Plus",
				Price:       400,
				UserID:      uuid.New(),
				StartDate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	resp, err := svc.GetByID(context.Background(), id.String())

	require.NoError(t, err)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "Yandex Plus", resp.ServiceName)
}

func TestGetByID_InvalidID(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.GetByID(context.Background(), "not-a-uuid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid id")
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (model.Subscription, error) {
			return model.Subscription{}, fmt.Errorf("subscription %s not found", id)
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	_, err := svc.GetByID(context.Background(), uuid.New().String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdate_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		updateFn: func(_ context.Context, uid uuid.UUID, s model.Subscription) (model.Subscription, error) {
			return model.Subscription{
				ID:          uid,
				ServiceName: s.ServiceName,
				Price:       s.Price,
				UserID:      uuid.New(),
				StartDate:   s.StartDate,
				EndDate:     s.EndDate,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	endDate := "12-2025"
	resp, err := svc.Update(context.Background(), id.String(), model.UpdateSubscriptionRequest{
		ServiceName: "Yandex Plus Family",
		Price:       500,
		StartDate:   "07-2025",
		EndDate:     &endDate,
	})

	require.NoError(t, err)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "Yandex Plus Family", resp.ServiceName)
	assert.Equal(t, 500, resp.Price)
}

func TestUpdate_InvalidID(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Update(context.Background(), "bad-id", model.UpdateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid id")
}

func TestUpdate_InvalidStartDate(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Update(context.Background(), uuid.New().String(), model.UpdateSubscriptionRequest{
		ServiceName: "Test",
		Price:       100,
		StartDate:   "bad-date",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid start_date")
}

func TestUpdate_EmptyServiceName(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Update(context.Background(), uuid.New().String(), model.UpdateSubscriptionRequest{
		ServiceName: "",
		Price:       100,
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service_name is required")
}

func TestUpdate_NegativePrice(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.Update(context.Background(), uuid.New().String(), model.UpdateSubscriptionRequest{
		ServiceName: "Test",
		Price:       -50,
		StartDate:   "01-2025",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "price must be positive")
}

func TestDelete_Success(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	err := svc.Delete(context.Background(), uuid.New().String())

	assert.NoError(t, err)
}

func TestDelete_InvalidID(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	err := svc.Delete(context.Background(), "bad-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid id")
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			return fmt.Errorf("subscription %s not found", id)
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	err := svc.Delete(context.Background(), uuid.New().String())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestList_Success(t *testing.T) {
	subs := []model.Subscription{
		{
			ID:          uuid.New(),
			ServiceName: "Netflix",
			Price:       600,
			UserID:      uuid.New(),
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	repo := &mockRepo{
		listFn: func(_ context.Context, f model.ListFilter) ([]model.Subscription, int64, error) {
			assert.Equal(t, 10, f.Limit)
			assert.Equal(t, 0, f.Offset)
			return subs, 1, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	resp, err := svc.List(context.Background(), nil, nil, 10, 0)

	require.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Subscriptions, 1)
	assert.Equal(t, "Netflix", resp.Subscriptions[0].ServiceName)
}

func TestList_WithUserIDFilter(t *testing.T) {
	userID := uuid.New()
	repo := &mockRepo{
		listFn: func(_ context.Context, f model.ListFilter) ([]model.Subscription, int64, error) {
			require.NotNil(t, f.UserID)
			assert.Equal(t, userID, *f.UserID)
			return nil, 0, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	uid := userID.String()
	_, err := svc.List(context.Background(), &uid, nil, 10, 0)

	assert.NoError(t, err)
}

func TestList_InvalidUserID(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())
	bad := "not-a-uuid"

	_, err := svc.List(context.Background(), &bad, nil, 10, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
}

func TestCalculateTotalCost_Success(t *testing.T) {
	repo := &mockRepo{
		calculateTotalCostFn: func(_ context.Context, start, end time.Time, _ *uuid.UUID, _ *string) (int64, error) {
			assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), start)
			assert.Equal(t, time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), end)
			return 9600, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	resp, err := svc.CalculateTotalCost(context.Background(), "01-2025", "12-2025", nil, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(9600), resp.TotalCost)
}

func TestCalculateTotalCost_WithFilters(t *testing.T) {
	userID := uuid.New()
	repo := &mockRepo{
		calculateTotalCostFn: func(_ context.Context, _ time.Time, _ time.Time, uid *uuid.UUID, sn *string) (int64, error) {
			require.NotNil(t, uid)
			assert.Equal(t, userID, *uid)
			require.NotNil(t, sn)
			assert.Equal(t, "Netflix", *sn)
			return 7200, nil
		},
	}

	svc := NewSubscriptionService(repo, silentLogger())
	uid := userID.String()
	sn := "Netflix"
	resp, err := svc.CalculateTotalCost(context.Background(), "01-2025", "12-2025", &uid, &sn)

	require.NoError(t, err)
	assert.Equal(t, int64(7200), resp.TotalCost)
}

func TestCalculateTotalCost_InvalidPeriodStart(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.CalculateTotalCost(context.Background(), "bad", "12-2025", nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid period_start")
}

func TestCalculateTotalCost_InvalidPeriodEnd(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.CalculateTotalCost(context.Background(), "01-2025", "bad", nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid period_end")
}

func TestCalculateTotalCost_EndBeforeStart(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())

	_, err := svc.CalculateTotalCost(context.Background(), "12-2025", "01-2025", nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "period_end must not be before period_start")
}

func TestCalculateTotalCost_InvalidUserID(t *testing.T) {
	svc := NewSubscriptionService(&mockRepo{}, silentLogger())
	bad := "not-a-uuid"

	_, err := svc.CalculateTotalCost(context.Background(), "01-2025", "12-2025", &bad, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
}
