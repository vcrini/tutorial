package iceberg

type Record map[string]any

type SchemaField struct {
	Name     string
	Type     string
	Required bool
}

type TableSchema struct {
	Name       string
	Fields     []SchemaField
	Partition  []string
	SortFields []string
}
