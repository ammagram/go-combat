package repository

import (
	"context"
	"fmt"
	"jobflow/internal/model"
	"sync"
)

type MemoryJobRepository struct {
	mu   sync.RWMutex
	jobs map[string]model.JobModel
}

func (r *MemoryJobRepository) Add(ctx context.Context, job model.JobModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.jobs[job.ID]; ok {
		return fmt.Errorf("job %q: %w", job.ID, ErrAlreadyExists)
	}

	r.jobs[job.ID] = job

	return nil
}

func (r *MemoryJobRepository) Get(ctx context.Context, id string) (model.JobModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[id]
	if !ok {
		return model.JobModel{}, ErrNotFound
	}

	return job, nil
}

func NewMemoryJobRepository() *MemoryJobRepository {
	return &MemoryJobRepository{
		jobs: make(map[string]model.JobModel),
	}
}
