package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mococa/gymshark-packs/internal/calculator"
	"github.com/mococa/gymshark-packs/internal/store"
)

func TestCalculate(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	packStore := store.NewInMemoryStore([]int{250, 500, 1000})
	h := New(packStore, calculator.New(), logger)

	t.Run("valid request", func(t *testing.T) {
		body := bytes.NewBufferString(`{"order": 501}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		h.Calculate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp CalculateResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Order != 501 {
			t.Errorf("order = %d, want 501", resp.Order)
		}
		if resp.TotalPacks != 2 {
			t.Errorf("total_packs = %d, want 2", resp.TotalPacks)
		}
	})

	t.Run("invalid order", func(t *testing.T) {
		body := bytes.NewBufferString(`{"order": -1}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
		w := httptest.NewRecorder()

		h.Calculate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		body := bytes.NewBufferString(`{"order": 100, "junk": "data"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
		w := httptest.NewRecorder()

		h.Calculate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (should reject unknown fields)", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("trailing JSON rejected", func(t *testing.T) {
		body := bytes.NewBufferString(`{"order": 100}{"extra": "data"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
		w := httptest.NewRecorder()

		h.Calculate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (should reject trailing JSON)", w.Code, http.StatusBadRequest)
		}
	})
}

func TestAddPackSize(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	packStore := store.NewInMemoryStore([]int{250})
	h := New(packStore, calculator.New(), logger)

	t.Run("add valid size", func(t *testing.T) {
		body := bytes.NewBufferString(`{"size": 500}`)
		req := httptest.NewRequest(http.MethodPost, "/api/pack-sizes", body)
		w := httptest.NewRecorder()

		h.AddPackSize(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}

		if !packStore.Exists(500) {
			t.Error("pack size 500 should exist after adding")
		}
	})

	t.Run("add duplicate size", func(t *testing.T) {
		body := bytes.NewBufferString(`{"size": 500}`)
		req := httptest.NewRequest(http.MethodPost, "/api/pack-sizes", body)
		w := httptest.NewRecorder()

		h.AddPackSize(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
		}
	})
}

func TestDeletePackSize(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	t.Run("delete existing size", func(t *testing.T) {
		packStore := store.NewInMemoryStore([]int{250, 500})
		h := New(packStore, calculator.New(), logger)

		req := httptest.NewRequest(http.MethodDelete, "/api/pack-sizes/500", nil)

		// Add chi URL params to context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("size", "500")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.DeletePackSize(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
		}

		if packStore.Exists(500) {
			t.Error("pack size 500 should not exist after deleting")
		}
	})

	t.Run("cannot delete last pack size", func(t *testing.T) {
		packStore := store.NewInMemoryStore([]int{250})
		h := New(packStore, calculator.New(), logger)

		req := httptest.NewRequest(http.MethodDelete, "/api/pack-sizes/250", nil)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("size", "250")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.DeletePackSize(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
		}

		if !packStore.Exists(250) {
			t.Error("last pack size should still exist after failed delete")
		}
	})
}

func TestGetPackSizes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	packStore := store.NewInMemoryStore([]int{250, 500, 1000})
	h := New(packStore, calculator.New(), logger)

	req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	w := httptest.NewRecorder()

	h.GetPackSizes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var sizes []int
	if err := json.NewDecoder(w.Body).Decode(&sizes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expected := []int{250, 500, 1000}
	if len(sizes) != len(expected) {
		t.Errorf("got %d sizes, want %d", len(sizes), len(expected))
	}
}

func TestHealth(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	packStore := store.NewInMemoryStore([]int{250})
	h := New(packStore, calculator.New(), logger)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}
