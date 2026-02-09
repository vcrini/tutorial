package s3

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"importer/internal/model"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Archiver[D any] struct {
	Client      *s3.Client
	Bucket      string
	PathValid   string
	PathInvalid string
	BoID        func(D) string
	Concurrency int
}

func (a Archiver[D]) Archive(ctx context.Context, block model.Block[D, Object]) error {
	concurrency := a.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(block.Elements)+len(block.Errors))
	sem := make(chan struct{}, concurrency)

	for _, element := range block.Elements {
		element := element
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := a.archiveElement(ctx, element); err != nil {
				errCh <- err
			}
		}()
	}

	for _, errItem := range block.Errors {
		errItem := errItem
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := a.archiveError(ctx, errItem); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (a Archiver[D]) archiveElement(ctx context.Context, element model.Element[D, Object]) error {
	srcKey := element.Ctx.Key
	if srcKey == "" {
		return fmt.Errorf("missing source key")
	}

	boID := a.BoID(element.Data)
	validKey := path.Join(a.PathValid, boID+".xml")

	switch element.Operation {
	case model.OperationSync:
		if err := a.copy(ctx, srcKey, validKey); err != nil {
			return err
		}
	case model.OperationDelete:
		if err := a.delete(ctx, validKey); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown operation")
	}

	return a.delete(ctx, srcKey)
}

func (a Archiver[D]) archiveError(ctx context.Context, errItem model.Error[Object]) error {
	srcKey := errItem.Ctx.Key
	if srcKey == "" {
		return fmt.Errorf("missing source key")
	}

	baseName := path.Base(srcKey)
	invalidKey := path.Join(a.PathInvalid, baseName)
	if err := a.copy(ctx, srcKey, invalidKey); err != nil {
		return err
	}

	return a.delete(ctx, srcKey)
}

func (a Archiver[D]) copy(ctx context.Context, srcKey, dstKey string) error {
	copySource := fmt.Sprintf("%s/%s", a.Bucket, strings.TrimPrefix(srcKey, "/"))
	_, err := a.Client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &a.Bucket,
		CopySource: &copySource,
		Key:        &dstKey,
	})
	return err
}

func (a Archiver[D]) delete(ctx context.Context, key string) error {
	_, err := a.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &a.Bucket,
		Key:    &key,
	})
	return err
}
