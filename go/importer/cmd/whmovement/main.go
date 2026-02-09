package main

import (
	"context"
	"fmt"
	"path"
	"time"

	"importer/internal/aws"
	"importer/internal/config"
	"importer/internal/iceberg"
	coremodel "importer/internal/model"
	"importer/internal/observability"
	"importer/internal/s3"
	"importer/internal/streams"
	"importer/internal/utils"
	"importer/internal/whmovement"
	whmodel "importer/internal/whmovement/model"
)

const (
	boName  = "WHMOVEMENT"
	assetID = "warehouse-movement"
	appName = "whmovement-data-importer"
)

func main() {
	logger := observability.NewLogger("whmovement")
	cfg, err := config.Load()
	if err != nil {
		logger.Errorf("config error: %v", err)
		return
	}

	ctx := context.Background()
	clients, err := aws.NewClients(ctx, cfg.AWSRegion)
	if err != nil {
		logger.Errorf("aws clients error: %v", err)
		return
	}

	warehouse := fmt.Sprintf("s3://%s%s%s", cfg.S3BucketName, cfg.DatabaseFolder, cfg.DatabaseName)
	catalog, err := iceberg.InitializeGlueCatalog(ctx, cfg.AWSRegion, warehouse)
	if err != nil {
		logger.Errorf("iceberg catalog error: %v", err)
		return
	}

	loader := s3.Loader{Client: clients.S3, Bucket: cfg.S3BucketName}
	archiver := s3.Archiver[whmodel.TWHMovementSyncDel]{
		Client:      clients.S3,
		Bucket:      cfg.S3BucketName,
		PathValid:   joinPath(cfg.S3ArchiveFolder, cfg.SocCod, boName),
		PathInvalid: joinPath(cfg.S3ArchiveInvalid, cfg.SocCod, boName),
		BoID:        whmovement.BoID,
	}
	publisher := aws.SQSPublisher[whmodel.TWHMovementSyncDel, s3.Object]{
		Client:  clients.SQS,
		Queue:   cfg.SQSQueue,
		AssetID: assetID,
	}

	importer := streams.Importer[whmodel.TWHMovementSyncDel, s3.Object]{
		List: func(ctx context.Context) ([]s3.Object, error) {
			root := cfg.S3SourceFolder
			if cfg.Reinit {
				root = cfg.S3ArchiveFolder
			}
			prefix := joinPath(root, cfg.SocCod, boName)
			return s3.ListXML(ctx, clients.S3, cfg.S3BucketName, prefix)
		},
		Key: func(key string, reinit bool) (coremodel.Element[string, string], error) {
			if reinit {
				return utils.KeySplitReinit(key)
			}
			return utils.KeySplit(key)
		},
		KeyOf:  func(obj s3.Object) string { return obj.Key },
		SizeOf: func(obj s3.Object) int64 { return obj.Size },
		Load:   loader,
		Decode: whmovement.NewDecoder(),
		UUID:   utils.UUIDService{},
		Store: iceberg.StoreService[whmodel.TWHMovementSyncDel, s3.Object]{
			Catalog:     catalog,
			Namespace:   cfg.DatabaseName,
			SetupTables: whmovement.SetupTables,
			ToRecords: func(data whmodel.TWHMovementSyncDel, block coremodel.Block[whmodel.TWHMovementSyncDel, s3.Object], deleted bool) map[string][]iceberg.Record {
				return whmovement.ToRecords(data, block, deleted)
			},
			BoID:        whmovement.BoID,
			BoPartition: whmovement.BoPartitionKey,
			BoOrdering:  whmovement.BoOrderingID,
		},
		Publish: publisher,
		Archive: archiver,
		Group: streams.GroupConfig{
			BatchSize:        cfg.BatchSize,
			BatchWeightBytes: cfg.BatchWeightInBytes,
			FlushInterval:    10 * time.Second,
		},
		Reinit: cfg.Reinit,
		Logger: logger,
		Retry:  streams.DefaultRetryPolicy(),
	}

	run(ctx, importer, cfg, logger)
}

func joinPath(root, soc, bo string) string {
	if root == "" {
		return path.Join(soc, bo) + "/"
	}
	return path.Join(root, soc, bo) + "/"
}

func run(ctx context.Context, importer streams.Importer[whmodel.TWHMovementSyncDel, s3.Object], cfg config.Config, logger observability.Logger) {
	if cfg.Reinit {
		logger.Infof("reinit enabled, delaying %s", cfg.ReinitDelay)
		time.Sleep(cfg.ReinitDelay)
		for {
			err := importer.RunOnce(ctx, false)
			if err == nil {
				return
			}
			logger.Errorf("reinit run error: %v", err)
			exponentialSleep(3*time.Second, time.Minute)
		}
	}

	for {
		err := importer.RunOnce(ctx, true)
		if err != nil {
			logger.Errorf("run error: %v", err)
			exponentialSleep(3*time.Second, time.Minute)
			continue
		}
		logger.Infof("run completed, sleeping %s", cfg.Interval)
		time.Sleep(cfg.Interval)
	}
}

var retryStep = 0

func exponentialSleep(min, max time.Duration) {
	backoff := min * time.Duration(1<<retryStep)
	if backoff > max {
		backoff = max
	} else {
		retryStep++
	}
	time.Sleep(backoff)
}
