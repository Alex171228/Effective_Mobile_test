package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/alexs/subscription-service/internal/model"
)

type SubscriptionHandler struct {
	service SubscriptionSvc
	logger  *slog.Logger
}

func NewSubscriptionHandler(svc SubscriptionSvc, logger *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *SubscriptionHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/subscriptions", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/cost", h.CalculateTotalCost)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

// Create godoc
// @Summary      Создать подписку
// @Description  Создание новой записи о подписке пользователя
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        subscription  body      model.CreateSubscriptionRequest  true  "Данные подписки"
// @Success      201           {object}  model.SubscriptionResponse
// @Failure      400           {object}  model.ErrorResponse
// @Failure      500           {object}  model.ErrorResponse
// @Router       /subscriptions [post]
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.Create(r.Context(), req)
	if err != nil {
		if isBadRequest(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// GetByID godoc
// @Summary      Получить подписку по ID
// @Description  Получение записи о подписке по её идентификатору
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "ID подписки (UUID)"
// @Success      200  {object}  model.SubscriptionResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /subscriptions/{id} [get]
func (h *SubscriptionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if isBadRequest(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// Update godoc
// @Summary      Обновить подписку
// @Description  Обновление записи о подписке по ID
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id            path      string                           true  "ID подписки (UUID)"
// @Param        subscription  body      model.UpdateSubscriptionRequest  true  "Обновлённые данные"
// @Success      200           {object}  model.SubscriptionResponse
// @Failure      400           {object}  model.ErrorResponse
// @Failure      404           {object}  model.ErrorResponse
// @Failure      500           {object}  model.ErrorResponse
// @Router       /subscriptions/{id} [put]
func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		if isNotFound(err) {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if isBadRequest(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// Delete godoc
// @Summary      Удалить подписку
// @Description  Удаление записи о подписке по ID
// @Tags         subscriptions
// @Param        id   path  string  true  "ID подписки (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Router       /subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.service.Delete(r.Context(), id); err != nil {
		if isNotFound(err) {
			h.respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if isBadRequest(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List godoc
// @Summary      Список подписок
// @Description  Получение списка подписок с пагинацией и фильтрацией
// @Tags         subscriptions
// @Produce      json
// @Param        user_id       query     string  false  "Фильтр по ID пользователя (UUID)"
// @Param        service_name  query     string  false  "Фильтр по названию сервиса"
// @Param        limit         query     int     false  "Лимит записей"  default(10)
// @Param        offset        query     int     false  "Смещение"       default(0)
// @Success      200           {object}  model.ListSubscriptionsResponse
// @Failure      400           {object}  model.ErrorResponse
// @Failure      500           {object}  model.ErrorResponse
// @Router       /subscriptions [get]
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	var userID, serviceName *string
	if v := r.URL.Query().Get("user_id"); v != "" {
		userID = &v
	}
	if v := r.URL.Query().Get("service_name"); v != "" {
		serviceName = &v
	}

	resp, err := h.service.List(r.Context(), userID, serviceName, limit, offset)
	if err != nil {
		if isBadRequest(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// CalculateTotalCost godoc
// @Summary      Рассчитать суммарную стоимость подписок
// @Description  Подсчёт суммарной стоимости всех подписок за выбранный период с фильтрацией
// @Tags         subscriptions
// @Produce      json
// @Param        period_start  query     string  true   "Начало периода (MM-YYYY)"  example(01-2025)
// @Param        period_end    query     string  true   "Конец периода (MM-YYYY)"   example(12-2025)
// @Param        user_id       query     string  false  "Фильтр по ID пользователя (UUID)"
// @Param        service_name  query     string  false  "Фильтр по названию сервиса"
// @Success      200           {object}  model.CostResponse
// @Failure      400           {object}  model.ErrorResponse
// @Failure      500           {object}  model.ErrorResponse
// @Router       /subscriptions/cost [get]
func (h *SubscriptionHandler) CalculateTotalCost(w http.ResponseWriter, r *http.Request) {
	periodStart := r.URL.Query().Get("period_start")
	periodEnd := r.URL.Query().Get("period_end")

	if periodStart == "" || periodEnd == "" {
		h.respondError(w, http.StatusBadRequest, "period_start and period_end are required (format: MM-YYYY)")
		return
	}

	var userID, serviceName *string
	if v := r.URL.Query().Get("user_id"); v != "" {
		userID = &v
	}
	if v := r.URL.Query().Get("service_name"); v != "" {
		serviceName = &v
	}

	resp, err := h.service.CalculateTotalCost(r.Context(), periodStart, periodEnd, userID, serviceName)
	if err != nil {
		if isBadRequest(err) {
			h.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

func (h *SubscriptionHandler) respondJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", slog.String("error", err.Error()))
	}
}

func (h *SubscriptionHandler) respondError(w http.ResponseWriter, code int, message string) {
	h.logger.Warn("request error",
		slog.Int("status", code),
		slog.String("error", message),
	)
	h.respondJSON(w, code, model.ErrorResponse{Error: message})
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

func isBadRequest(err error) bool {
	return strings.Contains(err.Error(), "invalid") ||
		strings.Contains(err.Error(), "required") ||
		strings.Contains(err.Error(), "must be")
}
