package worker_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"jobflow/internal/model"
	"jobflow/internal/worker"
)

func makeJobs(n int) chan model.JobModel {
	jobs := make(chan model.JobModel, n)
	for i := 0; i < n; i++ {
		jobs <- model.JobModel{ID: fmt.Sprintf("job-%d", i)}
	}
	close(jobs)

	return jobs
}

func TestRun_InvalidWorkers(t *testing.T) {
	pool := worker.NewWorkerPool(0)

	err := pool.Run(context.Background(), makeJobs(1), nil)

	if !errors.Is(err, worker.ErrInvalidWorkers) {
		t.Fatalf("got %v, want ErrInvalidWorkers", err)
	}
}

func TestRun_ProcessesAllJobs(t *testing.T) {
	var count atomic.Int64
	pool := worker.NewWorkerPool(4)

	err := pool.Run(context.Background(), makeJobs(50), func(ctx context.Context, job model.JobModel) error {
		count.Add(1)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count.Load() != 50 {
		t.Errorf("processed %d jobs, want 50", count.Load())
	}
	if pool.Processed() != 50 {
		t.Errorf("Processed() = %d, want 50", pool.Processed())
	}
}

func TestRun_LimitsConcurrency(t *testing.T) {
	const workers = 3

	var (
		mu      sync.Mutex
		current int
		max     int
	)

	pool := worker.NewWorkerPool(workers)

	err := pool.Run(context.Background(), makeJobs(20), func(ctx context.Context, job model.JobModel) error {
		mu.Lock()
		current++
		if current > max {
			max = current
		}
		mu.Unlock()

		time.Sleep(5 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()

		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if max > workers {
		t.Errorf("max concurrent = %d, want <= %d", max, workers)
	}
}

func TestRun_ReturnsProcessErrors(t *testing.T) {
	errBoom := errors.New("boom")
	pool := worker.NewWorkerPool(2)

	err := pool.Run(context.Background(), makeJobs(10), func(ctx context.Context, job model.JobModel) error {
		return errBoom
	})

	if !errors.Is(err, errBoom) {
		t.Fatalf("got %v, want errBoom", err)
	}
	if pool.Failed() != 10 {
		t.Errorf("Failed() = %d, want 10", pool.Failed())
	}
}

func TestRun_CancelBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var count atomic.Int64
	pool := worker.NewWorkerPool(2)

	err := pool.Run(ctx, makeJobs(10), func(ctx context.Context, job model.JobModel) error {
		count.Add(1)
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if count.Load() != 0 {
		t.Errorf("processed %d jobs after cancel, want 0", count.Load())
	}
}

func TestRun_CancelWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	jobs := make(chan model.JobModel) // открыт, задач нет
	pool := worker.NewWorkerPool(2)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := pool.Run(ctx, jobs, func(ctx context.Context, job model.JobModel) error {
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRun_CancelDuringProcessing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pool := worker.NewWorkerPool(2)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := pool.Run(ctx, makeJobs(10), func(ctx context.Context, job model.JobModel) error {
		<-ctx.Done()
		return ctx.Err()
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestRun_NoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	pool := worker.NewWorkerPool(8)
	err := pool.Run(context.Background(), makeJobs(100), func(ctx context.Context, job model.JobModel) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines: before=%d, after=%d", before, after)
	}
}
