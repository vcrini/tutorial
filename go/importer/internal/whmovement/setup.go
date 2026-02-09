package whmovement

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/iceberg-go/catalog"

	"importer/internal/iceberg"
	"importer/internal/whmovement/tables"
)

func SetupTables(ctx context.Context, cat iceberg.GlueCatalog, namespace string, reinit bool) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	exists, err := cat.Catalog.CheckNamespaceExists(ctx, catalog.ToIdentifier(namespace))
	if err != nil {
		return err
	}
	if !exists {
		if err := cat.Catalog.CreateNamespace(ctx, catalog.ToIdentifier(namespace), nil); err != nil {
			return err
		}
	}

	allSchemas := []iceberg.TableSchema{
		tables.MgtTable,
		tables.MgtAssocTable,
		tables.MgtLniTable,
		tables.MgtSpeTable,
		tables.MgtMgrTable,
		tables.MgtNotTable,
		tables.MgtMgrAssocTable,
		tables.MgtMgrTglTable,
		tables.MgtMgrAgeTable,
		tables.MgtMgrMgcTable,
		tables.MgtMgrMgpTable,
		tables.MgtMgrGrtFprTable,
		tables.MgtMgrNotTable,
		tables.MgtMgrBarTable,
		tables.MgtMgrCdsTable,
		tables.MgtMgrAgePrvTable,
		tables.MgtMgrCdsCprTglTable,
		tables.MgtMgrAssocBarTable,
		tables.MgtMgrAssocTglTable,
		tables.MgtMgrAssocValTable,
	}

	for _, schema := range allSchemas {
		ident := catalog.ToIdentifier(namespace, schema.Name)
		if reinit {
			_ = cat.Catalog.DropTable(ctx, ident)
		}
		exists, err := cat.Catalog.CheckTableExists(ctx, ident)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		iceSchema, err := iceberg.BuildSchema(schema)
		if err != nil {
			return err
		}
		location := strings.TrimRight(cat.Warehouse, "/") + "/" + schema.Name
		_, err = cat.Catalog.CreateTable(ctx, ident, iceSchema, catalog.WithLocation(location))
		if err != nil {
			return err
		}
	}

	return nil
}
