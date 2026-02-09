package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"importer/internal/model"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSPublisher[D any, C any] struct {
	Client  *sqs.Client
	Queue   string
	AssetID string
}

type sqsBody struct {
	AssetID        string `json:"asset_id"`
	SandboxPackage string `json:"sbx_package_guid"`
}

func (p SQSPublisher[D, C]) Publish(ctx context.Context, block model.Block[D, C]) error {
	if p.Client == nil {
		return fmt.Errorf("sqs client is nil")
	}
	payload := sqsBody{
		AssetID:        p.AssetID,
		SandboxPackage: block.UUID.String(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	messageGroup := fmt.Sprintf("%s-IO", p.AssetID)

	_, err = p.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:               &p.Queue,
		MessageBody:            stringPtr(string(body)),
		MessageDeduplicationId: stringPtr(block.UUID.String()),
		MessageGroupId:         &messageGroup,
	})
	return err
}

func stringPtr(v string) *string { return &v }
