package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
)

func main() {

	// just to show how to use arrays
	var element [3]int
	element[0] = 2
	for i := 0; i < 3; i++ {
		fmt.Printf("il %d numero è  %d\n", i, element[i])
	}
	element2 := [3]float64{1.0, 2.0, 3.0}
	//calculating media
	sum := 0.0
	for _, v := range element2 {
		sum += v
	}
	n := float64(len(element2))

	fmt.Printf("la media di %v è %0.2f\n", element2, sum/n)
	// reading from file
	fmt.Println("il contenuto del file è")
	file, err := os.Open("array.txt")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	var i int
	var elements []float64
	for scanner.Scan() {
    var elem float64
		fmt.Println(scanner.Text())
		elem, err = strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			log.Fatal(err)
		}
    elements=append(elements,elem)
		i += 1
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	if scanner.Err() != nil {
		log.Fatal(err)
	}
	fmt.Println("La media dei valori contenuti nel file è")
	sum = 0
	for _, v := range elements {
		sum += v
	}
	n = float64(len(elements))

	fmt.Printf("la media di %v è %0.2f\n", elements, sum/n)
}