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

type SeatHandler struct {
	seatService service.SeatService
	logger      *slog.Logger
}

func NewSeatHandler(seatService service.SeatService, logger *slog.Logger) *SeatHandler {
	return &SeatHandler{
		seatService: seatService,
		logger:      logger,
	}
}

func (h *SeatHandler) AddSeat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// limit the incoming request body size
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.AddSeatRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := h.seatService.AddSeat(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmptySeatNumber),
				errors.Is(err, service.ErrInvalidSeatClass),
				errors.Is(err, service.ErrInvalidHallID):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrHallNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, domain.ErrDuplicateSeat),
				errors.Is(err, domain.ErrDuplicateSeatNumber),
				errors.Is(err, domain.ErrSeatForeignKeyViolation):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				h.logger.Error("adding seat (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func (h *SeatHandler) GetSeatByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts Seat ID from URL
		seatIDStr := chi.URLParam(r, "seatID")
		seatID, err := strconv.ParseInt(seatIDStr, 10, 64)
		if err != nil || seatID <= 0 {
			http.Error(w, "invalid seat ID", http.StatusBadRequest)
			return
		}

		seat, err := h.seatService.GetSeatByID(r.Context(), seatID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidSeatID):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrSeatNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("getting seat by id (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(seat)
	}
}

func (h *SeatHandler) ListSeats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var seatList []service.SeatResponse
		var err error

		class := r.URL.Query().Get("class")
		if class == "" {
			seatList, err = h.seatService.ListAllSeats(r.Context())
			if err != nil {
				h.logger.Error("listing all seats (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		} else {
			seatList, err = h.seatService.ListSeatsByClass(r.Context(), domain.SeatClass(class))
			if err != nil {
				switch {
				case errors.Is(err, service.ErrInvalidSeatClass):
					http.Error(w, err.Error(), http.StatusBadRequest)
				default:
					h.logger.Error("listing seats by class (handler)", "err", err)
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(seatList)
	}
}

func (h *SeatHandler) ListSeatsByHallID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts hall ID from the URL
		hallIDStr := chi.URLParam(r, "hallID")
		hallID, err := strconv.ParseInt(hallIDStr, 10, 64)

		if err != nil || hallID <= 0 {
			http.Error(w, "invalid hall ID", http.StatusBadRequest)
			return
		}

		seatList, err := h.seatService.ListSeatsByHallID(r.Context(), hallID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidHallID):
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				h.logger.Error("listing seats by hall ID (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(seatList)
	}
}

func (h *SeatHandler) UpdateSeat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Decode and limit the incoming request body
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.UpdateSeatRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Extracts the seat ID from the URL
		seatIDStr := chi.URLParam(r, "seatID")
		seatID, err := strconv.ParseInt(seatIDStr, 10, 64)

		if err != nil || seatID <= 0 {
			http.Error(w, "invalid seat ID", http.StatusBadRequest)
			return
		}

		seat, err := h.seatService.UpdateSeat(r.Context(), seatID, req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidSeatID),
				errors.Is(err, service.ErrEmptySeatNumber),
				errors.Is(err, service.ErrInvalidSeatClass):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrSeatNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, domain.ErrDuplicateSeat),
				errors.Is(err, domain.ErrDuplicateSeatNumber):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				h.logger.Error("updating seats", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(seat)
	}
}

func (h *SeatHandler) RemoveSeat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts seatID from the URL
		seatIDStr := chi.URLParam(r, "seatID")
		seatID, err := strconv.ParseInt(seatIDStr, 10, 64)

		if err != nil || seatID <= 0 {
			http.Error(w, "invalid seat ID", http.StatusBadRequest)
			return
		}

		if err := h.seatService.RemoveSeat(r.Context(), seatID); err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidSeatID):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrSeatNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("removing seat (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
