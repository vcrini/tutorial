package main

import (
  "flag"
	"fmt"
	"os"

	"pragprog.com/rggo/interacting/todo"
)

// Hardcoding the file name
const todoFileName = ".todo.json"

func main() {
  //parsing command line flags
  task:= flag.String("task","", "Task to be included in the todo list")
  list:= flag.Bool("list",false, "List all tasks")
  complete:= flag.Int("complete",0, "Item to be completed")
  flag.Parse()

	l := &todo.List{}
	if err := l.Get(todoFileName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	//decide what to do according on number of arguments provided
	switch {
	case *list:
		// List current to do items
		for _, item := range *l {
      if !item.Done {
			fmt.Println(item.Task)
      }
		}

	// concatenate all provided params with a space and add to the list as an item
	default:
		//concatenate all arguments with a space
		item := strings.Join(os.Args[1:], " ")
		// add the task
		l.Add(item)
		// save the new list
		if err := l.Save(todoFileName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
