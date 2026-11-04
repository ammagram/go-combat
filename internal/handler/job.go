package handler

import (
	"encoding/json"
	"errors"
	"jobflow/internal/httpx"
	"jobflow/internal/model"
	"jobflow/internal/repository"
	"jobflow/internal/service"
	"log/slog"
	"net/http"
)

func createJob(jobService *service.JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddJobRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid json")
			return
		}

		if err := jobService.Create(r.Context(), req.ID, req.Type, req.Payload); err != nil {
			switch {
			case errors.Is(err, repository.ErrAlreadyExists):
				httpx.WriteError(w, http.StatusConflict, "job already exists")
			case errors.Is(err, model.ErrInvalidJob):
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
			default:
				slog.Error("createJob()", "err", err)
				httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func getJob(jobService *service.JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		job, err := jobService.GetByID(r.Context(), id)

		if err != nil {
			switch {
			case errors.Is(err, repository.ErrNotFound):
				httpx.WriteError(w, http.StatusNotFound, "job not found")
			default:
				slog.Error("getJob()", "err", err)
				httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		httpx.WriteJSON(w, http.StatusOK, toJobResponse(job))
	}
}
