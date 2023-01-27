package date

import (
	"errors"
	"fmt"
)

type Date struct {
	year  int
	month int
	day   int
}

type Note struct {
	text string
}

func (d *Date) SetMonth(month int) error {
	if month > 12 || month < 1 {
		return errors.New(fmt.Sprintf("Invalid month: %d", month))
	}
	d.month = month
	return nil
}
func (d *Date) SetYear(year int) error {
	if year < 1 {
		return errors.New(fmt.Sprintf("Invalid year: %d", year))
	}
	d.year = year
	return nil
}
func (d *Date) SetDay(day int) error {
	if day < 1 || day > 31 {
		return errors.New(fmt.Sprintf("Invalid day: %d", day))
	}
	d.day = day
	return nil
}
func (n *Note) SetText(text string) error {
	if text == "sveglia" {
		return errors.New(fmt.Sprintf("Invalid text: %s", text))
	}
	n.text = text
	return nil
}

// pointers for consistency
func (d *Date) Month() int {
	return d.month
}
func (d *Date) Year() int {
	return d.year
}
func (d *Date) Day() int {
	return d.day
}
func (n *Note) Text() string {
	return n.text
}
