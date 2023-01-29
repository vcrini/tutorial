package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vcrini/x/go/array/file"
)

func main() {
	numbers, err := file.GetFloats(os.Args[1])
  if err!=nil {
    log.Fatal(err)
  }
  var sum float64=0
  for _, number := range numbers {
    sum+=number
  }
  fmt.Println("La somma totale è ", sum)
}
