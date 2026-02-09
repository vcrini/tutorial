package streams

import (
	"context"
	"time"

	"importer/internal/model"
)

type Loader[C any] interface {
	Load(ctx context.Context, obj C) (string, error)
}

type Decoder[D any] interface {
	Decode(xml string) (D, error)
}

type UUIDService interface {
	New() string
}

type StoreService[D any, C any] interface {
	Store(ctx context.Context, block model.Block[D, C]) error
}

type PublishService[D any, C any] interface {
	Publish(ctx context.Context, block model.Block[D, C]) error
}

type ArchiveService[D any, C any] interface {
	Archive(ctx context.Context, block model.Block[D, C]) error
}

type GroupConfig struct {
	BatchSize        int
	BatchWeightBytes int64
	FlushInterval    time.Duration
}
