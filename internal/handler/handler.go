package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mococa/gymshark-packs/internal/calculator"
	"github.com/mococa/gymshark-packs/internal/store"
)

const (
	maxCalculateBodySize = 1 << 10 // 1 KB — request only ever carries one integer
	maxPackSizeBodySize  = 1 << 10 // 1 KB
	maxPackSizeValue     = 1_000_000
	maxOrderValue        = 10_000_000
)

type Handler struct {
	store  store.PackSizeStore
	calc   *calculator.Calculator
	logger *slog.Logger
}

func New(store store.PackSizeStore, calc *calculator.Calculator, logger *slog.Logger) *Handler {
	return &Handler{store: store, calc: calc, logger: logger}
}

// CalculateRequest represents the request body for pack calculation
type CalculateRequest struct {
	Order int `json:"order" example:"501"`
}

// CalculateResponse represents the response for pack calculation
type CalculateResponse struct {
	Order      int         `json:"order" example:"501"`
	TotalItems int         `json:"total_items" example:"750"`
	TotalPacks int         `json:"total_packs" example:"2"`
	Packs      map[int]int `json:"packs" example:"250:1,500:1"`
}

// AddPackSizeRequest represents the request to add a new pack size
type AddPackSizeRequest struct {
	Size int `json:"size" example:"750"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request"`
}

// Calculate godoc
// @Summary Calculate optimal pack breakdown
// @Description Calculate the minimum number of packs needed to fulfill an order
// @Tags packs
// @Accept json
// @Produce json
// @Param request body CalculateRequest true "Order quantity"
// @Success 200 {object} CalculateResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/calculate [post]
func (h *Handler) Calculate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCalculateBodySize)

	var req CalculateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.logger.Warn("failed to decode request", "error", err)
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Order <= 0 {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "order must be greater than 0"})
		return
	}
	if req.Order > maxOrderValue {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "order exceeds maximum allowed value"})
		return
	}

	sizes, err := h.store.GetAll()
	if err != nil {
		h.logger.Error("failed to retrieve pack sizes", "error", err)
		h.respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}
	if len(sizes) == 0 {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "no pack sizes configured"})
		return
	}

	packs := h.calc.CalculatePacks(req.Order, sizes)

	resp := CalculateResponse{
		Order:      req.Order,
		TotalItems: calculator.TotalItems(packs),
		TotalPacks: calculator.TotalPacks(packs),
		Packs:      packs,
	}

	h.logger.Info("calculated packs",
		"order", req.Order,
		"total_items", resp.TotalItems,
		"total_packs", resp.TotalPacks,
	)

	h.respondJSON(w, http.StatusOK, resp)
}

// GetPackSizes godoc
// @Summary Get all pack sizes
// @Description Retrieve all configured pack sizes
// @Tags pack-sizes
// @Produce json
// @Success 200 {array} int
// @Router /api/pack-sizes [get]
func (h *Handler) GetPackSizes(w http.ResponseWriter, r *http.Request) {
	sizes, err := h.store.GetAll()
	if err != nil {
		h.logger.Error("failed to retrieve pack sizes", "error", err)
		h.respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}
	h.respondJSON(w, http.StatusOK, sizes)
}

// AddPackSize godoc
// @Summary Add a new pack size
// @Description Add a new pack size to the available options
// @Tags pack-sizes
// @Accept json
// @Produce json
// @Param request body AddPackSizeRequest true "Pack size to add"
// @Success 201 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/pack-sizes [post]
func (h *Handler) AddPackSize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPackSizeBodySize)

	var req AddPackSizeRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Size <= 0 {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "pack size must be greater than 0"})
		return
	}
	if req.Size > maxPackSizeValue {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "pack size exceeds maximum allowed value"})
		return
	}

	if err := h.store.Add(req.Size); err != nil {
		if errors.Is(err, store.ErrPackSizeExists) {
			h.respondJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("failed to add pack size", "size", req.Size, "error", err)
		h.respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}

	h.logger.Info("added pack size", "size", req.Size)
	h.respondJSON(w, http.StatusCreated, map[string]string{"message": "pack size added"})
}

// DeletePackSize godoc
// @Summary Delete a pack size
// @Description Remove a pack size from the available options
// @Tags pack-sizes
// @Produce json
// @Param size path int true "Pack size to delete"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Router /api/pack-sizes/{size} [delete]
func (h *Handler) DeletePackSize(w http.ResponseWriter, r *http.Request) {
	sizeStr := chi.URLParam(r, "size")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 {
		h.respondJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid pack size"})
		return
	}

	if err := h.store.Remove(size); err != nil {
		if errors.Is(err, store.ErrLastPackSize) {
			h.respondJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, store.ErrPackSizeNotFound) {
			h.respondJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("failed to delete pack size", "size", size, "error", err)
		h.respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
		return
	}

	h.logger.Info("removed pack size", "size", size)
	w.WriteHeader(http.StatusNoContent)
}

// Health godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}
