package model

import (
	"bytes"
	"errors"
	"math/bits"
	"testing"
	"time"
)

func TestJobModel_transitionTo(t *testing.T) {
	tests := []struct {
		name    string
		from    JobStatus
		to      JobStatus
		wantErr error
	}{
		{name: "created -> waiting", from: Created, to: Waiting, wantErr: nil},
		{name: "created -> running", from: Created, to: Running, wantErr: ErrInvalidTransition},
		{name: "created -> done", from: Created, to: Done, wantErr: ErrInvalidTransition},
		{name: "created -> created", from: Created, to: Created, wantErr: ErrInvalidTransition},

		{name: "waiting -> created", from: Waiting, to: Created, wantErr: ErrInvalidTransition},
		{name: "waiting -> running", from: Waiting, to: Running, wantErr: nil},
		{name: "waiting -> done", from: Waiting, to: Done, wantErr: ErrInvalidTransition},
		{name: "waiting -> waiting", from: Waiting, to: Waiting, wantErr: ErrInvalidTransition},

		{name: "running -> created", from: Running, to: Created, wantErr: ErrInvalidTransition},
		{name: "running -> waiting", from: Running, to: Waiting, wantErr: nil},
		{name: "running -> done", from: Running, to: Done, wantErr: nil},
		{name: "running -> running", from: Running, to: Running, wantErr: ErrInvalidTransition},

		{name: "done -> created", from: Done, to: Created, wantErr: ErrInvalidTransition},
		{name: "done -> waiting", from: Done, to: Waiting, wantErr: ErrInvalidTransition},
		{name: "done -> running", from: Done, to: Running, wantErr: ErrInvalidTransition},
		{name: "done -> done", from: Done, to: Done, wantErr: ErrInvalidTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			job := JobModel{Status: tt.from}
			var wantStatus JobStatus
			if tt.wantErr == nil {
				wantStatus = tt.to
			} else {
				wantStatus = tt.from
			}

			// Act
			err := job.transitionTo(tt.to)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("transitionTo() got %v, want %v", err, tt.wantErr)
			}
			if job.Status != wantStatus {
				t.Errorf("transitionTo() got %v, want %v", job.Status, wantStatus)
			}
		})
	}
}

func TestTransitions_GrayCode(t *testing.T) {
	for from, targets := range transitions {
		for _, to := range targets {
			t.Run(from.String()+" -> "+to.String(), func(t *testing.T) {
				diff := uint8(from ^ to)

				if got := bits.OnesCount8(diff); got != 1 {
					t.Errorf("%s (%02b) -> %s (%02b): differ in %d bits, want 1",
						from, uint8(from), to, uint8(to), got)
				}
			})
		}
	}
}

func TestJobModel_Start(t *testing.T) {
	tests := []struct {
		name       string
		from       JobStatus
		wantErr    error
		wantStatus JobStatus
	}{
		{name: "from: Created", from: Created, wantErr: ErrInvalidTransition, wantStatus: Created},
		{name: "from: Waiting", from: Waiting, wantErr: nil, wantStatus: Running},
		{name: "from: Running", from: Running, wantErr: ErrInvalidTransition, wantStatus: Running},
		{name: "from: Done", from: Done, wantErr: ErrInvalidTransition, wantStatus: Done},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			job := JobModel{Status: tt.from}

			// Act
			err := job.Start()

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if job.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", job.Status, tt.wantStatus)
			}
		})
	}
}

func TestJobModel_Enqueue(t *testing.T) {
	tests := []struct {
		name       string
		from       JobStatus
		wantErr    error
		wantStatus JobStatus
	}{
		{name: "from: Created", from: Created, wantErr: nil, wantStatus: Waiting},
		{name: "from: Running", from: Running, wantErr: nil, wantStatus: Waiting},
		{name: "from: Waiting", from: Waiting, wantErr: ErrInvalidTransition, wantStatus: Waiting},
		{name: "from: Done", from: Done, wantErr: ErrInvalidTransition, wantStatus: Done},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			job := JobModel{Status: tt.from}

			// Act
			err := job.Enqueue()

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if job.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", job.Status, tt.wantStatus)
			}
		})
	}
}

func TestJobModel_Complete(t *testing.T) {
	tests := []struct {
		name       string
		from       JobStatus
		wantErr    error
		wantStatus JobStatus
	}{
		{name: "created -> done", from: Created, wantErr: ErrInvalidTransition, wantStatus: Created},
		{name: "waiting -> done", from: Waiting, wantErr: ErrInvalidTransition, wantStatus: Waiting},
		{name: "running -> done", from: Running, wantErr: nil, wantStatus: Done},
		{name: "done -> done", from: Done, wantErr: ErrInvalidTransition, wantStatus: Done},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			job := JobModel{Status: tt.from}

			// Act
			err := job.Complete()

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if job.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", job.Status, tt.wantStatus)
			}
		})
	}
}

func TestJobModel_Fail(t *testing.T) {
	cause := errors.New("error")

	tests := []struct {
		name        string
		from        JobStatus
		cause       error
		wantErr     error
		wantErrText string
		wantStatus  JobStatus
	}{
		{name: "running -> done", from: Running, cause: cause, wantErr: nil, wantErrText: cause.Error(), wantStatus: Done},
		{name: "created -> done", from: Created, cause: cause, wantErr: ErrInvalidTransition, wantErrText: "", wantStatus: Created},
		{name: "waiting -> done", from: Waiting, cause: cause, wantErr: ErrInvalidTransition, wantErrText: "", wantStatus: Waiting},
		{name: "done -> done", from: Done, cause: cause, wantErr: ErrInvalidTransition, wantErrText: "", wantStatus: Done},
		{name: "nil cause", from: Running, cause: nil, wantErr: ErrNilCause, wantErrText: "", wantStatus: Running},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			job := JobModel{Status: tt.from}

			// Act
			err := job.Fail(tt.cause)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if job.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", job.Status, tt.wantStatus)
			}
			if job.Err != tt.wantErrText {
				t.Errorf("Err = %q, want %q", job.Err, tt.wantErrText)
			}
		})
	}
}

func TestJobModel_IsFinished(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{name: "created", status: Created, want: false},
		{name: "waiting ", status: Waiting, want: false},
		{name: "running", status: Running, want: false},
		{name: "done", status: Done, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			job := JobModel{Status: tt.status}

			// Act
			got := job.IsFinished()

			// Assert
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		job     JobModel
		wantErr error
	}{
		{name: "valid", job: JobModel{ID: "1", Type: "email", Status: Created, Payload: []byte{0, 1, 2}}, wantErr: nil},
		{name: "empty ID", job: JobModel{ID: "", Type: "email", Status: Created, Payload: []byte{0, 1, 2}}, wantErr: ErrInvalidJob},
		{name: "empty Type", job: JobModel{ID: "1", Type: "", Status: Created, Payload: []byte{0, 1, 2}}, wantErr: ErrInvalidJob},
		{name: "invalid Status", job: JobModel{ID: "1", Type: "email", Status: 99, Payload: []byte{0, 1, 2}}, wantErr: ErrInvalidJob},
		{name: "empty Payload", job: JobModel{ID: "1", Type: "email", Status: Created, Payload: nil}, wantErr: ErrInvalidJob},
		{name: "zero-length Payload", job: JobModel{ID: "1", Type: "email", Status: Created, Payload: []byte{}}, wantErr: ErrInvalidJob},
		{name: "payload at limit", job: JobModel{ID: "1", Type: "email", Status: Created, Payload: make([]byte, MaxPayloadSize)}, wantErr: nil},
		{name: "payload over limit", job: JobModel{ID: "1", Type: "email", Status: Created, Payload: make([]byte, MaxPayloadSize+1)}, wantErr: ErrInvalidJob},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := tt.job.Validate()

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   string
	}{
		{name: "created", status: Created, want: "created"},
		{name: "waiting", status: Waiting, want: "waiting"},
		{name: "running", status: Running, want: "running"},
		{name: "done", status: Done, want: "done"},
		{name: "unknown", status: 99, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := tt.status.String()

			// Assert
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJobStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{name: "created", status: Created, want: true},
		{name: "waiting", status: Waiting, want: true},
		{name: "running", status: Running, want: true},
		{name: "done", status: Done, want: true},
		{name: "unknown", status: 99, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := tt.status.IsValid()

			// Assert
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewJob(t *testing.T) {
	// Arrange
	id := "1"
	jobType := "email"
	payload := []byte{1, 2, 3}

	// Act
	before := time.Now()
	job := NewJob(id, jobType, payload)
	after := time.Now()

	// Assert
	if job.ID != id {
		t.Errorf("ID = %v, want %v", job.ID, id)
	}
	if job.Type != jobType {
		t.Errorf("Type = %v, want %v", job.Type, jobType)
	}
	if job.Status != Created {
		t.Errorf("Status = %v, want %v", job.Status, Created)
	}
	if !bytes.Equal(job.Payload, payload) {
		t.Errorf("Payload = %v, want %v", job.Payload, payload)
	}
	if job.CreatedAt.Before(before) || job.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want between %v and %v", job.CreatedAt, before, after)
	}
}
