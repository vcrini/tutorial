package model

type ErrorKind string

const (
	ErrorFileFormat ErrorKind = "file_format"
	ErrorDecode     ErrorKind = "decode"
)

type Error[C any] struct {
	Kind      ErrorKind
	SocCod    string
	BoType    string
	BoCod     string
	Operation Operation
	Ctx       C
}

func NewFileFormatError[C any](ctx C) Error[C] {
	return Error[C]{
		Kind: ErrorFileFormat,
		Ctx:  ctx,
	}
}

func NewDecodeError[C any](socCod, boType, boCod string, op Operation, ctx C) Error[C] {
	return Error[C]{
		Kind:      ErrorDecode,
		SocCod:    socCod,
		BoType:    boType,
		BoCod:     boCod,
		Operation: op,
		Ctx:       ctx,
	}
}
