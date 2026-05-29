package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/ashanniwantha/ticket-booking/internal/service"
)

type AuthHandler struct {
	userService service.UserService
	logger      *slog.Logger
}

func NewAuthHandler(userService service.UserService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		logger:      logger,
	}
}

func (h *AuthHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.RegisterRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		userResp, err := h.userService.Register(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrUsernameOrEmailEmpty),
				errors.Is(err, service.ErrPasswordEmpty),
				errors.Is(err, service.ErrPasswordTooLong):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, domain.ErrDuplicateEmail),
				errors.Is(err, domain.ErrDuplicateUsername):
				http.Error(w, err.Error(), http.StatusConflict)
			default:
				h.logger.Error("register user", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(userResp)
	}
}

func (h *AuthHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.LoginRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		token, err := h.userService.Login(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrMissingCredentials):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, service.ErrInvalidCredentials):
				http.Error(w, err.Error(), http.StatusUnauthorized)
			default:
				h.logger.Error("login user", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}
