package whmovement

import (
	"fmt"
	"strconv"

	whmodel "importer/internal/whmovement/model"
)

func KeyMgt(mgt whmodel.TBMGT) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		valueString(mgt.MGTSOCCOD),
		valueString(mgt.MGTMGACOD),
		intString(mgt.MGTANNO),
		valueString(mgt.MGTINMCOD),
		intString(mgt.MGTNUM),
	)
}

func KeyMgtAssoc(mgt whmodel.TBMGT, assoc whmodel.TBMGTASSOC) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s",
		KeyMgt(mgt),
		valueString(assoc.MGTSOCCOD),
		valueString(assoc.MGTMGACOD),
		intString(assoc.MGTANNO),
		valueString(assoc.MGTINMCOD),
		intString(assoc.MGTNUM),
	)
}

func KeyMgtLni(mgt whmodel.TBMGT, lni whmodel.TBMGTLNI) string {
	return fmt.Sprintf("%s-%s", KeyMgt(mgt), valueString(lni.MGTLNILNGCOD))
}

func KeyMgtSpe(mgt whmodel.TBMGT, spe whmodel.TBMGTSPE) string {
	return fmt.Sprintf("%s-%s", KeyMgt(mgt), valueString(spe.MGTSPESPECOD))
}

func KeyMgtNot(mgt whmodel.TBMGT, not whmodel.TNOT) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgt(mgt), valueString(not.TPNCOD), valueString(not.LNGCOD))
}

func KeyMgr(mgt whmodel.TBMGT, mgr whmodel.TBMGR) string {
	return fmt.Sprintf("%s-%s", KeyMgt(mgt), intString(mgr.MGRRIGA))
}

func KeyMgrAge(mgt whmodel.TBMGT, mgr whmodel.TBMGR, mgrAge whmodel.TBMGRAGE) string {
	return fmt.Sprintf("%s-%s", KeyMgr(mgt, mgr), valueString(mgrAge.MGRAGEAGECODC))
}

func KeyMgrCds(mgt whmodel.TBMGT, mgr whmodel.TBMGR, mgrCds whmodel.TBMGRCDS) string {
	return fmt.Sprintf("%s-%s", KeyMgr(mgt, mgr), valueString(mgrCds.MGRCDSCSTCOD))
}

func KeyMgrAssoc(mgt whmodel.TBMGT, mgr whmodel.TBMGR, mgrAssoc whmodel.TBMGRASSOC) string {
	return fmt.Sprintf("%s-%s", KeyMgr(mgt, mgr), intString(mgrAssoc.MGRRIGA))
}

func KeyMgrBar(mgt whmodel.TBMGT, mgr whmodel.TBMGR, bar whmodel.TBMGRBAR) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s",
		KeyMgr(mgt, mgr),
		valueString(bar.MGRBARBARCOD),
		valueString(bar.MGRBARTBACOD),
		valueString(bar.MGRBARCLICOD),
		valueString(bar.MGRBARFRNCOD),
		valueString(bar.MGRBARIDENTIFICATIVO),
	)
}

func KeyMgrGrtFpr(mgt whmodel.TBMGT, mgr whmodel.TBMGR, grt whmodel.TBMGRGRTFPR) string {
	return fmt.Sprintf("%s-%s-%s",
		KeyMgr(mgt, mgr),
		valueString(grt.MGRGRTFPRGRTCOD),
		intString(grt.MGRGRTFPRGRTFPRCOD),
	)
}

func KeyMgrMgc(mgt whmodel.TBMGT, mgr whmodel.TBMGR, mgc whmodel.TBMGC) string {
	return fmt.Sprintf("%s-%s", KeyMgr(mgt, mgr), intString(mgc.MGCORD))
}

func KeyMgrMgp(mgt whmodel.TBMGT, mgr whmodel.TBMGR, mgp whmodel.TBMGP) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgr(mgt, mgr), valueString(mgp.MGPGRTCOD), valueString(mgp.MGPTGLCOD))
}

func KeyMgrTgl(mgt whmodel.TBMGT, mgr whmodel.TBMGR, tgl whmodel.TBMGRTGL) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgr(mgt, mgr), valueString(tgl.MGRTGLGRTCOD), valueString(tgl.MGRTGLTGLCOD))
}

func KeyMgrNot(mgt whmodel.TBMGT, mgr whmodel.TBMGR, not whmodel.TNOT) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgr(mgt, mgr), valueString(not.TPNCOD), valueString(not.LNGCOD))
}

func KeyMgrAgePrv(mgt whmodel.TBMGT, mgr whmodel.TBMGR, mgrAge whmodel.TBMGRAGE, prv whmodel.TBMGRAGEPRV) string {
	return fmt.Sprintf("%s-%s", KeyMgrAge(mgt, mgr, mgrAge), intString(prv.MGRAGEPRVORD))
}

func KeyMgrCdsCprTgl(mgt whmodel.TBMGT, mgr whmodel.TBMGR, cds whmodel.TBMGRCDS, cpr whmodel.TBMGRCDSCPRTGL) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgrCds(mgt, mgr, cds), valueString(cpr.MGRCDSCPRTGLGRTCODP), valueString(cpr.MGRCDSCPRTGLTGLCODP))
}

func KeyMgrAssocBar(mgt whmodel.TBMGT, mgr whmodel.TBMGR, assoc whmodel.TBMGRASSOC, bar whmodel.TBMGRBAR) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s",
		KeyMgrAssoc(mgt, mgr, assoc),
		valueString(bar.MGRBARBARCOD),
		valueString(bar.MGRBARTBACOD),
		valueString(bar.MGRBARCLICOD),
		valueString(bar.MGRBARFRNCOD),
		valueString(bar.MGRBARIDENTIFICATIVO),
	)
}

func KeyMgrAssocTgl(mgt whmodel.TBMGT, mgr whmodel.TBMGR, assoc whmodel.TBMGRASSOC, tgl whmodel.TBMGRTGLASSOC) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgrAssoc(mgt, mgr, assoc), valueString(tgl.MGRTGLGRTCOD), valueString(tgl.MGRTGLTGLCOD))
}

func KeyMgrAssocVal(mgt whmodel.TBMGT, mgr whmodel.TBMGR, assoc whmodel.TBMGRASSOC, val whmodel.TBMGPVALASSOC) string {
	return fmt.Sprintf("%s-%s-%s", KeyMgrAssoc(mgt, mgr, assoc), valueString(val.MGPGRTCOD), valueString(val.MGPTGLCOD))
}

func BoID(record whmodel.TWHMovementSyncDel) string {
	return KeyMgt(record.DATAAREA.WHMOVEMENT)
}

func BoPartitionKey(record whmodel.TWHMovementSyncDel) string {
	return valueString(record.DATAAREA.WHMOVEMENT.MGTDATAINS)
}

func BoOrderingID(record whmodel.TWHMovementSyncDel) string {
	return valueString(record.APPLICATIONAREA.DATACREAZIONE)
}

func valueString(v any) string {
	return fmt.Sprint(v)
}

func intString(v int) string {
	return strconv.FormatInt(int64(v), 10)
}
