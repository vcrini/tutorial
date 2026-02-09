package streams

import (
	"context"
	"fmt"
	"time"

	"importer/internal/model"

	"github.com/google/uuid"
)

type Importer[D any, C any] struct {
	List    func(ctx context.Context) ([]C, error)
	Key     func(key string, reinit bool) (model.Element[string, string], error)
	KeyOf   func(ctx C) string
	SizeOf  func(ctx C) int64
	Load    Loader[C]
	Decode  Decoder[D]
	UUID    UUIDService
	Store   StoreService[D, C]
	Publish PublishService[D, C]
	Archive ArchiveService[D, C]
	Group   GroupConfig
	Reinit  bool
	Logger  Logger
	Retry   RetryPolicy
}

type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

func (i Importer[D, C]) RunOnce(ctx context.Context, enableArchive bool) error {
	if i.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if i.List == nil || i.Key == nil || i.KeyOf == nil || i.SizeOf == nil {
		return fmt.Errorf("missing source functions")
	}
	if i.Load == nil || i.Decode == nil || i.UUID == nil || i.Store == nil || i.Publish == nil {
		return fmt.Errorf("missing services")
	}
	if enableArchive && i.Archive == nil {
		return fmt.Errorf("archive service missing")
	}

	objects, err := i.List(ctx)
	if err != nil {
		return err
	}

	blocks := i.group(ctx, objects)
	for _, block := range blocks {
		if err := Retry(ctx, i.Retry, func() error { return i.Store.Store(ctx, block) }); err != nil {
			return err
		}
		if err := Retry(ctx, i.Retry, func() error { return i.Publish.Publish(ctx, block) }); err != nil {
			return err
		}
		if enableArchive {
			if err := Retry(ctx, i.Retry, func() error { return i.Archive.Archive(ctx, block) }); err != nil {
				return err
			}
		}
	}

	return nil
}

func (i Importer[D, C]) group(ctx context.Context, objects []C) []model.Block[D, C] {
	batchSize := i.Group.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	batchWeight := i.Group.BatchWeightBytes
	if batchWeight <= 0 {
		batchWeight = 5 * 1024 * 1024
	}

	var blocks []model.Block[D, C]
	var current model.Block[D, C]
	var currentWeight int64

	flush := func() {
		if len(current.Elements) == 0 && len(current.Errors) == 0 {
			return
		}
		blocks = append(blocks, current)
		current = model.Block[D, C]{}
		currentWeight = 0
	}

	for _, obj := range objects {
		key := i.KeyOf(obj)
		parsed, err := i.Key(key, i.Reinit)
		if err != nil {
			current.Errors = append(current.Errors, model.NewFileFormatError(obj))
			if len(current.Errors)+len(current.Elements) >= batchSize {
				flush()
			}
			continue
		}

		content, err := i.Load.Load(ctx, obj)
		if err != nil {
			current.Errors = append(current.Errors, model.NewDecodeError(parsed.SocCod, parsed.BoType, parsed.BoCod, parsed.Operation, obj))
			if len(current.Errors)+len(current.Elements) >= batchSize {
				flush()
			}
			continue
		}

		decoded, err := i.Decode.Decode(content)
		if err != nil {
			current.Errors = append(current.Errors, model.NewDecodeError(parsed.SocCod, parsed.BoType, parsed.BoCod, parsed.Operation, obj))
			if len(current.Errors)+len(current.Elements) >= batchSize {
				flush()
			}
			continue
		}

		el := model.Element[D, C]{
			Data:      decoded,
			SocCod:    parsed.SocCod,
			BoType:    parsed.BoType,
			BoCod:     parsed.BoCod,
			Operation: parsed.Operation,
			Size:      i.SizeOf(obj),
			Ctx:       obj,
		}

		if current.UUID == uuid.Nil {
			current.UUID = uuid.MustParse(i.UUID.New())
			current.Created = time.Now().UTC()
		}

		current.Elements = append(current.Elements, el)
		currentWeight += el.Size
		if len(current.Elements)+len(current.Errors) >= batchSize || currentWeight >= batchWeight {
			flush()
		}
	}

	flush()

	return blocks
}
