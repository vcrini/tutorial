package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AWSRegion          string
	SocCod             string
	DatabaseName       string
	DatabaseFolder     string
	S3BucketName       string
	S3SourceFolder     string
	S3ArchiveFolder    string
	S3ArchiveInvalid   string
	BatchSize          int
	BatchWeightInBytes int64
	Interval           time.Duration
	SQSQueue           string
	Reinit             bool
	ReinitDelay        time.Duration
}

func Load() (Config, error) {
	get := func(key string) (string, error) {
		val := strings.TrimSpace(os.Getenv(key))
		if val == "" {
			return "", fmt.Errorf("missing env %s", key)
		}
		return val, nil
	}

	awsRegion, err := get("AWS_REGION")
	if err != nil {
		return Config{}, err
	}
	socCod, err := get("SOC_COD")
	if err != nil {
		return Config{}, err
	}
	databaseName, err := get("DATABASE_NAME")
	if err != nil {
		return Config{}, err
	}
	databaseFolder := strings.TrimSpace(os.Getenv("DATABASE_FOLDER"))
	if databaseFolder == "" {
		databaseFolder = "/data/database/"
	}
	bucket, err := get("S3_BUCKET_NAME")
	if err != nil {
		return Config{}, err
	}
	sourceFolder, err := get("S3_SOURCE_FOLDER")
	if err != nil {
		return Config{}, err
	}
	archiveFolder, err := get("S3_ARCHIVE_FOLDER")
	if err != nil {
		return Config{}, err
	}
	archiveInvalid, err := get("S3_ARCHIVE_INVALID_FOLDER")
	if err != nil {
		return Config{}, err
	}
	sqsQueue, err := get("SQS_QUEUE")
	if err != nil {
		return Config{}, err
	}

	batchSize := 100
	if v := strings.TrimSpace(os.Getenv("BATCH_SIZE")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			batchSize = parsed
		}
	}

	batchWeight := int64(5 * 1024 * 1024)
	if v := strings.TrimSpace(os.Getenv("BATCH_WEIGHT_IN_BYTE")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			batchWeight = parsed
		}
	}
	if v := strings.TrimSpace(os.Getenv("BATCH_WEIGHT")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			batchWeight = parsed
		}
	}

	interval := 5 * time.Minute
	if v := strings.TrimSpace(os.Getenv("INTERVAL_IN_MINUTE")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			interval = time.Duration(parsed) * time.Minute
		}
	}

	reinit := false
	if v := strings.TrimSpace(os.Getenv("REINIT")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			reinit = parsed
		}
	}

	reinitDelay := time.Minute
	if v := strings.TrimSpace(os.Getenv("DELAY_IN_MINUTE")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			reinitDelay = time.Duration(parsed) * time.Minute
		}
	}

	return Config{
		AWSRegion:          awsRegion,
		SocCod:             socCod,
		DatabaseName:       databaseName,
		DatabaseFolder:     databaseFolder,
		S3BucketName:       bucket,
		S3SourceFolder:     sourceFolder,
		S3ArchiveFolder:    archiveFolder,
		S3ArchiveInvalid:   archiveInvalid,
		BatchSize:          batchSize,
		BatchWeightInBytes: batchWeight,
		Interval:           interval,
		SQSQueue:           sqsQueue,
		Reinit:             reinit,
		ReinitDelay:        reinitDelay,
	}, nil
}
