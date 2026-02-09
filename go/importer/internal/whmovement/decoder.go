package whmovement

import (
	"encoding/xml"

	"importer/internal/streams"
	whmodel "importer/internal/whmovement/model"
)

func NewDecoder() streams.Decoder[whmodel.TWHMovementSyncDel] {
	return streams.XMLDecoder[whmodel.TWHMovementSyncDel]{
		DecodeFunc: func(data []byte) (whmodel.TWHMovementSyncDel, error) {
			var out whmodel.TWHMovementSyncDel
			if err := xml.Unmarshal(data, &out); err != nil {
				return whmodel.TWHMovementSyncDel{}, err
			}
			return out, nil
		},
	}
}
