package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexs/subscription-service/internal/model"
)

type mockService struct {
	createFn             func(ctx context.Context, req model.CreateSubscriptionRequest) (model.SubscriptionResponse, error)
	getByIDFn            func(ctx context.Context, id string) (model.SubscriptionResponse, error)
	updateFn             func(ctx context.Context, id string, req model.UpdateSubscriptionRequest) (model.SubscriptionResponse, error)
	deleteFn             func(ctx context.Context, id string) error
	listFn               func(ctx context.Context, userID, serviceName *string, limit, offset int) (model.ListSubscriptionsResponse, error)
	calculateTotalCostFn func(ctx context.Context, periodStart, periodEnd string, userID, serviceName *string) (model.CostResponse, error)
}

func (m *mockService) Create(ctx context.Context, req model.CreateSubscriptionRequest) (model.SubscriptionResponse, error) {
	return m.createFn(ctx, req)
}
func (m *mockService) GetByID(ctx context.Context, id string) (model.SubscriptionResponse, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockService) Update(ctx context.Context, id string, req model.UpdateSubscriptionRequest) (model.SubscriptionResponse, error) {
	return m.updateFn(ctx, id, req)
}
func (m *mockService) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}
func (m *mockService) List(ctx context.Context, userID, serviceName *string, limit, offset int) (model.ListSubscriptionsResponse, error) {
	return m.listFn(ctx, userID, serviceName, limit, offset)
}
func (m *mockService) CalculateTotalCost(ctx context.Context, periodStart, periodEnd string, userID, serviceName *string) (model.CostResponse, error) {
	return m.calculateTotalCostFn(ctx, periodStart, periodEnd, userID, serviceName)
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupRouter(h *SubscriptionHandler) *chi.Mux {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	return r
}

// --- Create ---

func TestCreateHandler_Success(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, req model.CreateSubscriptionRequest) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{
				ID:          "test-id",
				ServiceName: req.ServiceName,
				Price:       req.Price,
				UserID:      req.UserID,
				StartDate:   req.StartDate,
				CreatedAt:   "2025-07-01T00:00:00Z",
				UpdatedAt:   "2025-07-01T00:00:00Z",
			}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	body := `{"service_name":"Yandex Plus","price":400,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp model.SubscriptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Yandex Plus", resp.ServiceName)
	assert.Equal(t, 400, resp.Price)
}

func TestCreateHandler_InvalidJSON(t *testing.T) {
	h := NewSubscriptionHandler(&mockService{}, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader("{invalid}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateHandler_ValidationError(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ model.CreateSubscriptionRequest) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{}, fmt.Errorf("invalid user_id: bad format")
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	body := `{"service_name":"Test","price":100,"user_id":"bad","start_date":"01-2025"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- GetByID ---

func TestGetByIDHandler_Success(t *testing.T) {
	svc := &mockService{
		getByIDFn: func(_ context.Context, id string) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{
				ID:          id,
				ServiceName: "Netflix",
				Price:       600,
			}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/some-id", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp model.SubscriptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Netflix", resp.ServiceName)
}

func TestGetByIDHandler_NotFound(t *testing.T) {
	svc := &mockService{
		getByIDFn: func(_ context.Context, id string) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{}, fmt.Errorf("subscription %s not found", id)
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/missing-id", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Update ---

func TestUpdateHandler_Success(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, id string, req model.UpdateSubscriptionRequest) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{
				ID:          id,
				ServiceName: req.ServiceName,
				Price:       req.Price,
				StartDate:   req.StartDate,
			}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	body := `{"service_name":"Updated","price":999,"start_date":"01-2025"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/some-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp model.SubscriptionResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Updated", resp.ServiceName)
	assert.Equal(t, 999, resp.Price)
}

func TestUpdateHandler_InvalidJSON(t *testing.T) {
	h := NewSubscriptionHandler(&mockService{}, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/some-id", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateHandler_NotFound(t *testing.T) {
	svc := &mockService{
		updateFn: func(_ context.Context, id string, _ model.UpdateSubscriptionRequest) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{}, fmt.Errorf("subscription %s not found", id)
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	body := `{"service_name":"X","price":1,"start_date":"01-2025"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/subscriptions/missing", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- Delete ---

func TestDeleteHandler_Success(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _ string) error {
			return nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/some-id", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteHandler_NotFound(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, id string) error {
			return fmt.Errorf("subscription %s not found", id)
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/subscriptions/missing", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- List ---

func TestListHandler_Success(t *testing.T) {
	svc := &mockService{
		listFn: func(_ context.Context, _ *string, _ *string, limit, offset int) (model.ListSubscriptionsResponse, error) {
			assert.Equal(t, 10, limit)
			assert.Equal(t, 0, offset)
			return model.ListSubscriptionsResponse{
				Subscriptions: []model.SubscriptionResponse{
					{ID: "1", ServiceName: "Netflix"},
				},
				Total: 1,
			}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp model.ListSubscriptionsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Subscriptions, 1)
}

func TestListHandler_WithFilters(t *testing.T) {
	svc := &mockService{
		listFn: func(_ context.Context, userID, serviceName *string, limit, offset int) (model.ListSubscriptionsResponse, error) {
			require.NotNil(t, userID)
			assert.Equal(t, "some-user-id", *userID)
			require.NotNil(t, serviceName)
			assert.Equal(t, "Netflix", *serviceName)
			assert.Equal(t, 5, limit)
			assert.Equal(t, 10, offset)
			return model.ListSubscriptionsResponse{Total: 0, Subscriptions: []model.SubscriptionResponse{}}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions?user_id=some-user-id&service_name=Netflix&limit=5&offset=10", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListHandler_LimitCap(t *testing.T) {
	svc := &mockService{
		listFn: func(_ context.Context, _ *string, _ *string, limit, _ int) (model.ListSubscriptionsResponse, error) {
			assert.Equal(t, 100, limit)
			return model.ListSubscriptionsResponse{Total: 0, Subscriptions: []model.SubscriptionResponse{}}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions?limit=9999", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// --- CalculateTotalCost ---

func TestCalculateTotalCostHandler_Success(t *testing.T) {
	svc := &mockService{
		calculateTotalCostFn: func(_ context.Context, start, end string, _ *string, _ *string) (model.CostResponse, error) {
			assert.Equal(t, "01-2025", start)
			assert.Equal(t, "12-2025", end)
			return model.CostResponse{TotalCost: 9600}, nil
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/cost?period_start=01-2025&period_end=12-2025", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp model.CostResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, int64(9600), resp.TotalCost)
}

func TestCalculateTotalCostHandler_MissingParams(t *testing.T) {
	h := NewSubscriptionHandler(&mockService{}, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/cost", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCalculateTotalCostHandler_MissingPeriodEnd(t *testing.T) {
	h := NewSubscriptionHandler(&mockService{}, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/cost?period_start=01-2025", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCalculateTotalCostHandler_ValidationError(t *testing.T) {
	svc := &mockService{
		calculateTotalCostFn: func(_ context.Context, _, _ string, _ *string, _ *string) (model.CostResponse, error) {
			return model.CostResponse{}, fmt.Errorf("invalid period_start (expected MM-YYYY): bad")
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/cost?period_start=bad&period_end=12-2025", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateHandler_InternalError(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ model.CreateSubscriptionRequest) (model.SubscriptionResponse, error) {
			return model.SubscriptionResponse{}, fmt.Errorf("database connection lost")
		},
	}

	h := NewSubscriptionHandler(svc, silentLogger())
	r := setupRouter(h)

	body := `{"service_name":"Test","price":100,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"01-2025"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
