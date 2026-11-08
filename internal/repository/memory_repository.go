package repository

import (
	"context"
	"fmt"
	"jobflow/internal/model"
	"slices"
	"strings"
	"sync"
)

type MemoryJobRepository struct {
	mu   sync.RWMutex
	jobs map[string]model.JobModel
}

func compareJobs(a, b model.JobModel) int {
	if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
		return c
	}

	return strings.Compare(a.ID, b.ID)
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

func (r *MemoryJobRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.jobs[id]; !ok {
		return ErrNotFound
	}

	delete(r.jobs, id)

	return nil
}

func (r *MemoryJobRepository) List(ctx context.Context) ([]model.JobModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]model.JobModel, 0, len(r.jobs))

	for _, job := range r.jobs {
		list = append(list, job)
	}

	slices.SortFunc(list, compareJobs)

	return list, nil
}

func (r *MemoryJobRepository) FindByType(ctx context.Context, jobType string) ([]model.JobModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]model.JobModel, 0)

	for _, job := range r.jobs {
		if job.Type == jobType {
			list = append(list, job)
		}
	}

	slices.SortFunc(list, compareJobs)

	return list, nil
}

func NewMemoryJobRepository() *MemoryJobRepository {
	return &MemoryJobRepository{
		jobs: make(map[string]model.JobModel),
	}
}
