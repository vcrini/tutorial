package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
)

func scanDirectory(path string) error {
	fmt.Println("entering ",path)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		if file.IsDir() {
			err := scanDirectory(filePath)
			if err != nil {
				return err
			}
		} else {
			fmt.Println(filePath," is a file")
		}
	}
	return nil
}
func reportPanic() {
  p:=recover()
  if p==nil {
    return
  }
  err, ok:=p.(error)
  if ok {
    fmt.Println("Managing panic ",err)
  } else {
    panic(p)
  }
}
func main() {
  defer reportPanic()
	err := scanDirectory(".")
	if err != nil {
		log.Fatal(err)
	}
}
