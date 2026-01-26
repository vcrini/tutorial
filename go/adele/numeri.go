package main

import (
	"math/rand/v2"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {
	p := message.NewPrinter(language.Italian)
	p.Printf("Leggi i seguenti numeri\n\n")
	for range 3 {
		for range 8 {
			p.Printf("%d ", rand.IntN(999999)+1)
		}
		p.Printf("\n")
	}
	p.Printf("\n")
	p.Printf("Scrivi in lettere\n\n")
	for range 4 {
		p.Println(rand.IntN(999999) + 1)
	}
	p.Printf("\n")
	p.Printf("Riordina dal più grande al più piccolo\n\n")
	for range 8 {
		for range 4 {
			p.Printf("%d ", rand.IntN(999999)+1)
		}
		p.Printf("\n")
	}
}
