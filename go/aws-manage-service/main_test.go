package main

import (
	"strings"
	"testing"

	"github.com/vcrini/go-utils"
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
	res := strings.TrimSuffix(utils.Exe(command), "\n")
	// TODO: it's not os neutral
	if res != exp {
		t.Errorf("Expected '%s', got '%s' instead.\n", exp, res)
	}
}
