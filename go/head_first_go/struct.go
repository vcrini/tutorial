package main

import "fmt"

type details struct {
	title   string
	name    string
	surname string
	age     int
}
type address struct {
	via    string
	number int
	cap    int
}
type full struct {
	details
	address
}

func main() {
	var e details
	e.name = "Valerio"
	e.surname = "Crini"
	e.age = 48
	fmt.Printf("person is %s %s with age %d\n", e.name, e.surname, e.age)
	var a details
	a.name = "Adele"
	a.surname = "Crini"
	a.age = 8
	fmt.Printf("person is %s %s with age %d\n", a.name, a.surname, a.age)
	var b full
	b.name = "Valerio"
	b.via = "Fra Paolo Sarpi"
	b = useTitleNoPointers(b)
	fmt.Printf("person %s is %s and lives in '%s'\n", b.details.title, b.details.name, b.address.via)
	c := useTitleWrong(b)
	fmt.Printf("person %s is %s and lives in '%s'\n", c.details.title, c.details.name, c.address.via)
	useTitlePointers(&b)
	fmt.Printf("person name is %s and lives in '%s'\n", b.details.name, b.address.via)
	fmt.Printf("person is %v\n", b)
}
func useTitleNoPointers(s full) full {
	s.title = "Mr."
	return s
}
func useTitleWrong(s full) *full {
	s.title = "Miss"
	return &s
}
func useTitlePointers(s *full) {
	s.title = "Mr."
}
