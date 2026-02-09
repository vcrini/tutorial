package whmovement

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"importer/internal/iceberg"
	coremodel "importer/internal/model"
	whmodel "importer/internal/whmovement/model"
	"importer/internal/whmovement/tables"
)

func ToRecords[C any](data whmodel.TWHMovementSyncDel, block coremodel.Block[whmodel.TWHMovementSyncDel, C], deleted bool) map[string][]iceberg.Record {
	mgt := data.DATAAREA.WHMOVEMENT
	boID := KeyMgt(mgt)
	partition := BoPartitionKey(data)

	out := map[string][]iceberg.Record{}

	appendRecord := func(table string, record iceberg.Record) {
		out[table] = append(out[table], record)
	}

	// MGT
	mgtRec := baseRecord(block, boID, KeyMgt(mgt), partition, deleted)
	addFieldsByPrefix(mgtRec, mgt, "MGT_")
	addDatoAgg(mgtRec, mgt.DATOAGG)
	appendRecord(tables.MgtTable.Name, mgtRec)

	// MGT_ASSOC
	for _, assoc := range mgt.DETTBMGTASSOC.BMGTASSOC {
		rec := baseRecord(block, boID, KeyMgtAssoc(mgt, assoc), partition, deleted)
		addFieldsByPrefix(rec, assoc, "MGT_")
		addMainMgtKeys(rec, mgt)
		addDatoAgg(rec, assoc.DATOAGG)
		appendRecord(tables.MgtAssocTable.Name, rec)
	}

	// MGT_LNI
	for _, lni := range mgt.DETTBMGTLNI.BMGTLNI {
		rec := baseRecord(block, boID, KeyMgtLni(mgt, lni), partition, deleted)
		addAllFields(rec, lni)
		addMgtKeys(rec, mgt)
		appendRecord(tables.MgtLniTable.Name, rec)
	}

	// MGT_SPE
	for _, spe := range mgt.DETTBMGTSPE.BMGTSPE {
		rec := baseRecord(block, boID, KeyMgtSpe(mgt, spe), partition, deleted)
		addAllFields(rec, spe)
		addMgtKeys(rec, mgt)
		appendRecord(tables.MgtSpeTable.Name, rec)
	}

	// MGT_NOT
	for _, notItem := range mgt.DETTBMGTNOT.BMGTNOT {
		rec := baseRecord(block, boID, KeyMgtNot(mgt, notItem), partition, deleted)
		addAllFields(rec, notItem)
		addMgtKeys(rec, mgt)
		appendRecord(tables.MgtNotTable.Name, rec)
	}

	// MGT_MGR and nested structures
	for _, mgr := range mgt.DETTBMGR.BMGR {
		mgrRec := baseRecord(block, boID, KeyMgr(mgt, mgr), partition, deleted)
		addFieldsByPrefix(mgrRec, mgr, "MGR_")
		addMgtKeys(mgrRec, mgt)
		addDatoAgg(mgrRec, mgr.DATOAGG)
		appendRecord(tables.MgtMgrTable.Name, mgrRec)

		for _, assoc := range mgr.DETTBMGRASSOC.BMGRASSOC {
			rec := baseRecord(block, boID, KeyMgrAssoc(mgt, mgr, assoc), partition, deleted)
			addFieldsByPrefix(rec, assoc, "MGR_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			addDatoAgg(rec, assoc.DATOAGG)
			appendRecord(tables.MgtMgrAssocTable.Name, rec)
		}

		for _, tgl := range mgr.DETTBMGRTGL.BMGRTGL {
			rec := baseRecord(block, boID, KeyMgrTgl(mgt, mgr, tgl), partition, deleted)
			addFieldsByPrefix(rec, tgl, "MGR_TGL_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrTglTable.Name, rec)
		}

		for _, age := range mgr.DETTBMGRAGE.BMGRAGE {
			rec := baseRecord(block, boID, KeyMgrAge(mgt, mgr, age), partition, deleted)
			addFieldsByPrefix(rec, age, "MGR_AGE_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrAgeTable.Name, rec)

			for _, prv := range age.DETTBMGRAGEPRV.BMGRAGEPRV {
				prvRec := baseRecord(block, boID, KeyMgrAgePrv(mgt, mgr, age, prv), partition, deleted)
				addFieldsByPrefix(prvRec, prv, "MGR_AGE_PRV_")
				addMgtKeys(prvRec, mgt)
				addMgrKey(prvRec, mgr)
				appendRecord(tables.MgtMgrAgePrvTable.Name, prvRec)
			}
		}

		for _, mgc := range mgr.DETTBMGC.BMGC {
			rec := baseRecord(block, boID, KeyMgrMgc(mgt, mgr, mgc), partition, deleted)
			addFieldsByPrefix(rec, mgc, "MGC_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrMgcTable.Name, rec)
		}

		for _, mgp := range mgr.DETTBMGP.BMGP {
			rec := baseRecord(block, boID, KeyMgrMgp(mgt, mgr, mgp), partition, deleted)
			addFieldsByPrefix(rec, mgp, "MGP_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrMgpTable.Name, rec)
		}

		for _, grt := range mgr.DETTBMGRGRTFPR.BMGRGRTFPR {
			rec := baseRecord(block, boID, KeyMgrGrtFpr(mgt, mgr, grt), partition, deleted)
			addFieldsByPrefix(rec, grt, "MGR_GRT_FPR_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrGrtFprTable.Name, rec)
		}

		for _, notItem := range mgr.DETTBMGRNOT.BMGRNOT {
			rec := baseRecord(block, boID, KeyMgrNot(mgt, mgr, notItem), partition, deleted)
			addAllFields(rec, notItem)
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrNotTable.Name, rec)
		}

		for _, bar := range mgr.DETTBMGRBAR.BMGRBAR {
			rec := baseRecord(block, boID, KeyMgrBar(mgt, mgr, bar), partition, deleted)
			addFieldsByPrefix(rec, bar, "MGR_BAR_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrBarTable.Name, rec)
		}

		for _, cds := range mgr.DETTBMGRCDS.BMGRCDS {
			rec := baseRecord(block, boID, KeyMgrCds(mgt, mgr, cds), partition, deleted)
			addFieldsByPrefix(rec, cds, "MGR_CDS_")
			addMgtKeys(rec, mgt)
			addMgrKey(rec, mgr)
			appendRecord(tables.MgtMgrCdsTable.Name, rec)

			for _, cpr := range cds.DETTBMGRCDSCPRTGL.BMGRCDSCPRTGL {
				cprRec := baseRecord(block, boID, KeyMgrCdsCprTgl(mgt, mgr, cds, cpr), partition, deleted)
				addFieldsByPrefix(cprRec, cpr, "MGR_CDS_CPR_TGL_")
				addMgtKeys(cprRec, mgt)
				addMgrKey(cprRec, mgr)
				appendRecord(tables.MgtMgrCdsCprTglTable.Name, cprRec)
			}
		}

		for _, assoc := range mgr.DETTBMGRASSOC.BMGRASSOC {
			for _, barAssoc := range assoc.DETTBMGRBARASSOC.BMGRBARASSOC {
				rec := baseRecord(block, boID, KeyMgrAssocBar(mgt, mgr, assoc, barAssoc), partition, deleted)
				addFieldsByPrefix(rec, barAssoc, "MGR_BAR_")
				addMainMgtKeys(rec, mgt)
				addMainMgrKey(rec, mgr)
				addMgrAssocKey(rec, assoc)
				appendRecord(tables.MgtMgrAssocBarTable.Name, rec)
			}

			for _, tglAssoc := range assoc.DETTBMGRTGLASSOC.BMGRTGLASSOC {
				rec := baseRecord(block, boID, KeyMgrAssocTgl(mgt, mgr, assoc, tglAssoc), partition, deleted)
				addFieldsByPrefix(rec, tglAssoc, "MGR_TGL_")
				addMainMgtKeys(rec, mgt)
				addMainMgrKey(rec, mgr)
				addMgrAssocKey(rec, assoc)
				appendRecord(tables.MgtMgrAssocTglTable.Name, rec)
			}

			for _, valAssoc := range assoc.DETTBMGPVALASSOC.BMGPVALASSOC {
				rec := baseRecord(block, boID, KeyMgrAssocVal(mgt, mgr, assoc, valAssoc), partition, deleted)
				addFieldsByPrefix(rec, valAssoc, "MGP_")
				addMainMgtKeys(rec, mgt)
				addMainMgrKey(rec, mgr)
				addMgrAssocKey(rec, assoc)
				appendRecord(tables.MgtMgrAssocValTable.Name, rec)
			}
		}
	}

	return out
}

func baseRecord[C any](block coremodel.Block[whmodel.TWHMovementSyncDel, C], boID, rowID, partition string, deleted bool) iceberg.Record {
	lastUpdate := block.Created
	if lastUpdate.IsZero() {
		lastUpdate = time.Now().UTC()
	}
	return iceberg.Record{
		"sandbox_package_guid": block.UUID.String(),
		"last_update":          lastUpdate,
		"bo_id":                boID,
		"bo_partition_key":     partition,
		"row_id":               rowID,
		"deleted":              deleted,
	}
}

func addAllFields(record iceberg.Record, v any) {
	addFields(record, v, func(string) bool { return true })
}

func addFieldsByPrefix(record iceberg.Record, v any, prefix string) {
	addFields(record, v, func(name string) bool { return strings.HasPrefix(name, prefix) })
}

func addFieldsByNames(record iceberg.Record, v any, names map[string]struct{}) {
	addFields(record, v, func(name string) bool { _, ok := names[name]; return ok })
}

func addFields(record iceberg.Record, v any, allow func(name string) bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)
		xmlTag := fieldType.Tag.Get("xml")
		name := xmlFieldName(xmlTag)
		if name == "" || name == "-" {
			continue
		}
		if !allow(name) {
			continue
		}
		if !isScalar(field) {
			continue
		}

		record[strings.ToLower(name)] = field.Interface()
	}
}

func xmlFieldName(tag string) string {
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, " ")
	name := parts[len(parts)-1]
	name = strings.Split(name, ",")[0]
	return name
}

func isScalar(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Struct:
		return v.Type().PkgPath() == "time" && v.Type().Name() == "Time"
	default:
		return false
	}
}

func addDatoAgg(record iceberg.Record, list []whmodel.TDATOAGG) {
	for _, item := range list {
		num := fmt.Sprint(item.NUMDAGG)
		typeName, value := datoAggTypeValue(item)
		if typeName == "" {
			continue
		}
		record["dagg_type"+num] = typeName
		if value != "" {
			record["dagg_value"+num] = value
		}
	}
}

func datoAggTypeValue(item whmodel.TDATOAGG) (string, string) {
	if item.CAR != "" {
		return "CAR", item.CAR
	}
	if item.DAT != "" {
		return "DAT", string(item.DAT)
	}
	if item.NUM != "" {
		return "NUM", string(item.NUM)
	}
	return "", ""
}

func addMgtKeys(record iceberg.Record, mgt whmodel.TBMGT) {
	addFieldsByNames(record, mgt, map[string]struct{}{
		"MGT_SOC_COD": {},
		"MGT_MGA_COD": {},
		"MGT_ANNO":    {},
		"MGT_INM_COD": {},
		"MGT_NUM":     {},
	})
}

func addMainMgtKeys(record iceberg.Record, mgt whmodel.TBMGT) {
	record["main_mgt_soc_cod"] = mgt.MGTSOCCOD
	record["main_mgt_mga_cod"] = mgt.MGTMGACOD
	record["main_mgt_anno"] = mgt.MGTANNO
	record["main_mgt_inm_cod"] = mgt.MGTINMCOD
	record["main_mgt_num"] = mgt.MGTNUM
}

func addMgrKey(record iceberg.Record, mgr whmodel.TBMGR) {
	record["mgr_riga"] = mgr.MGRRIGA
}

func addMainMgrKey(record iceberg.Record, mgr whmodel.TBMGR) {
	record["main_mgr_riga"] = mgr.MGRRIGA
}

func addMgrAssocKey(record iceberg.Record, assoc whmodel.TBMGRASSOC) {
	record["mgr_riga"] = assoc.MGRRIGA
}
