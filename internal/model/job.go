package model

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

const MaxPayloadSize = 64 * 1024

type JobStatus uint8

func (s JobStatus) String() string {
	switch s {
	case Created:
		return "created"
	case Running:
		return "running"
	case Waiting:
		return "waiting"
	case Done:
		return "done"
	default:
		return "unknown"
	}
}

func (s JobStatus) IsValid() bool {
	switch s {
	case Created, Running, Waiting, Done:
		return true
	default:
		return false
	}
}

const (
	Created JobStatus = 0b00
	Waiting JobStatus = 0b01
	Running JobStatus = 0b11
	Done    JobStatus = 0b10
)

var transitions = map[JobStatus][]JobStatus{
	Created: {Waiting},
	Waiting: {Running},
	Running: {Waiting, Done},
	Done:    {},
}

var (
	ErrInvalidJob        = errors.New("invalid job")
	ErrInvalidTransition = errors.New("invalid transition")
	ErrNilCause          = errors.New("nil cause")
)

type JobModel struct {
	ID        string
	Type      string
	Payload   []byte
	Status    JobStatus
	Err       string
	CreatedAt time.Time
}

func (j *JobModel) transitionTo(next JobStatus) error {
	if slices.Contains(transitions[j.Status], next) {
		j.Status = next
		return nil
	}
	return fmt.Errorf("%s -> %s: %w", j.Status, next, ErrInvalidTransition)
}

func (j *JobModel) Start() error {
	return j.transitionTo(Running)
}

func (j *JobModel) Complete() error {
	return j.transitionTo(Done)
}

func (j *JobModel) Enqueue() error {
	return j.transitionTo(Waiting)
}

func (j *JobModel) Fail(cause error) error {
	if cause == nil {
		return fmt.Errorf("fail: %w", ErrNilCause)
	}

	if err := j.transitionTo(Done); err != nil {
		return err
	}
	j.Err = cause.Error()
	return nil
}

func (j JobModel) IsFinished() bool {
	return j.Status == Done
}

func (j JobModel) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job id: %w", ErrInvalidJob)
	}

	if j.Type == "" {
		return fmt.Errorf("job type: %w", ErrInvalidJob)
	}

	if !j.Status.IsValid() {
		return fmt.Errorf("job status: %w", ErrInvalidJob)
	}

	if len(j.Payload) == 0 {
		return fmt.Errorf("job payload: %w", ErrInvalidJob)
	}

	if len(j.Payload) > MaxPayloadSize {
		return fmt.Errorf("job payload: exceeds %d bytes: %w", MaxPayloadSize, ErrInvalidJob)
	}

	return nil
}

func NewJob(id, jobType string, payload []byte) JobModel {
	return JobModel{
		ID:        id,
		Type:      jobType,
		Payload:   payload,
		Status:    Created,
		CreatedAt: time.Now(),
	}
}
