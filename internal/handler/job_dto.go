package handler

import (
	"jobflow/internal/model"
	"time"
)

type AddJobRequest struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

type JobResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func toJobResponse(j model.JobModel) JobResponse {
	return JobResponse{
		ID:        j.ID,
		Type:      j.Type,
		Status:    j.Status.String(),
		CreatedAt: j.CreatedAt,
	}
}
