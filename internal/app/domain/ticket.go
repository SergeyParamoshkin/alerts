package domain

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	Queue     string
	Key       string
	Summary   string
	Type      string
}
