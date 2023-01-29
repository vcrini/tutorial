// operation on file
package file

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
)

// reads file from disk called 'array.txt and extracts values in an array
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
	defer file.Close()
	if scanner.Err() != nil {
		log.Fatal(err)
	}
	return elements
}

func OpenFile(fileName string) (*os.File, error) {
	fmt.Printf("Opening file '%s'\n", fileName)
	file, err := os.Open(fileName)
	return file, err
}
func CloseFile(file *os.File) {
	fmt.Printf("Closing file %v\n", file)
	file.Close()
}

func GetFloats(fileName string) ([]float64, error) {
	var numbers []float64
	file, err := OpenFile(fileName)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		number, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, number)
	}
	CloseFile(file)
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return numbers, nil

}
