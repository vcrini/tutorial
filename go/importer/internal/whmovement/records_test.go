package whmovement

import (
	"testing"

	coremodel "importer/internal/model"
	whmodel "importer/internal/whmovement/model"
)

func TestDatoAggMapping(t *testing.T) {
	rec := map[string]any{}
	list := []whmodel.TDATOAGG{{CAR: "X", NUMDAGG: 1}}
	addDatoAgg(rec, list)
	if rec["dagg_type1"] != "CAR" {
		t.Fatalf("expected dagg_type1 to be CAR, got %v", rec["dagg_type1"])
	}
}

func TestKeyMgt(t *testing.T) {
	mgt := whmodel.TBMGT{MGTSOCCOD: "045", MGTMGACOD: "M01", MGTANNO: 2025, MGTINMCOD: "C", MGTNUM: 12}
	key := KeyMgt(mgt)
	if key != "045-M01-2025-C-12" {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestToRecordsBase(t *testing.T) {
	data := whmodel.TWHMovementSyncDel{
		DATAAREA: whmodel.TWHMovementSyncDelDA{
			WHMOVEMENT: whmodel.TBMGT{MGTSOCCOD: "045", MGTMGACOD: "M01", MGTANNO: 2025, MGTINMCOD: "C", MGTNUM: 12},
		},
		APPLICATIONAREA: whmodel.TApplicationArea{DATACREAZIONE: "2025-01-01T00:00:00"},
	}
	block := coremodel.Block[whmodel.TWHMovementSyncDel, any]{}
	out := ToRecords(data, block, false)
	if len(out) == 0 {
		t.Fatalf("expected records, got none")
	}
	if _, ok := out["mgt"]; !ok {
		t.Fatalf("expected mgt table in output")
	}
}
