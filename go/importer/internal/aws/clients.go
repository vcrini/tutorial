package aws

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Clients struct {
	S3  *s3.Client
	SQS *sqs.Client
}

func NewClients(ctx context.Context, region string) (Clients, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return Clients{}, err
	}

	return Clients{
		S3:  s3.NewFromConfig(cfg),
		SQS: sqs.NewFromConfig(cfg),
	}, nil
}
