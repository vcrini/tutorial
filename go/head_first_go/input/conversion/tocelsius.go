package main

import (
	"fmt"
	"log"
  "esempi.com/packages/flutil"
)


func main() {
	fmt.Print("Inserisci la temperatura fahrenheit: ")
	fahrenheit, err := flutil.GetFloat()
	if err != nil {
		log.Fatal(err)
	}
  const base = 32
	celsius := (fahrenheit - base) * 5
	fmt.Printf("%0.2f gradi celsius\n", celsius)
}
