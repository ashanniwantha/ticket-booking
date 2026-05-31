package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/ashanniwantha/ticket-booking/internal/service"
	"github.com/go-chi/chi/v5"
)

type HallHandler struct {
	hallService service.HallService
	logger      *slog.Logger
}

func NewHallHandler(hallService service.HallService, logger *slog.Logger) *HallHandler {
	return &HallHandler{
		hallService: hallService,
		logger:      logger,
	}
}

func (h *HallHandler) AddHall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode request body
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.AddHallRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		hallResp, err := h.hallService.AddHall(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptyHallName):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrDuplicateHallName):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				h.logger.Error("add hall (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(hallResp)
	}
}

func (h *HallHandler) GetHall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts HallID from URL
		hallIDStr := chi.URLParam(r, "hall_id")
		hallID, err := strconv.ParseInt(hallIDStr, 10, 64)
		if err != nil || hallID <= 0 {
			http.Error(w, "invalid hall ID", http.StatusBadRequest)
			return
		}

		hallResp, err := h.hallService.GetHall(r.Context(), hallID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidHallID):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrHallNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("view hall (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hallResp)
	}
}

func (h *HallHandler) UpdateHall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts HallID from URL
		hallIDStr := chi.URLParam(r, "hall_id")
		hallID, err := strconv.ParseInt(hallIDStr, 10, 64)
		if err != nil || hallID <= 0 {
			http.Error(w, "invalid hall ID", http.StatusBadRequest)
			return
		}

		// Decode request body
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.UpdateHallRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		hallResp, err := h.hallService.UpdateHall(r.Context(), hallID, req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidHallID),
				errors.Is(err, service.ErrEmptyHallName):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrDuplicateHallName):
				http.Error(w, err.Error(), http.StatusConflict)
			case errors.Is(err, domain.ErrHallNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("update hall (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(hallResp)
	}
}

func (h *HallHandler) RemoveHall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts the HallID from URL
		hallIDStr := chi.URLParam(r, "hall_id")
		hallID, err := strconv.ParseInt(hallIDStr, 10, 64)
		if err != nil || hallID <= 0 {
			http.Error(w, "invalid hall ID", http.StatusBadRequest)
			return
		}

		if err := h.hallService.RemoveHall(r.Context(), hallID); err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidHallID):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrHallNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("remove hall (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
