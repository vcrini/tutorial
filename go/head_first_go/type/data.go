package main
import (
  "fmt" 
  "log"
  "example.com/x/date"
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
  fmt.Printf("Date is %d/%d/%d\n",date.Year(), date.Month(), date.Day())
	err = date.SetMonth(14)
	if err != nil {
		log.Fatal(err)
	}
}
