package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Exe(s []string) string {
	cmd := exec.Command(s[0], s[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot run:\n%s\n%s\n '%s'", strings.Join(s, "+"), out, err)
		os.Exit(1)
	}
	return string(out)
}
