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

type MovieHandler struct {
	movieService service.MovieService
	logger       *slog.Logger
}

func NewMovieHandler(movieService service.MovieService, logger *slog.Logger) *MovieHandler {
	return &MovieHandler{
		movieService: movieService,
		logger:       logger,
	}
}

func (h *MovieHandler) AddMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.AddMovieRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		movie, err := h.movieService.AddMovie(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieTitleEmpty):
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				h.logger.Error("adding movie (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(movie)
	}
}

func (h *MovieHandler) GetMovieByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		movieIDStr := chi.URLParam(r, "movie_id")
		movieID, err := strconv.ParseInt(movieIDStr, 10, 64)

		if err != nil || movieID <= 0 {
			http.Error(w, "invalid movie ID", http.StatusBadRequest)
			return
		}

		movie, err := h.movieService.GetMovieByID(r.Context(), movieID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("getting movie by id (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(movie)
	}
}

func (h *MovieHandler) ListMovies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var moviesList []service.MovieResponse
		var err error

		title := r.URL.Query().Get("title")

		if title != "" {
			moviesList, err = h.movieService.ListMovieByTitle(r.Context(), title)
			h.logger.Info("listing movies by title (handler)")
		} else {
			moviesList, err = h.movieService.ListAllMovies(r.Context())
			h.logger.Info("listing all movies (handler)")
		}

		if err != nil {
			h.logger.Error("getting all movies (handler)", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(moviesList)

	}
}

func (h *MovieHandler) UpdateMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract Movie ID
		movieIDStr := chi.URLParam(r, "movie_id")
		movieID, err := strconv.ParseInt(movieIDStr, 10, 64)

		if err != nil || movieID <= 0 {
			http.Error(w, "invalid movie ID", http.StatusBadRequest)
			return
		}

		// Decode and limit the incoming request body
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		var req service.UpdateMovieRequest
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		movie, err := h.movieService.UpdateMovie(r.Context(), movieID, req)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, domain.ErrMovieTitleEmpty):
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				h.logger.Error("updating movie (handler)", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(movie)
	}
}

func (h *MovieHandler) RemoveMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extracts movie ID
		movieIDStr := chi.URLParam(r, "movie_id")
		movieID, err := strconv.ParseInt(movieIDStr, 10, 64)

		if err != nil || movieID <= 0 {
			http.Error(w, "invalid movie ID", http.StatusBadRequest)
			return
		}

		if err := h.movieService.RemoveMovie(r.Context(), movieID); err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				h.logger.Error("removing movie", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
