package main

import (
	"example.com/x/date"
	"fmt"
	"log"
)

func main() {
	var date date.Date
	fmt.Println(date)
	err := date.SetMonth(3)
	if err != nil {
		log.Fatal(err)
	}
	err = date.SetDay(20)
	if err != nil {
		log.Fatal(err)
	}
	err = date.SetYear(2008)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(date)
	fmt.Printf("Date is %d/%d/%d\n", date.Year(), date.Month(), date.Day())
	err = date.SetMonth(14)
	if err != nil {
		log.Fatal(err)
	}
  var n = date.Note
	// note.SetText("Nota 1")
	// note.SetMonth(12)
	// note.SetDay(10)
	// note.SetYear(2014)
	// fmt.Printf("Nota: %s del %d/%d/%d", note.Text(), note.Year(), note.Month(), note.Day())
}
