// Package config loads AWS/S3/SQS configuration from env vars.
package config

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func LoadAWS(ctx context.Context) (*s3.Client, *sqs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithSharedConfigProfile(os.Getenv("AWSPROFILE")), // marco.guidi-proclient
	)
	if err != nil {
		return nil, nil, err
	}
	return s3.NewFromConfig(cfg), sqs.NewFromConfig(cfg), nil
}
