package iceberg

import "github.com/apache/iceberg-go"

func ToTableSchema(schema *iceberg.Schema) TableSchema {
	fields := make([]SchemaField, 0, schema.NumFields())
	for _, f := range schema.Fields() {
		fields = append(fields, SchemaField{
			Name:     f.Name,
			Type:     f.Type.String(),
			Required: f.Required,
		})
	}
	return TableSchema{Name: "", Fields: fields}
}
