package service

import (
	"context"
	"errors"
	"fmt"
	"jobflow/internal/model"
	"log/slog"
	"sync"
)

var (
	ErrQueueFull     = errors.New("job queue is full")
	ErrServiceClosed = errors.New("job service is closed")
)

const defaultQueueSize = 100

type JobRepository interface {
	Add(ctx context.Context, job model.JobModel) error
	Get(ctx context.Context, id string) (model.JobModel, error)
	Update(ctx context.Context, job model.JobModel) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.JobModel, error)
	FindByType(ctx context.Context, jobType string) ([]model.JobModel, error)
}

type WorkFunc func(ctx context.Context, job model.JobModel) error

type Option func(*JobService)

func WithWork(w WorkFunc) Option {
	return func(s *JobService) {
		s.work = w
	}
}

func WithQueueSize(n int) Option {
	return func(s *JobService) {
		if n >= 1 {
			s.queue = make(chan model.JobModel, n)
		}
	}
}

type JobService struct {
	repo   JobRepository
	work   WorkFunc
	mu     sync.RWMutex
	closed bool
	queue  chan model.JobModel
}

type CreateJobInput struct {
	ID      string
	Type    string
	Payload []byte
}

func (s *JobService) Create(ctx context.Context, in CreateJobInput) error {
	newJob := model.NewJob(in.ID, in.Type, in.Payload)

	if err := newJob.Validate(); err != nil {
		return err
	}

	if err := newJob.Enqueue(); err != nil {
		return fmt.Errorf("create %q: %w", in.ID, err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return ErrServiceClosed
	}

	if err := s.repo.Add(ctx, newJob); err != nil {
		return err
	}

	select {
	case s.queue <- newJob:
		return nil
	default:
		s.rollback(ctx, newJob.ID)
		return ErrQueueFull
	}
}

func (s *JobService) rollback(ctx context.Context, id string) {
	if err := s.repo.Delete(context.WithoutCancel(ctx), id); err != nil {
		slog.Error("JobService.Create: rollback", "id", id, "err", err)
	}
}

func (s *JobService) GetByID(ctx context.Context, ID string) (model.JobModel, error) {
	job, err := s.repo.Get(ctx, ID)

	if err != nil {
		return model.JobModel{}, fmt.Errorf("GetByID() error: %w", err)
	}

	return job, nil
}

func (s *JobService) Delete(ctx context.Context, ID string) error {
	if err := s.repo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("JobService.Delete() error = %w", err)
	}

	return nil
}

func (s *JobService) Process(ctx context.Context, job model.JobModel) error {
	current, err := s.repo.Get(ctx, job.ID)
	if err != nil {
		return fmt.Errorf("process %q: get: %w", job.ID, err)
	}

	if err := current.Start(); err != nil {
		return fmt.Errorf("process %q: start: %w", job.ID, err)
	}

	if err := s.repo.Update(ctx, current); err != nil {
		return fmt.Errorf("process %q: save running: %w", job.ID, err)
	}

	workErr := s.work(ctx, current)

	if workErr != nil {
		err = current.Fail(workErr)
	} else {
		err = current.Complete()
	}

	if err != nil {
		return fmt.Errorf("process %q: finish: %w", job.ID, err)
	}

	saveErr := s.repo.Update(context.WithoutCancel(ctx), current)
	if saveErr != nil {
		slog.Error("JobService.Process: save result", "id", job.ID, "err", saveErr)
		saveErr = fmt.Errorf("process %q: save result: %w", job.ID, saveErr)
	}

	if workErr != nil {
		workErr = fmt.Errorf("process %q: work: %w", job.ID, workErr)
	}

	return errors.Join(workErr, saveErr)
}

func (s *JobService) Queue() <-chan model.JobModel {
	return s.queue
}

func (s *JobService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	close(s.queue)
}

func NewJobService(repo JobRepository, opts ...Option) *JobService {
	s := &JobService{
		repo:  repo,
		work:  func(context.Context, model.JobModel) error { return nil },
		queue: make(chan model.JobModel, defaultQueueSize),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}
