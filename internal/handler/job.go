package handler

import (
	"context"
	"encoding/json"
	"errors"
	"jobflow/internal/httpx"
	"jobflow/internal/model"
	"jobflow/internal/repository"
	"jobflow/internal/service"
	"log/slog"
	"net/http"
)

type jobService interface {
	Create(ctx context.Context, in service.CreateJobInput) error
	GetByID(ctx context.Context, id string) (model.JobModel, error)
}

type JobController struct {
	jobs jobService
}

func (c *JobController) Create(w http.ResponseWriter, r *http.Request) {
	var req AddJobRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	input := service.CreateJobInput{
		ID:      req.ID,
		Type:    req.Type,
		Payload: req.Payload,
	}

	if err := c.jobs.Create(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, repository.ErrAlreadyExists):
			httpx.WriteError(w, http.StatusConflict, "job already exists")
		case errors.Is(err, model.ErrInvalidJob):
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrQueueFull), errors.Is(err, service.ErrServiceClosed):
			httpx.WriteError(w, http.StatusServiceUnavailable, "service unavailable, retry later")
		default:
			slog.Error("JobController.Create()", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (c *JobController) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	job, err := c.jobs.GetByID(r.Context(), id)

	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "job not found")
		default:
			slog.Error("JobControlle.Get()", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toJobResponse(job))
}

func NewJobController(jobs jobService) *JobController {
	return &JobController{
		jobs: jobs,
	}
}
