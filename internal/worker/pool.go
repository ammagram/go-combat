package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"jobflow/internal/model"
)

var ErrInvalidWorkers = errors.New("worker pool: workers must be positive")

type ProcessFunc func(ctx context.Context, job model.JobModel) error

type WorkerPool struct {
	workers   int
	processed atomic.Int64
	failed    atomic.Int64
}

func NewWorkerPool(n int) *WorkerPool {
	return &WorkerPool{workers: n}
}

func (p *WorkerPool) Run(
	ctx context.Context,
	jobs <-chan model.JobModel,
	process ProcessFunc,
) error {
	if p.workers <= 0 {
		return fmt.Errorf("workers = %d: %w", p.workers, ErrInvalidWorkers)
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		errs     []error
		canceled atomic.Bool
	)

	recordErr := func(err error) {
		mu.Lock()
		defer mu.Unlock()

		errs = append(errs, err)
	}

	for range p.workers {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					canceled.Store(true)
					return

				case job, ok := <-jobs:
					if !ok {
						return
					}

					if ctx.Err() != nil {
						canceled.Store(true)
						return
					}

					if err := process(ctx, job); err != nil {
						p.failed.Add(1)
						recordErr(fmt.Errorf("job %s: %w", job.ID, err))
						continue
					}

					p.processed.Add(1)
				}
			}
		})
	}

	wg.Wait()

	if canceled.Load() {
		errs = append(errs, ctx.Err())
	}

	return errors.Join(errs...)
}

func (p *WorkerPool) Processed() int64 { return p.processed.Load() }
func (p *WorkerPool) Failed() int64    { return p.failed.Load() }
