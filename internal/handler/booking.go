package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ashanniwantha/ticket-booking/internal/auth"
	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/ashanniwantha/ticket-booking/internal/service"
	"github.com/go-chi/chi/v5"
)

type BookingHandler struct {
	bookingService service.BookingService
	logger         *slog.Logger
}

func NewBookingHandler(bookingService service.BookingService,
	logger *slog.Logger) *BookingHandler {
	return &BookingHandler{
		bookingService: bookingService,
		logger:         logger,
	}
}

func (h *BookingHandler) HoldSeat() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID of authenticated user from the context
		userID, ok := auth.UserIDFromContext(r)
		if !ok || userID <= 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		// Extracts request body into a separate struct
		var body struct {
			ScreeningID int64 `json:"screening_id"`
			SeatID      int64 `json:"seat_id"`
		}
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		// Build service request with the authenticated user
		req := service.HoldSeatRequest{
			ScreeningID: body.ScreeningID,
			SeatID:      body.SeatID,
			UserID:      userID,
		}

		ticket, err := h.bookingService.HoldSeat(r.Context(), req)
		if err != nil {

			switch {
			case errors.Is(err, service.ErrSeatScreeningHallMismatch),
				errors.Is(err, service.ErrInvalidScreeningID),
				errors.Is(err, service.ErrInvalidSeatID),
				errors.Is(err, service.ErrInvalidUserID):
				http.Error(w, err.Error(), http.StatusBadRequest)
				return

			case errors.Is(err, domain.ErrScreeningNotFound),
				errors.Is(err, domain.ErrSeatNotFound),
				errors.Is(err, domain.ErrUserNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
				return

			case errors.Is(err, domain.ErrSeatUnavailable):
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}

			h.logger.Error("hold seat (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ticket)
	}
}

func (h *BookingHandler) ConfirmHold() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extarcts ticket ID from URL
		ticketIDStr := chi.URLParam(r, "ticket_id")

		ticketID, err := strconv.ParseInt(ticketIDStr, 10, 64)
		if err != nil || ticketID <= 0 {
			http.Error(w, "invalid ticket ID", http.StatusBadRequest)
			return
		}

		// Get user ID of authenticated user from the context
		userID, ok := auth.UserIDFromContext(r)
		if !ok || userID <= 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		err = h.bookingService.ConfirmHold(r.Context(), ticketID, userID)
		if err != nil {

			switch {
			case errors.Is(err, service.ErrInvalidTicketID),
				errors.Is(err, service.ErrInvalidUserID):
				http.Error(w, err.Error(), http.StatusBadRequest)

			case errors.Is(err, domain.ErrTicketNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)

			case errors.Is(err, service.ErrTicketNotHold):
				http.Error(w, err.Error(), http.StatusConflict)

			default:
				h.logger.Error("confirm hold (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "confirmed"})
	}
}

func (h *BookingHandler) CancelTicket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extarcts ticket ID from URL
		ticketIDStr := chi.URLParam(r, "ticket_id")

		ticketID, err := strconv.ParseInt(ticketIDStr, 10, 64)
		if err != nil || ticketID <= 0 {
			http.Error(w, "invalid ticket ID", http.StatusBadRequest)
			return
		}

		// Get user ID of authenticated user from the context
		userID, ok := auth.UserIDFromContext(r)
		if !ok || userID <= 0 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		err = h.bookingService.CancelTicket(r.Context(), ticketID, userID)
		if err != nil {

			switch {
			case errors.Is(err, domain.ErrTicketNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)

			case errors.Is(err, service.ErrTicketAlreadyCancelled):
				http.Error(w, err.Error(), http.StatusConflict)

			default:
				h.logger.Error("cancel ticket (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
	}
}
