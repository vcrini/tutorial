package main

import (
	"strings"
	"testing"
)

func TestDummy(t *testing.T) {
	exp := "dummy"
	res := "dummy"
	if res != exp {
		t.Errorf("Expected '%s', got '%s' instead.\n", exp, res)
	}
}
func TestExe(t *testing.T) {
	command := []string{"echo", "ciao"}
	exp := "ciao"
	res := strings.TrimSuffix(exe(command), "\n")
	// TODO: it's not os neutral
	if res != exp {
		t.Errorf("Expected '%s', got '%s' instead.\n", exp, res)
	}
}

// TestCountWords tests the count function set to count words
/*func TestCountWords(t *testing.T) {
	b := bytes.NewBufferString("parola1 parola2 parola3 parola4")
	exp := 4
	res := count(b, false, false)
	if res != exp {
		t.Errorf("Expected %d, got %d instead.\n", exp, res)
	}
}
*/
