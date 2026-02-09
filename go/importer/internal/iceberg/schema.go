package iceberg

import (
	"fmt"
	"strings"

	"github.com/apache/iceberg-go"
)

func BuildSchema(schema TableSchema) (*iceberg.Schema, error) {
	fields := make([]iceberg.NestedField, 0, len(schema.Fields))
	fieldIDs := make([]int, 0, len(schema.Fields))
	id := 1
	for _, f := range schema.Fields {
		iceType, err := toIcebergType(f.Type)
		if err != nil {
			return nil, err
		}
		fields = append(fields, iceberg.NestedField{ID: id, Name: f.Name, Type: iceType, Required: f.Required})
		fieldIDs = append(fieldIDs, id)
		id++
	}

	s := iceberg.NewSchemaWithIdentifiers(1, fieldIDs, fields...)
	return s, nil
}

func toIcebergType(t string) (iceberg.Type, error) {
	switch strings.ToLower(t) {
	case "string":
		return iceberg.PrimitiveTypes.String, nil
	case "bool", "boolean":
		return iceberg.PrimitiveTypes.Bool, nil
	case "int", "int32":
		return iceberg.PrimitiveTypes.Int32, nil
	case "int64", "long":
		return iceberg.PrimitiveTypes.Int64, nil
	case "float", "double", "decimal":
		return iceberg.PrimitiveTypes.Float64, nil
	case "timestamp":
		return iceberg.PrimitiveTypes.Timestamp, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", t)
	}
}
