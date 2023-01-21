//operation on file
package file

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
)

//reads file from disk called 'array.txt and extracts values in an array
func ReadFile() []float64 {

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
		elements = append(elements, elem)
		i += 1
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	if scanner.Err() != nil {
		log.Fatal(err)
	}
	return elements
}
