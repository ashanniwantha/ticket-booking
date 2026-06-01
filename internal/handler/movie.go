package handler

import (
	"errors"
	"net/http"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/ashanniwantha/ticket-booking/internal/service"
)

type MovieHandler struct {
	*BaseHandler // Promotes all base utilities directly to h
	movieService service.MovieService
}

func NewMovieHandler(base *BaseHandler, movieService service.MovieService) *MovieHandler {
	return &MovieHandler{
		BaseHandler:  base,
		movieService: movieService,
	}
}

func (h *MovieHandler) AddMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req service.AddMovieRequest
		if !h.DecodeJSON(w, r, &req) {
			return
		}

		movie, err := h.movieService.AddMovie(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieTitleEmpty):
				h.RespondError(w, http.StatusBadRequest, err.Error())
			default:
				h.logger.Error("adding movie failed (handler)", "err", err)
				h.RespondError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}

		h.Respond(w, http.StatusCreated, movie)
	}
}

func (h *MovieHandler) GetMovieByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		movieID, err := h.ParseIDParam(r, "movie_id")
		if err != nil {
			h.RespondError(w, http.StatusBadRequest, "invalid movie ID")
			return
		}

		movie, err := h.movieService.GetMovieByID(r.Context(), movieID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieNotFound):
				h.RespondError(w, http.StatusNotFound, err.Error())
			default:
				h.logger.Error("getting movie by id failed (handler)", "err", err, "movie_id", movieID)
				h.RespondError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		h.Respond(w, http.StatusOK, movie)
	}
}

func (h *MovieHandler) ListMovies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var moviesList []service.MovieResponse
		var err error

		title := r.URL.Query().Get("title")

		if title != "" {
			moviesList, err = h.movieService.ListMovieByTitle(r.Context(), title)
			h.logger.Info("listing movies by title (handler)", "title", title)
		} else {
			moviesList, err = h.movieService.ListAllMovies(r.Context())
			h.logger.Info("listing all movies (handler)")
		}

		if err != nil {
			h.logger.Error("failed to fetch movies list (handler)", "err", err, "filtered_by_title", title != "")
			h.RespondError(w, http.StatusInternalServerError, "internal error")
			return
		}
		h.Respond(w, http.StatusOK, moviesList)
	}
}

func (h *MovieHandler) UpdateMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		movieID, err := h.ParseIDParam(r, "movie_id")
		if err != nil {
			h.RespondError(w, http.StatusBadRequest, "invalid movie ID")
			return
		}

		var req service.UpdateMovieRequest
		if !h.DecodeJSON(w, r, &req) {
			return
		}

		movie, err := h.movieService.UpdateMovie(r.Context(), movieID, req)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieNotFound):
				h.RespondError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, domain.ErrMovieTitleEmpty):
				h.RespondError(w, http.StatusBadRequest, err.Error())
			default:
				h.logger.Error("updating movie failed (handler)", "err", err, "movie_id", movieID)
				h.RespondError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		h.Respond(w, http.StatusOK, movie)
	}
}

func (h *MovieHandler) RemoveMovie() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		movieID, err := h.ParseIDParam(r, "movie_id")
		if err != nil {
			h.RespondError(w, http.StatusBadRequest, "invalid movie ID")
			return
		}

		if err := h.movieService.RemoveMovie(r.Context(), movieID); err != nil {
			switch {
			case errors.Is(err, domain.ErrMovieNotFound):
				h.RespondError(w, http.StatusNotFound, err.Error())
			default:
				h.logger.Error("removing movie failed (handler)", "err", err, "movie_id", movieID)
				h.RespondError(w, http.StatusInternalServerError, "internal error")
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
