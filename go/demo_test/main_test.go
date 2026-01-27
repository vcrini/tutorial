package demo_test

import "testing"

func TestSum(t *testing.T) {
	if Sum(2, 2) != 4 {
		t.Errorf("Sum(2,2) = %d; want 4", Sum(2, 2))
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
		t.Errorf("ReturnArray()[0] = %s; want \"dangerous\"", ReturnArray()[0])
	}
}
