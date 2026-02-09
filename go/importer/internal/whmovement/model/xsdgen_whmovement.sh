#!/bin/sh
set -e

XSD_DIR="../../../gthsdu-lib-sbx-erp-data-importer/modules/whmovement/src/main/xsd"
OUT="whmovement_xsd.go"
TMP=$(mktemp)

awk '{print} /<xsd:schema/{exit}' "$XSD_DIR/S3SC_WHMovementSyncDel.xsd" > "$TMP"

append_inner() {
  awk 'BEGIN{inSchema=0} /<xsd:schema/{inSchema=1; next} /<xsd:include/{next} /<\/xsd:schema>/{inSchema=0; next} inSchema{print}' "$1" >> "$TMP"
}

append_inner "$XSD_DIR/S3SC_Types.xsd"
append_inner "$XSD_DIR/S3SC_BD_Types.xsd"
append_inner "$XSD_DIR/S3SC_MGT_Types.xsd"
append_inner "$XSD_DIR/S3SC_WHMovementSyncDel.xsd"

echo "</xsd:schema>" >> "$TMP"

/Users/vcrini/go/bin/xsdgen -ns http://www.csc.com/stealth3000 -o "$OUT" -pkg model "$TMP"

rm -f "$TMP"
