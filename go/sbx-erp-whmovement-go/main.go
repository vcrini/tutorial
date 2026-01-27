package main

import (
	"context"
	"encoding/xml"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Bucket         string
	SourceFolder   string // datalandingzone
	ArchiveFolder  string // dataarchive
	ArchiveInvalid string // dataarchiveinvaliddocuments
	SQSQueue       string
	SocCod         string // 045
	BatchSize      int    // 100
	Reinit         bool
	// Iceberg: endpoint gRPC/REST
	IcebergEndpoint string
}

type BO struct { // Es. TBMGT da XSD [file:1]
	BoType string `xml:"boType"`
	BoCod  string `xml:"boCod"`
	// ... campi da XSD
}

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cfg := loadConfig(log) // viper/os.Getenv

	if cfg.Reinit {
		reinitTables(cfg, log) // Chiama proxy drop
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return processS3(ctx, cfg, log) })
	if err := g.Wait(); err != nil {
		log.Fatal().Err(err).Msg("Process failed")
	}
}

func processS3(ctx context.Context, cfg Config, log zerolog.Logger) error {
	awsCfg, _ := config.LoadDefaultConfig(ctx)
	s3c := s3.NewFromConfig(awsCfg)
	sqsc := sqs.NewFromConfig(awsCfg)

	// List XML files [web:6]
	pag := s3.NewListObjectsV2Paginator(s3c, &s3.ListObjectsV2Input{
		Bucket: &cfg.Bucket,
		Prefix: aws.String(cfg.SourceFolder),
	})
	batch := make([]*s3.Object, 0, cfg.BatchSize)

	for pag.HasMorePages() {
		out, err := pag.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, obj := range out.Contents {
			if !strings.HasSuffix(*obj.Key, ".xml") {
				continue
			}
			batch = append(batch, obj)

			if len(batch) == cfg.BatchSize {
				if err := processBatch(ctx, batch, cfg, s3c, sqsc, log); err != nil {
					log.Error().Err(err).Msg("Batch failed")
				}
				batch = batch[:0]
			}
		}
	}
	return processBatch(ctx, batch, cfg, s3c, sqsc, log)
}

func processBatch(ctx context.Context, batch []*s3.Object, cfg Config, s3c *s3.Client, sqsc *sqs.Client, log zerolog.Logger) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, obj := range batch {
		obj := obj // Capture
		g.Go(func() error {
			// Get XML
			getOut, err := s3c.GetObject(ctx, &s3.GetObjectInput{Key: obj.Key, Bucket: &cfg.Bucket})
			if err != nil {
				return err
			}
			defer getOut.Body.Close()

			// Parse XML → BO
			var bo BO
			if err := xml.NewDecoder(getOut.Body).Decode(&bo); err != nil {
				return archiveInvalid(ctx, obj, cfg, s3c, log) // Errors → invalid
			}

			// Upsert Iceberg (gRPC call proxy) [web:11]
			if err := upsertIceberg(ctx, bo, cfg.SocCod, cfg.IcebergEndpoint); err != nil {
				log.Error().Err(err).Msg("Iceberg upsert failed")
				return archiveInvalid(ctx, obj, cfg, s3c, log)
			}

			// Publish UUID to SQS FIFO [file:1]
			uuid := generateSortableUUID() // Custom func
			_, err = sqsc.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:               &cfg.SQSQueue,
				MessageBody:            aws.String(`{"uuid":"` + uuid + `"}`),
				MessageGroupId:         aws.String("whmovement"), // FIFO
				MessageDeduplicationId: aws.String(uuid),
			})
			if err != nil {
				return err
			}

			// Archive valid
			return archiveValid(ctx, obj, cfg, s3c, log)
		})
	}
	return g.Wait()
}

// Stub funcs (implementa da XSD structs, gRPC Iceberg, utils)
func archiveValid(ctx context.Context, obj *s3.Object, cfg Config, s3c *s3.Client, log zerolog.Logger) error {
	_, err := s3c.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &cfg.Bucket,
		CopySource: aws.String(cfg.Bucket + "/" + *obj.Key),
		Key:        aws.String(cfg.ArchiveFolder + "/" + *obj.Key),
	})
	if err == nil {
		s3c.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &cfg.Bucket, Key: obj.Key})
	}
	return err
}

func archiveInvalid(ctx context.Context, obj *s3.Object, cfg Config, s3c *s3.Client, log zerolog.Logger) error {
	// Simile, ma ArchiveInvalidFolder
	return nil
}

func upsertIceberg(ctx context.Context, bo BO, socCod, endpoint string) error {
	// gRPC call: conn, client := grpc.DialContext → Upsert RPC [web:11]
	return nil // Implementa proxy Java separato
}

func reinitTables(cfg Config, log zerolog.Logger) {
	// gRPC/REST: drop tables via proxy
}

func generateSortableUUID() string { return "uuid-v7" } // uuid.NewV7
func loadConfig(log zerolog.Logger) Config {
	return Config{ /* os.Getenv */ }
}
