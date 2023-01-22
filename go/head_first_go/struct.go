package main

import "fmt"

func main() {
	var e struct {
		name    string
		surname string
		age     int
	}
	e.name = "Valerio"
	e.surname = "Crini"
	e.age = 48
	fmt.Printf("person is %s %s with age %d", e.name, e.surname, e.age)
}
