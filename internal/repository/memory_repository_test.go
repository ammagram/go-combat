package repository

import (
	"context"
	"errors"
	"jobflow/internal/model"
	"reflect"
	"testing"
	"time"
)

func newRepoWith(t *testing.T, jobs ...model.JobModel) *MemoryJobRepository {
	t.Helper()
	repo := NewMemoryJobRepository()
	for _, job := range jobs {
		if err := repo.Add(context.Background(), job); err != nil {
			t.Fatalf("Add(%q) error: %v", job.ID, err)
		}
	}
	return repo
}

func newTestJob(id, jobType string, createdAt time.Time) model.JobModel {
	return model.JobModel{
		ID:        id,
		Type:      jobType,
		Status:    model.Created,
		Payload:   []byte{1, 2, 3},
		CreatedAt: createdAt,
	}
}

func TestMemoryRepository_Add(t *testing.T) {
	baseTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	jobA := newTestJob("a", "test", baseTime)
	jobB := newTestJob("b", "test", baseTime.Add(time.Hour))
	jobAOther := newTestJob("a", "other", baseTime.Add(2*time.Hour))

	tests := []struct {
		name       string
		seed       []model.JobModel
		add        model.JobModel
		wantErr    error
		wantStored model.JobModel
	}{
		{name: "add to empty repo", add: jobA, wantStored: jobA},
		{name: "add job with another id", seed: []model.JobModel{jobA}, add: jobB, wantStored: jobB},
		{name: "duplicate id keep original", seed: []model.JobModel{jobA}, add: jobAOther, wantErr: ErrAlreadyExists, wantStored: jobA},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := newRepoWith(t, tt.seed...)
			ctx := context.Background()

			// Act
			err := repo.Add(ctx, tt.add)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Add(%q) error = %v, want %v", tt.add.ID, err, tt.wantErr)
			}

			got, getErr := repo.Get(ctx, tt.add.ID)
			if getErr != nil {
				t.Fatalf("Get(%q) after Add: %v", tt.add.ID, getErr)
			}
			if !reflect.DeepEqual(got, tt.wantStored) {
				t.Errorf("MemoryRepository.Add(%q): got %v, want %v", tt.add.ID, got, tt.wantStored)
			}

			for _, s := range tt.seed {
				if s.ID == tt.add.ID {
					continue
				}
				other, err := repo.Get(ctx, s.ID)
				if err != nil || !reflect.DeepEqual(other, s) {
					t.Errorf("seed %q changed: got %v, err %v", s.ID, other, err)
				}
			}
		})
	}
}

func TestMemoryRepository_Get(t *testing.T) {
	baseTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	jobA := newTestJob("a", "test", baseTime)
	jobB := newTestJob("b", "test", baseTime.Add(time.Hour))
	jobC := newTestJob("c", "test", baseTime.Add(2*time.Hour))

	tests := []struct {
		name    string
		seed    []model.JobModel
		id      string
		wantErr error
		wantJob model.JobModel
	}{
		{name: "existing job", seed: []model.JobModel{jobA}, id: "a", wantJob: jobA},
		{name: "missing id", seed: []model.JobModel{jobA, jobB}, id: "c", wantErr: ErrNotFound},
		{name: "empty repo", id: "a", wantErr: ErrNotFound},
		{name: "pick right one", seed: []model.JobModel{jobA, jobB, jobC}, id: "b", wantJob: jobB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := newRepoWith(t, tt.seed...)

			// Act
			job, err := repo.Get(context.Background(), tt.id)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("MemoryRepository.Get(%q) got %v, want %v", tt.id, err, tt.wantErr)
			}
			if !reflect.DeepEqual(job, tt.wantJob) {
				t.Errorf("MemoryRepository.Get(%q) got %v, want %v", tt.id, job, tt.wantJob)
			}
		})
	}
}

func TestMemoryRepository_Delete(t *testing.T) {
	baseTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	jobA := newTestJob("a", "test", baseTime)
	jobB := newTestJob("b", "test", baseTime.Add(time.Hour))

	tests := []struct {
		name    string
		seed    []model.JobModel
		id      string
		wantErr error
	}{
		{name: "existing job", seed: []model.JobModel{jobA, jobB}, id: jobA.ID},
		{name: "missing id", seed: []model.JobModel{jobB}, id: jobA.ID, wantErr: ErrNotFound},
		{name: "empty repo", id: jobA.ID, wantErr: ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := newRepoWith(t, tt.seed...)
			ctx := context.Background()

			// Act
			err := repo.Delete(ctx, tt.id)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				_, gotErr := repo.Get(ctx, tt.id)
				if !errors.Is(gotErr, ErrNotFound) {
					t.Errorf("Get(%q) after Delete: err = %v, want ErrNotFound", tt.id, gotErr)
				}
			}
			for _, s := range tt.seed {
				if s.ID == tt.id {
					continue
				}
				other, getErr := repo.Get(ctx, s.ID)
				if getErr != nil || !reflect.DeepEqual(other, s) {
					t.Errorf("seed %q changed: got %v, err %v", s.ID, other, getErr)
				}
			}
		})
	}
}

func TestMemoryRepository_List(t *testing.T) {
	baseTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	jobA := newTestJob("a", "test", baseTime)
	jobB := newTestJob("b", "test", baseTime.Add(time.Hour))
	jobC := newTestJob("c", "test", baseTime.Add(2*time.Hour))
	jobD := newTestJob("d", "test", baseTime.Add(3*time.Hour))
	jobE := newTestJob("e", "test", baseTime.Add(4*time.Hour))

	tests := []struct {
		name string
		seed []model.JobModel
		want []model.JobModel
	}{
		{
			name: "empty repo",
			seed: []model.JobModel{},
			want: []model.JobModel{},
		},
		{
			name: "single job",
			seed: []model.JobModel{jobA},
			want: []model.JobModel{jobA},
		},
		{
			name: "sorted by CreatedAt",
			seed: []model.JobModel{jobD, jobC, jobE, jobB, jobA},
			want: []model.JobModel{jobA, jobB, jobC, jobD, jobE},
		},
		{
			name: "equal CreatedAt sorted by ID",
			seed: []model.JobModel{
				newTestJob("c", "test", baseTime),
				newTestJob("b", "test", baseTime),
				newTestJob("e", "test", baseTime),
				newTestJob("a", "test", baseTime),
				newTestJob("d", "test", baseTime),
			},
			want: []model.JobModel{
				newTestJob("a", "test", baseTime),
				newTestJob("b", "test", baseTime),
				newTestJob("c", "test", baseTime),
				newTestJob("d", "test", baseTime),
				newTestJob("e", "test", baseTime),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := newRepoWith(t, tt.seed...)
			ctx := context.Background()

			// Act
			list, err := repo.List(ctx)

			// Assert
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(list) != len(tt.want) {
				t.Fatalf("List() returned %d jobs, want %d", len(list), len(tt.want))
			}
			for i, job := range list {
				if !reflect.DeepEqual(job, tt.want[i]) {
					t.Errorf("List()[%d] = %v, want %v", i, job, tt.want[i])
				}
			}
		})
	}
}

func TestMemoryRepository_FindByType(t *testing.T) {
	baseTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		jobType string
		seed    []model.JobModel
		want    []model.JobModel
	}{
		{
			name:    "empty repo",
			jobType: "a",
			seed:    []model.JobModel{},
			want:    []model.JobModel{},
		},
		{
			name:    "no match",
			jobType: "b",
			seed: []model.JobModel{
				newTestJob("job1", "c", baseTime),
				newTestJob("job2", "d", baseTime),
			},
			want: []model.JobModel{},
		},
		{
			name:    "single match",
			jobType: "c",
			seed: []model.JobModel{
				newTestJob("job1", "c", baseTime),
				newTestJob("job2", "d", baseTime),
				newTestJob("job3", "d", baseTime),
			},
			want: []model.JobModel{
				newTestJob("job1", "c", baseTime),
			},
		},
		{
			name:    "only requested type",
			jobType: "d",
			seed: []model.JobModel{
				newTestJob("job1", "c", baseTime),
				newTestJob("job2", "d", baseTime),
				newTestJob("job3", "d", baseTime),
				newTestJob("job4", "a", baseTime),
				newTestJob("job5", "b", baseTime),
			},
			want: []model.JobModel{
				newTestJob("job2", "d", baseTime),
				newTestJob("job3", "d", baseTime),
			},
		},
		{
			name:    "sorted by CreatedAt",
			jobType: "d",
			seed: []model.JobModel{
				newTestJob("job1", "c", baseTime),
				newTestJob("job2", "d", baseTime.Add(time.Hour*2)),
				newTestJob("job3", "d", baseTime.Add(time.Hour)),
				newTestJob("job4", "a", baseTime),
				newTestJob("job5", "d", baseTime.Add(time.Hour*3)),
				newTestJob("job6", "d", baseTime),
				newTestJob("job7", "b", baseTime),
			},
			want: []model.JobModel{
				newTestJob("job6", "d", baseTime),
				newTestJob("job3", "d", baseTime.Add(time.Hour)),
				newTestJob("job2", "d", baseTime.Add(time.Hour*2)),
				newTestJob("job5", "d", baseTime.Add(time.Hour*3)),
			},
		},
		{
			name:    "equal CreatedAt sorted by ID",
			jobType: "d",
			seed: []model.JobModel{
				newTestJob("job1", "c", baseTime),
				newTestJob("job6", "d", baseTime),
				newTestJob("job3", "d", baseTime),
				newTestJob("job2", "d", baseTime),
				newTestJob("job4", "a", baseTime),
				newTestJob("job5", "d", baseTime),
				newTestJob("job7", "b", baseTime),
			},
			want: []model.JobModel{
				newTestJob("job2", "d", baseTime),
				newTestJob("job3", "d", baseTime),
				newTestJob("job5", "d", baseTime),
				newTestJob("job6", "d", baseTime),
			},
		},
		{
			name:    "exact type match",
			jobType: "email",
			seed: []model.JobModel{
				newTestJob("job1", "email", baseTime),
				newTestJob("job2", "EMAIL", baseTime),
				newTestJob("job3", "email2", baseTime),
			},
			want: []model.JobModel{
				newTestJob("job1", "email", baseTime),
			},
		},
		{
			name:    "empty jobType",
			jobType: "",
			seed: []model.JobModel{
				newTestJob("job1", "", baseTime),
				newTestJob("job2", "EMAIL", baseTime),
				newTestJob("job3", "email2", baseTime),
				newTestJob("job4", "  ", baseTime),
			},
			want: []model.JobModel{
				newTestJob("job1", "", baseTime),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := newRepoWith(t, tt.seed...)

			// Act
			list, err := repo.FindByType(context.Background(), tt.jobType)

			// Assert
			if err != nil {
				t.Fatalf("FindByType() error = %v", err)
			}
			if len(list) != len(tt.want) {
				t.Fatalf("FindByType() returned %d jobs, want %d", len(list), len(tt.want))
			}
			for i, job := range list {
				if !reflect.DeepEqual(job, tt.want[i]) {
					t.Errorf("FindByType()[%d] = %v, want %v", i, job, tt.want[i])
				}
			}
		})
	}
}
