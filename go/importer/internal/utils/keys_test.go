package utils

import "testing"

func TestKeySplit(t *testing.T) {
	key := "data/landing_zone/045/WHMOVEMENT/20250506065250_620408282_S.xml"
	el, err := KeySplit(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if el.SocCod != "045" || el.BoType != "WHMOVEMENT" || el.BoCod != "620408282" {
		t.Fatalf("unexpected parsed element: %+v", el)
	}
	if el.Operation != "sync" {
		t.Fatalf("unexpected operation: %v", el.Operation)
	}
}

func TestKeySplitReinit(t *testing.T) {
	key := "data/archive/045/WHMOVEMENT/045-M01L-2025-C-153258.xml"
	el, err := KeySplitReinit(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if el.SocCod != "045" || el.BoType != "WHMOVEMENT" || el.BoCod != "045-M01L-2025-C-153258" {
		t.Fatalf("unexpected parsed element: %+v", el)
	}
	if el.Operation != "sync" {
		t.Fatalf("unexpected operation: %v", el.Operation)
	}
}
