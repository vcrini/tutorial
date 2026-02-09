package iceberg

import (
	"context"
	"fmt"

	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/glue"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

type GlueCatalog struct {
	Catalog   catalog.Catalog
	Warehouse string
}

func InitializeGlueCatalog(ctx context.Context, region, warehouse string) (GlueCatalog, error) {
	if warehouse == "" {
		return GlueCatalog{}, fmt.Errorf("warehouse is required")
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return GlueCatalog{}, err
	}
	cat := glue.NewCatalog(glue.WithAwsConfig(cfg))
	return GlueCatalog{Catalog: cat, Warehouse: warehouse}, nil
}
