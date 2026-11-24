package service

import (
	"bytes"
	"context"
	"errors"
	"jobflow/internal/model"
	"reflect"
	"testing"
)

type fakeRepo struct {
	JobRepository
	addErr    error
	added     []model.JobModel
	getJob    model.JobModel
	getErr    error
	gotID     string
	deleteErr error
	deletedID string
}

func (f *fakeRepo) Add(ctx context.Context, newJob model.JobModel) error {
	if f.addErr != nil {
		return f.addErr
	}

	f.added = append(f.added, newJob)

	return nil
}

func (f *fakeRepo) Get(ctx context.Context, id string) (model.JobModel, error) {
	f.gotID = id

	if f.getErr != nil {
		return model.JobModel{}, f.getErr
	}

	return f.getJob, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id string) error {
	f.deletedID = id

	if f.deleteErr != nil {
		return f.deleteErr
	}

	return nil
}

var _ JobRepository = (*fakeRepo)(nil)

func TestJobService_Create(t *testing.T) {
	var input = CreateJobInput{ID: "a", Type: "type", Payload: []byte{1, 2, 3}}
	var repoErr = errors.New("repo failed")

	tests := []struct {
		name      string
		input     CreateJobInput
		repoErr   error
		wantErr   error
		wantAdded int
	}{
		{name: "success", wantAdded: 1, input: input},
		{name: "invalid input", wantAdded: 0, wantErr: model.ErrInvalidJob, input: CreateJobInput{ID: "", Type: "", Payload: []byte{}}},
		{name: "repository error", wantAdded: 0, repoErr: repoErr, wantErr: repoErr, input: input},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := &fakeRepo{addErr: tt.repoErr}
			service := NewJobService(repo)

			// Act
			err := service.Create(context.Background(), tt.input)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if len(repo.added) != tt.wantAdded {
				t.Fatalf("Create() len(repo.added) = %d, want %d", len(repo.added), tt.wantAdded)
			}
			if tt.wantAdded == 1 {
				got := repo.added[0]

				if got.ID != tt.input.ID {
					t.Errorf("Create() got.ID = %v, want %v", got.ID, tt.input.ID)
				}
				if got.Type != tt.input.Type {
					t.Errorf("Create() got.Type = %v, want %v", got.Type, tt.input.Type)
				}
				if !bytes.Equal(got.Payload, tt.input.Payload) {
					t.Errorf("Create() got.Payload = %v, want %v", got.Payload, tt.input.Payload)
				}
				if got.Status != model.Waiting {
					t.Errorf("Create() got.Status = %v, want %v", got.Status, model.Waiting)
				}
				if got.CreatedAt.IsZero() {
					t.Errorf("Create() got.CreatedAt is zero")
				}
			}
		})
	}
}

func TestJobService_GetByID(t *testing.T) {
	job := model.JobModel{
		ID:      "a",
		Type:    "email",
		Status:  model.Created,
		Payload: []byte{1, 2, 3},
	}
	errNotFound := errors.New("not found")

	tests := []struct {
		name    string
		id      string
		getJob  model.JobModel
		getErr  error
		wantJob model.JobModel
		wantErr error
	}{
		{name: "success", getJob: job, id: job.ID, wantJob: job},
		{name: "not found", id: job.ID, getErr: errNotFound, wantErr: errNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := &fakeRepo{
				getJob: tt.getJob,
				getErr: tt.getErr,
			}
			service := NewJobService(repo)

			// Act
			job, err := service.GetByID(context.Background(), tt.id)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetByID() error = %v, want %v", err, tt.wantErr)
			}
			if repo.gotID != tt.id {
				t.Errorf("GetByID() id = %v, want %v", repo.gotID, tt.id)
			}
			if !reflect.DeepEqual(job, tt.wantJob) {
				t.Errorf("GetByID() got %v, want %v", job, tt.wantJob)
			}
		})
	}
}

func TestJobService_Delete(t *testing.T) {
	errNotFound := errors.New("not found")

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "success delete", id: "a", wantErr: nil},
		{name: "not found job", id: "a", wantErr: errNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := &fakeRepo{
				deleteErr: tt.wantErr,
			}
			service := NewJobService(repo)

			// Act
			err := service.Delete(context.Background(), tt.id)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if repo.deletedID != tt.id {
				t.Errorf("Delete() deletedID = %v, want %v", repo.deletedID, tt.id)
			}
		})
	}
}
