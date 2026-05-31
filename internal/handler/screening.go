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

type ScreeningHandler struct {
	screeningService service.ScreeningService
	logger           *slog.Logger
}

func NewScreeningHandler(svc service.ScreeningService, logger *slog.Logger) *ScreeningHandler {
	return &ScreeningHandler{
		screeningService: svc,
		logger:           logger,
	}
}

func (h *ScreeningHandler) AddScreening() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode and limit the incoming request body
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.AddScreeningReq
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		screening, err := h.screeningService.AddScreening(r.Context(), req)

		if err != nil {
			switch {

			case errors.Is(err, service.ErrInvalidMovieID),
				errors.Is(err, service.ErrInvalidHallID),
				errors.Is(err, service.ErrInvalidSchedule),
				errors.Is(err, domain.ErrInvalidScreeningData):
				http.Error(w, err.Error(), http.StatusBadRequest)

			case errors.Is(err, domain.ErrMovieNotFound),
				errors.Is(err, domain.ErrHallNotFound),
				errors.Is(err, domain.ErrScreeningForeignKeyViolation):
				http.Error(w, err.Error(), http.StatusNotFound)

			case errors.Is(err, domain.ErrScreeningTimeConflict):
				http.Error(w, err.Error(), http.StatusConflict)

			default:
				h.logger.Error("adding screening (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(screening)
	}
}

func (h *ScreeningHandler) GetScreeningByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts screening ID from the URL
		screeningIDStr := chi.URLParam(r, "screening_id")
		screeningID, err := strconv.ParseInt(screeningIDStr, 10, 64)

		if err != nil || screeningID <= 0 {
			http.Error(w, "invalid screening ID", http.StatusBadRequest)
			return
		}

		screening, err := h.screeningService.GetScreeningByID(r.Context(), screeningID)
		if err != nil {
			if errors.Is(err, domain.ErrScreeningNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}

			h.logger.Error("getting screening by id (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(screening)
	}

}

func (h *ScreeningHandler) ListScreeningsByMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var screeningList []service.ScreeningResponse
		var err error

		// Extracts movie ID from the URL
		movieIDStr := chi.URLParam(r, "movie_id")
		movieID, convErr := strconv.ParseInt(movieIDStr, 10, 64)

		if convErr != nil || movieID <= 0 {
			http.Error(w, "invalid movie id", http.StatusBadRequest)
			return
		}

		// Extracts hall ID from query if available
		hallIDStr := r.URL.Query().Get("hall_id")

		// Check if hall ID available if so use it
		if hallIDStr != "" {
			hallID, convErr := strconv.ParseInt(hallIDStr, 10, 64)

			if convErr != nil || hallID <= 0 {
				http.Error(w, "invalid hall ID", http.StatusBadRequest)
				return
			}
			h.logger.Info("listing screenings by movie and hall (handler)")
			screeningList, err = h.screeningService.ListScreeningsByMovieAndHall(
				r.Context(), movieID, hallID,
			)
		} else {
			h.logger.Info("listing screenings by movie")
			screeningList, err = h.screeningService.ListScreeningsByMovie(
				r.Context(), movieID,
			)
		}

		if err != nil {
			h.logger.Error("listing screenings just by movie (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(screeningList)
	}
}

func (h *ScreeningHandler) ListScreeningsByHall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var screeningList []service.ScreeningResponse
		var err error

		// Extracts hall ID from the URL
		hallIDStr := chi.URLParam(r, "hall_id")

		hallID, convErr := strconv.ParseInt(hallIDStr, 10, 64)
		if convErr != nil || hallID <= 0 {
			http.Error(w, "invalid hall ID", http.StatusBadRequest)
			return
		}

		// Extracts movie ID from query if available
		movieIDStr := r.URL.Query().Get("movie_id")

		// Check if movie ID available if so use it
		if movieIDStr != "" {
			movieID, convErr := strconv.ParseInt(movieIDStr, 10, 64)

			if convErr != nil || movieID <= 0 {
				http.Error(w, "invalid movie ID", http.StatusBadRequest)
				return
			}
			h.logger.Info("listing screenings by hall and movie (handler)")
			screeningList, err = h.screeningService.ListScreeningsByMovieAndHall(
				r.Context(), movieID, hallID,
			)

		} else {
			h.logger.Info("listing screenings just by hall (handler)")
			screeningList, err = h.screeningService.ListScreeningsByHall(
				r.Context(), hallID,
			)
		}

		if err != nil {
			h.logger.Error("listing screenings by hall (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(screeningList)
	}
}

func (h *ScreeningHandler) ListAllScreenings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		screeningList, err := h.screeningService.ListAllScreenings(r.Context())

		if err != nil {
			h.logger.Error("listing all screening (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(screeningList)
	}
}

func (h *ScreeningHandler) UpdateScreening() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts screeningID from URL
		screeningIDStr := chi.URLParam(r, "screening_id")

		screeningID, err := strconv.ParseInt(screeningIDStr, 10, 64)
		if err != nil || screeningID <= 0 {
			http.Error(w, "invalid screening ID", http.StatusBadRequest)
			return
		}

		// Decode and limit incoming request body
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.UpdateScreeningReq
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		screening, err := h.screeningService.UpdateScreening(r.Context(), screeningID, req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidMovieID),
				errors.Is(err, service.ErrInvalidHallID),
				errors.Is(err, service.ErrInvalidSchedule),
				errors.Is(err, domain.ErrInvalidScreeningData):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrScreeningNotFound),
				errors.Is(err, domain.ErrMovieNotFound),
				errors.Is(err, domain.ErrHallNotFound),
				errors.Is(err, domain.ErrScreeningForeignKeyViolation):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, domain.ErrScreeningTimeConflict):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				h.logger.Error("updating screening (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(screening)
	}
}

func (h *ScreeningHandler) RemoveScreening() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts screening ID
		screeningIDStr := chi.URLParam(r, "screening_id")

		screeningID, err := strconv.ParseInt(screeningIDStr, 10, 64)
		if err != nil || screeningID <= 0 {
			http.Error(w, "invalid screening ID", http.StatusBadRequest)
			return
		}

		if err := h.screeningService.RemoveScreening(r.Context(), screeningID); err != nil {
			if errors.Is(err, domain.ErrScreeningNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}

			h.logger.Error("deleting screening (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
