package streams

import "fmt"

type XMLDecoder[D any] struct {
	DecodeFunc func(data []byte) (D, error)
}

func (d XMLDecoder[D]) Decode(xmlString string) (D, error) {
	var zero D
	if d.DecodeFunc == nil {
		return zero, fmt.Errorf("DecodeFunc is required")
	}

	return d.DecodeFunc([]byte(xmlString))
}
