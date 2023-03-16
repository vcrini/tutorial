package main_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	binName  = "todo"
	fileName = ".test.json"
)

func TestMain(m *testing.M) {
	if err := os.Setenv("TODO_FILENAME", fileName); err != nil {
		fmt.Fprintf(os.Stderr, "%s \nCannot set env variable %s", err, "TODO_FILENAME")
	}
	fmt.Printf("Building tool... with test file '%s'", os.Getenv("TODO_FILENAME"))
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	build := exec.Command("go", "build", "-o", binName)

	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot build tool %s: %s", binName, err)
		os.Exit(1)
	}

	fmt.Println("Running tests...")
	result := m.Run()

	fmt.Println("Cleaning up...")
	os.Remove(binName)
	os.Remove(fileName)
	os.Exit(result)
}
func TestTodoCLI(t *testing.T) {
	task := "test task number 1"

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(dir, binName)
	t.Run("AddNewTask", func(t *testing.T) {
		cmd := exec.Command(cmdPath, "-add", task)
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
	task2 := "test number 2"
	t.Run("AddNewTestFromSTDIN", func(t *testing.T) {
		cmd := exec.Command(cmdPath, "-add")
		cmdStdIn, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(cmdStdIn, task2); err != nil {
			t.Fatal(err)
		}
		cmdStdIn.Close()
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ListTasks", func(t *testing.T) {
		cmd := exec.Command(cmdPath, "-list")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}

		expected := fmt.Sprintf("  1: %s\n  2: %s\n", task, task2)
		sOut := string(out)
		if expected != sOut {
			t.Errorf("Expected %q, got %q instead\n", expected, sOut)
		}

	})
	t.Run("CompleteTask", func(t *testing.T) {
		cmd := exec.Command(cmdPath, "-complete", "1")
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command(cmdPath, "-list")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}

		expected := fmt.Sprintf("X 1: %s\n  2: %s\n", task, task2)
		sOut := string(out)
		if expected != sOut {
			t.Errorf("Expected %q, got %q instead\n", expected, sOut)
		}

	})
	task2 = "t1\nt2"
	t.Run("AddNewTestFromSTDINWithNewLines", func(t *testing.T) {
		cmd := exec.Command(cmdPath, "-add")
		cmdStdIn, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(cmdStdIn, task2); err != nil {
			t.Fatal(err)
		}
		cmdStdIn.Close()
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ListTasks", func(t *testing.T) {
		cmd := exec.Command(cmdPath, "-list")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}

		expected := "X 1: test task number 1\n  2: test number 2\n  3: t1\n  4: t2\n"
		sOut := string(out)
		if expected != sOut {
			t.Errorf("Expected %q, got %q instead\n", expected, sOut)
		}

	})
}
