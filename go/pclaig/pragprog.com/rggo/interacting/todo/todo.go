package todo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// item struct represents a ToDo items
type item struct {
	Task        string
	Done        bool
	CreatedAt   time.Time
	CompletedAt time.Time
}

// List represents a list of ToDo items
type List []item

// Add creates a new todo item and appends it to the list
func (l *List) Add(task string) {
	t := item{
		Task:        task,
		Done:        false,
		CreatedAt:   time.Now(),
		CompletedAt: time.Time{},
	}
	*l = append(*l, t)
}

// Complete method marks a ToDo item as completed by
// settings Done = true and CompletedAt to the current time
func (l *List) Complete(i int) error {
	ls := *l
	if i < 0 || i > len(ls) {
		return fmt.Errorf("Item %d does not exists", i)
	}
	// Adjusting index for 0 biased index
	ls[i-1].Done = true
	ls[i-1].CompletedAt = time.Now()
	return nil
}

//Delete method deletes a ToDo item from the list
func (l *List) Delete(i int) error {
	ls := *l
	if i < 0 || i > len(ls) {
		return fmt.Errorf("Item %d does not exists", i)
	}
	//not fully understood why with ls is on right is not working
	*l = append(ls[:i-1], ls[i:]...)
	return nil
}

// Save encodes the list as Json and saves it
// using the provided file name
func (l *List) Save(filename string) error {
	js, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, js, 0644)
}
func (l *List) Get(filename string) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(file) == 0 {
		return nil
	}
	return json.Unmarshal(file, l)

}
