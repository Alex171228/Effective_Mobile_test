package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToSubscriptionResponse_WithoutEndDate(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	start := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)

	sub := Subscription{
		ID:          id,
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      userID,
		StartDate:   start,
		EndDate:     nil,
		CreatedAt:   created,
		UpdatedAt:   created,
	}

	resp := ToSubscriptionResponse(sub)

	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "Yandex Plus", resp.ServiceName)
	assert.Equal(t, 400, resp.Price)
	assert.Equal(t, userID.String(), resp.UserID)
	assert.Equal(t, "07-2025", resp.StartDate)
	assert.Nil(t, resp.EndDate)
	assert.Equal(t, created.Format(time.RFC3339), resp.CreatedAt)
	assert.Equal(t, created.Format(time.RFC3339), resp.UpdatedAt)
}

func TestToSubscriptionResponse_WithEndDate(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	sub := Subscription{
		ID:          id,
		ServiceName: "Netflix",
		Price:       600,
		UserID:      userID,
		StartDate:   start,
		EndDate:     &end,
		CreatedAt:   created,
		UpdatedAt:   created,
	}

	resp := ToSubscriptionResponse(sub)

	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, "Netflix", resp.ServiceName)
	assert.Equal(t, 600, resp.Price)
	assert.Equal(t, "01-2025", resp.StartDate)
	assert.NotNil(t, resp.EndDate)
	assert.Equal(t, "12-2025", *resp.EndDate)
}
