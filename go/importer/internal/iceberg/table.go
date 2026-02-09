package iceberg

import "fmt"

type MemoryTable struct {
	name   string
	schema TableSchema
}

func NewMemoryTable(schema TableSchema) *MemoryTable {
	return &MemoryTable{name: schema.Name, schema: schema}
}

func (t *MemoryTable) Name() string {
	return t.name
}

func (t *MemoryTable) Schema() TableSchema {
	return t.schema
}

func (t *MemoryTable) Upsert(records []Record) error {
	if len(records) == 0 {
		return nil
	}
	return fmt.Errorf("iceberg upsert not implemented")
}
