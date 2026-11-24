package main

import (
	"context"
	"errors"
	"jobflow/internal/handler"
	"jobflow/internal/model"
	"jobflow/internal/repository"
	"jobflow/internal/service"
	"jobflow/internal/worker"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	workers         = 4
	shutdownTimeout = 10 * time.Second
)

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo := repository.NewMemoryJobRepository()

	jobService := service.NewJobService(repo, service.WithWork(func(ctx context.Context, j model.JobModel) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}))

	mux := http.NewServeMux()
	handler.Register(mux, jobService)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	pool := worker.NewWorkerPool(workers)
	poolCtx, cancelPool := context.WithCancel(context.Background())
	defer cancelPool()

	poolDone := make(chan error, 1)
	go func() {
		poolDone <- pool.Run(poolCtx, jobService.Queue(), jobService.Process)
	}()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	exitCode := 0

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			exitCode = 1
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown", "err", err)
		exitCode = 1
	}

	jobService.Close()

	select {
	case err := <-poolDone:
		logPoolResult(pool, err)
	case <-shutdownCtx.Done():
		slog.Warn("pool did not drain in time, cancelling workers")
		cancelPool()
		logPoolResult(pool, <-poolDone)
		exitCode = 1
	}

	return exitCode
}

func logPoolResult(pool *worker.WorkerPool, err error) {
	slog.Info("pool stopped",
		"processed", pool.Processed(),
		"failed", pool.Failed(),
		"err", err,
	)
}

func main() {
	os.Exit(run())
}
