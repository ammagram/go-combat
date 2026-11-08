package service

import (
	"context"
	"fmt"
	"jobflow/internal/model"
)

type JobRepository interface {
	Add(ctx context.Context, job model.JobModel) error
	Get(ctx context.Context, id string) (model.JobModel, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.JobModel, error)
}

type JobService struct {
	repo JobRepository
}

func (s *JobService) Create(ctx context.Context, id string, jobType string, payload []byte) error {
	newJob := model.NewJob(id, jobType, payload)

	if err := newJob.Validate(); err != nil {
		return err
	}

	if err := s.repo.Add(ctx, newJob); err != nil {
		return err
	}

	return nil
}

func (s *JobService) GetByID(ctx context.Context, id string) (model.JobModel, error) {
	job, err := s.repo.Get(ctx, id)

	if err != nil {
		return model.JobModel{}, fmt.Errorf("GetByID() error: %w", err)
	}

	return job, nil
}

func NewJobService(repo JobRepository) *JobService {
	return &JobService{
		repo: repo,
	}
}
