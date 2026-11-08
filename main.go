package main

import (
	"errors"
	"jobflow/internal/handler"
	"jobflow/internal/repository"
	"jobflow/internal/service"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	repo := repository.NewMemoryJobRepository()
	jobService := service.NewJobService(repo)

	handler.Register(mux, jobService)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
