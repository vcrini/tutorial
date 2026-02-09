package model

import (
	"time"

	"github.com/google/uuid"
)

type Block[D any, C any] struct {
	UUID     uuid.UUID
	Created  time.Time
	Elements []Element[D, C]
	Errors   []Error[C]
}
