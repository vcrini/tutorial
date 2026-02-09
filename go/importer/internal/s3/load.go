package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Loader struct {
	Client *s3.Client
	Bucket string
}

func (l Loader) Load(ctx context.Context, obj Object) (string, error) {
	out, err := l.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &l.Bucket,
		Key:    &obj.Key,
	})
	if err != nil {
		return "", err
	}
	defer out.Body.Close()

	b, err := io.ReadAll(out.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
