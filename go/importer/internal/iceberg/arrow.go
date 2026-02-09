package iceberg

import (
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func RecordsToArrow(schema TableSchema, records []Record) (arrow.Table, error) {
	fields := make([]arrow.Field, 0, len(schema.Fields))
	builders := make([]array.Builder, 0, len(schema.Fields))
	alloc := memory.DefaultAllocator

	for _, f := range schema.Fields {
		arrowType, err := toArrowType(f.Type)
		if err != nil {
			return nil, err
		}
		fields = append(fields, arrow.Field{Name: f.Name, Type: arrowType, Nullable: !f.Required})
		builders = append(builders, array.NewBuilder(alloc, arrowType))
	}

	for _, rec := range records {
		for i, f := range schema.Fields {
			val, ok := rec[f.Name]
			if !ok {
				builders[i].AppendNull()
				continue
			}
			appendValue(builders[i], val)
		}
	}

	cols := make([]arrow.Array, 0, len(builders))
	for _, b := range builders {
		cols = append(cols, b.NewArray())
		b.Release()
	}

	arrSchema := arrow.NewSchema(fields, nil)
	rec := array.NewRecord(arrSchema, cols, int64(len(records)))
	for _, col := range cols {
		col.Release()
	}
	defer rec.Release()

	table := array.NewTableFromRecords(arrSchema, []arrow.Record{rec})
	return table, nil
}

func toArrowType(t string) (arrow.DataType, error) {
	switch strings.ToLower(t) {
	case "string":
		return arrow.BinaryTypes.String, nil
	case "bool", "boolean":
		return arrow.FixedWidthTypes.Boolean, nil
	case "int", "int32":
		return arrow.PrimitiveTypes.Int32, nil
	case "int64", "long":
		return arrow.PrimitiveTypes.Int64, nil
	case "float", "double", "decimal":
		return arrow.PrimitiveTypes.Float64, nil
	case "timestamp":
		return arrow.FixedWidthTypes.Timestamp_ms, nil
	default:
		return nil, fmt.Errorf("unsupported arrow type: %s", t)
	}
}

func appendValue(b array.Builder, v any) {
	switch builder := b.(type) {
	case *array.StringBuilder:
		builder.Append(fmt.Sprint(v))
	case *array.BooleanBuilder:
		switch x := v.(type) {
		case bool:
			builder.Append(x)
		default:
			builder.Append(false)
		}
	case *array.Int32Builder:
		switch x := v.(type) {
		case int:
			builder.Append(int32(x))
		case int32:
			builder.Append(x)
		case int64:
			builder.Append(int32(x))
		default:
			builder.Append(0)
		}
	case *array.Int64Builder:
		switch x := v.(type) {
		case int:
			builder.Append(int64(x))
		case int32:
			builder.Append(int64(x))
		case int64:
			builder.Append(x)
		default:
			builder.Append(0)
		}
	case *array.Float64Builder:
		switch x := v.(type) {
		case float32:
			builder.Append(float64(x))
		case float64:
			builder.Append(x)
		case int:
			builder.Append(float64(x))
		case int64:
			builder.Append(float64(x))
		default:
			builder.Append(0)
		}
	case *array.TimestampBuilder:
		switch x := v.(type) {
		case time.Time:
			builder.Append(arrow.Timestamp(x.UnixMilli()))
		default:
			builder.AppendNull()
		}
	default:
		b.AppendNull()
	}
}
