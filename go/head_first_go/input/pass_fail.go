package main

import (
  "esempi.com/packages/flutil"
	"fmt"
	"log"
)


func main() {
	fmt.Print("Inserisci un valore: ")
	grade, err := flutil.GetFloat()
	if err != nil {
		log.Fatal(err)
	}
	var status string
	if grade >= 60 {
		status = "passing"
	} else {
		status = "falling"
	}
	fmt.Println("Un voto di ", grade, "è ", status)
}
