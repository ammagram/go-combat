package handler

import (
	"jobflow/internal/service"
	"net/http"
)

func Register(mux *http.ServeMux, jobs *service.JobService) {
	jobController := NewJobController(jobs)

	mux.HandleFunc("POST /jobs", jobController.Create)
	mux.HandleFunc("GET /jobs/{id}", jobController.Get)
}
