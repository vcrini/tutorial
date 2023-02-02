package main

import (
	"bytes"
	"testing"
)

// TestCountWords tests the count function set to count words
func TestCountWords(t *testing.T) {
	b := bytes.NewBufferString("parola1 parola2 parola3 parola4")
	exp := 4
	res := count(b)
	if res != exp {
		t.Errorf("Expected %d, got %d instead.\n", exp, res)
	}
}
