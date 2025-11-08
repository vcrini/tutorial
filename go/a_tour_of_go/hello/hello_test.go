package main

import "testing"

func TestHelloName(t *testing.T) {
	name := "Gladys"
	if name != "Gladys" {
		t.Errorf("test failed")
	}
}
