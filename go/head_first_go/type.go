package main

import "fmt"

type Liters float64
type Gallons float64
type Milliliters float64

func main() {
	var l Liters = 10.0
	var g Gallons = 10.0
	l.sayHi()
	var m1, m2 Milliliters
	m1 = l.toMilliliters()
	m2 = g.toMilliliters()
	fmt.Printf("%0.2f liters are %0.2f milliliters\n", l, m1)
	fmt.Printf("%0.2f gallons are %0.2f milliliters\n", g, m2)
}
func (l Liters) sayHi() {
	fmt.Println("Ciao da ", l)
}
func (l *Liters) toMilliliters() Milliliters {
	*l = *l * 1000
	return Milliliters(*l)
}
func (g *Gallons) toMilliliters() Milliliters {
	*g = *g * 1000 * 3.78541
	return Milliliters(*g)
}
