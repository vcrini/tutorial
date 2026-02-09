package iceberg

import (
	"context"
	"fmt"
	"sort"

	"github.com/apache/iceberg-go/catalog"

	"importer/internal/model"
)

type StoreService[D any, C any] struct {
	Catalog     GlueCatalog
	Namespace   string
	SetupTables func(ctx context.Context, cat GlueCatalog, namespace string, reinit bool) error
	ToRecords   func(data D, block model.Block[D, C], deleted bool) map[string][]Record
	BoID        func(D) string
	BoPartition func(D) string
	BoOrdering  func(D) string
}

func (s StoreService[D, C]) Store(ctx context.Context, block model.Block[D, C]) error {
	if s.SetupTables == nil || s.ToRecords == nil {
		return fmt.Errorf("iceberg store not configured")
	}

	if err := s.SetupTables(ctx, s.Catalog, s.Namespace, false); err != nil {
		return err
	}

	if len(block.Elements) == 0 {
		return nil
	}

	latest := map[string]model.Element[D, C]{}
	for _, element := range block.Elements {
		boID := s.BoID(element.Data)
		if existing, ok := latest[boID]; ok {
			if s.BoOrdering(element.Data) > s.BoOrdering(existing.Data) {
				latest[boID] = element
			}
		} else {
			latest[boID] = element
		}
	}

	var keys []string
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, boID := range keys {
		element := latest[boID]
		deleted := element.Operation == model.OperationDelete
		recordsByTable := s.ToRecords(element.Data, block, deleted)

		for tableName, records := range recordsByTable {
			if err := appendRecords(ctx, s.Catalog, s.Namespace, tableName, records); err != nil {
				return err
			}
		}
	}

	return nil
}

func appendRecords(ctx context.Context, cat GlueCatalog, namespace, tableName string, records []Record) error {
	if len(records) == 0 {
		return nil
	}
	ident := catalog.ToIdentifier(namespace, tableName)
	tbl, err := cat.Catalog.LoadTable(ctx, ident, nil)
	if err != nil {
		return err
	}

	iceSchema := tbl.Schema()
	schema := ToTableSchema(iceSchema)
	arrowTbl, err := RecordsToArrow(schema, records)
	if err != nil {
		return err
	}
	defer arrowTbl.Release()

	_, err = tbl.AppendTable(ctx, arrowTbl, int64(len(records)), nil)
	return err
}
