package utils

import (
	"fmt"
	"strings"

	"importer/internal/model"
)

// keySplit parses keys with naming convention:
// data/landing_zone/{SOC_COD}/{BO_TYPE}/{timestamp}_{boCod}_{op}.xml
func KeySplit(key string) (model.Element[string, string], error) {
	parts := strings.Split(key, "/")
	if len(parts) < 5 {
		return model.Element[string, string]{}, fmt.Errorf("invalid key format")
	}

	socCod := parts[2]
	boType := parts[3]
	fileName := parts[4]
	fileStem := strings.SplitN(fileName, ".", 2)[0]
	fileNameSplit := strings.Split(fileStem, "_")
	if len(fileNameSplit) < 3 {
		return model.Element[string, string]{}, fmt.Errorf("invalid file name")
	}

	boCod := fileNameSplit[1]
	var op model.Operation
	switch fileNameSplit[2] {
	case "S":
		op = model.OperationSync
	case "D":
		op = model.OperationDelete
	default:
		return model.Element[string, string]{}, fmt.Errorf("invalid operation")
	}

	return model.Element[string, string]{
		Data:      fileName,
		SocCod:    socCod,
		BoType:    boType,
		BoCod:     boCod,
		Operation: op,
		Size:      0,
		Ctx:       key,
	}, nil
}

// keySplitReinit parses keys with naming convention:
// data/archive/{SOC_COD}/{BO_TYPE}/{boCod}.xml
func KeySplitReinit(key string) (model.Element[string, string], error) {
	parts := strings.Split(key, "/")
	if len(parts) < 5 {
		return model.Element[string, string]{}, fmt.Errorf("invalid key format")
	}

	socCod := parts[2]
	boType := parts[3]
	fileName := parts[4]
	boCod := strings.SplitN(fileName, ".", 2)[0]

	return model.Element[string, string]{
		Data:      fileName,
		SocCod:    socCod,
		BoType:    boType,
		BoCod:     boCod,
		Operation: model.OperationSync,
		Size:      0,
		Ctx:       key,
	}, nil
}
