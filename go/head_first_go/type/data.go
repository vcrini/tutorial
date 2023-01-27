package main

import (
	"example.com/x/date"
	"fmt"
	"log"
)

func main() {
	var date1 date.Date
	fmt.Println(date1)
	err := date1.SetMonth(3)
	if err != nil {
		log.Fatal(err)
	}
	err = date1.SetDay(20)
	if err != nil {
		log.Fatal(err)
	}
	err = date1.SetYear(2008)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(date1)
	fmt.Printf("Date is %d/%d/%d\n", date1.Year(), date1.Month(), date1.Day())
	var note date.Note
	note.SetText("Nota 1")
	note.SetMonth(12)
	note.SetDay(10)
	note.SetYear(2014)
	fmt.Printf("Nota: %s del %d/%d/%d\n", note.Text(), note.Year(), note.Month(), note.Day())
	err = date1.SetMonth(14)
	if err != nil {
		log.Fatal(err)
	}
}
