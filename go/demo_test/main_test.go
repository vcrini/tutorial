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
