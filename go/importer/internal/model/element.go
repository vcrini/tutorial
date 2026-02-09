package model

type Element[D any, C any] struct {
	Data      D
	SocCod    string
	BoType    string
	BoCod     string
	Operation Operation
	Size      int64
	Ctx       C
}
