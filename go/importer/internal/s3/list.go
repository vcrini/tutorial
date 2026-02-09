package s3

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func ListXML(ctx context.Context, client *s3.Client, bucket, prefix string) ([]Object, error) {
	var out []Object
	p := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || !strings.HasSuffix(*obj.Key, ".xml") {
				continue
			}
			size := int64(0)
			if obj.Size != nil {
				size = *obj.Size
			}
			out = append(out, Object{Key: *obj.Key, Size: size})
		}
	}

	return out, nil
}
