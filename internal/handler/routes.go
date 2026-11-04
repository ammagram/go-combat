package handler

import (
	"jobflow/internal/service"
	"net/http"
)

func Register(mux *http.ServeMux, jobs *service.JobService) {
	mux.HandleFunc("POST /jobs", createJob(jobs))
	mux.HandleFunc("GET /jobs/{id}", getJob(jobs))
}
