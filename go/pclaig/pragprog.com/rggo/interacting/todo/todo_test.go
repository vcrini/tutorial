package todo_test

import (
	"io/ioutil"
	"os"
	"testing"

	"pragprog.com/rggo/interacting/todo"
)

// TestAdd tests the Add method of the list type
func TestAdd(t *testing.T) {
	l := todo.List{}
	taskName := "Nuovo task"
	l.Add(taskName)
	if l[0].Task != taskName {
		t.Errorf("Expected %q, got %q instead.", taskName, l[0].Task)
	}
}

// TestComplete tests the Complete method of the List type
func TestComplete(t *testing.T) {
	l := todo.List{}
	taskName := "Nuovo task"
	l.Add(taskName)
	if l[0].Task != taskName {
		t.Errorf("Expected %q, got %q instead.", taskName, l[0].Task)
	}
	if l[0].Done {
		t.Errorf("%q should not be completed", taskName)
	}
	l.Complete(1)
	if !l[0].Done {
		t.Errorf("%q should be completed", taskName)
	}
}

// TestDelete tests the Delete Method of the List type
func TestDelete(t *testing.T) {
	l := todo.List{}
	tasks := []string{
		"Task1",
		"Task2",
		"Task3",
	}
	for _, v := range tasks {
		l.Add(v)
	}
	if l[0].Task != tasks[0] {
		t.Errorf("Expected %q, got %q instead.", tasks[0], l[0].Task)
	}
	l.Delete(2)
	num := 2
	if len(l) != num {
		t.Errorf("Expected %d, got %d instead. %v", num, len(l), l)
	}
	if l[1].Task != tasks[2] {
		t.Errorf("Expected %q, got %q instead.", tasks[2], l[1].Task)
	}

}

// TestSaveGet tests Save and Get methods of the List type
func TestSaveGet(t *testing.T) {
	l1 := todo.List{}
	l2 := todo.List{}
	taskName := "Nuovo Task"

	l1.Add(taskName)
	if l1[0].Task != taskName {
		t.Errorf("Expected %q, got %q instead.", taskName, l1[0].Task)
	}
	tf, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Non riesco a creare il file temporaneo: %s", err)
	}
	defer os.Remove(tf.Name())
	if err := l1.Save(tf.Name()); err != nil {
		t.Fatalf("Error saving list to file: %s", err)
	}
	if err := l2.Get(tf.Name()); err != nil {
		t.Fatalf("Error getting list from file: %s", err)
	}
	if l1[0].Task != l2[0].Task {
		t.Errorf("Task %q should patch %q task", l1[0].Task, l2[0].Task)
	}
}
