package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"pragprog.com/rggo/interacting/todo"
)

// Hardcoding the file name
var todoFileName = ".todo.json"

func main() {
	//parsing command line flags
	add := flag.Bool("add", false, "Add task to the todo list")
	list := flag.Bool("list", false, "List all tasks")
	verbose := flag.Bool("verbose", false, "Display verbose output when listing tasks")
	list_todo := flag.Bool("list_todo", false, "List all tasks not yet completed")
	complete := flag.Int("complete", 0, "Item to be completed")
	del := flag.Int("delete", -1, "Item to be deleted")
	flag.Parse()

	l := &todo.List{}
	// if env is defined then use this file name
	if os.Getenv("TODO_FILENAME") != "" {
		todoFileName = os.Getenv("TODO_FILENAME")
	}
	if err := l.Get(todoFileName); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	//decide what to do according on number of arguments provided
	switch {
	case *list_todo:
		//List current tasks not yet completed
		for k, t := range *l {
			if !t.Done {
				fmt.Printf(" %d: %s\n", k+1, t.Task)
			}
		}
	case *list:
		// List current to do items
		if *verbose {

			for k, t := range *l {
				prefix := "  "
				if t.Done {
					prefix = "X "
				}
				fmt.Printf("%s%d: %s completed: %t %s %s\n", prefix, k+1, t.Task, t.Done, t.CreatedAt, t.CompletedAt)
			}

		} else {
			fmt.Print(l)
		}
	case *del > 0:
		//delete given item
		if err := l.Delete(*del); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		//save new list
		if err := l.Save(todoFileName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *complete > 0:
		//complete given item
		if err := l.Complete(*complete); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		//save new list
		if err := l.Save(todoFileName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *add:
		//When any arguments (excluding flags) are provided they will be
		// used as new task
		t, err := getTask(os.Stdin, flag.Args()...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, v := range t {
			l.Add(v)
		}
		//Save the new list
		if err := l.Save(todoFileName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

	// concatenate all provided params with a space and add to the list as an item
	default:
		// Invalid flag provided
		fmt.Fprintln(os.Stderr, "Invalid option")
		flag.Usage()
		os.Exit(1)
	}
}
func getTask(r io.Reader, args ...string) ([]string, error) {
	var lines = []string{}
	if len(args) > 0 {
		return append(lines, strings.Join(args, " ")), nil
	}

	s := bufio.NewScanner(r)
	for s.Scan() {
		if err := s.Err(); err != nil {
			return append(lines, ""), err
		}
		if len(s.Text()) == 0 {
			return append(lines, ""), fmt.Errorf("Task cannot be blank")
		}
		lines = append(lines, s.Text())
	}
	return lines, nil
}
