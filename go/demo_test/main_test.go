package demo_test

import (
	"reflect"
	"testing"
)

func TestSum(t *testing.T) {
	got := Sum(2, 2)
	if got != 4 {
		t.Errorf("Sum(2,2) = %d; want 4", got)
	}
}

func TestStruct(t *testing.T) {
	if ReturnStruct().X != 1 {
		t.Errorf("ReturnStruct().X = %d; want 1", ReturnStruct().X)
	}
}

func TestArray(t *testing.T) {
	v := ReturnArray()
	if v[0] != "dangerous" {
		t.Errorf("ReturnArray()[0] = %s; want \"dangerous\"", ReturnArray()[0])
	}
	v[0] = "dungeons"
	if v[0] != "dungeons" {
		t.Errorf("ReturnArray()[0] = %s; want \"dangerous\"", v[0])
	}
}

func TestSliceLiteral(t *testing.T) {
	v := ReturnLiteralSlice()
	if v[0] != "dungeons" {
		t.Errorf("ReturnArray()[0] = %s; want \"dungeons\"", ReturnArray()[0])
	}
	v = v[:1]
	if z := len(v); z != 1 {
		t.Errorf("%d; want 1", z)
	}
	if z := cap(v); z != 3 {
		t.Errorf("%d; want 3", z)
	}
}

func TestMap(t *testing.T) {
	v := ReturnMap()
	if v["O"].X != 0 {
		t.Errorf(" %d; want \"0\"", v["O"].X)
	}
	if !reflect.DeepEqual(v["O"], name{0, 0}) {
		t.Errorf(" %d; want \"{0,0}\"", v["O"])
	}
	delete(v, "O")
	if _, ok := v["O"]; ok {
		t.Errorf("want \"nil\"")
	}
}
