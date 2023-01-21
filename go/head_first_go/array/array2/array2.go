package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"github.com/vcrini/x/go/array/media"
)

func main() {
	arguments := os.Args[1:]
	var elements []float64
	for _, v := range arguments {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			log.Fatal(err)
			elements = nil
		}
		elements = append(elements, value)
	}
	fmt.Printf("la media dei valori di %v è %0.2f\n", elements, media.Media(elements))
}
