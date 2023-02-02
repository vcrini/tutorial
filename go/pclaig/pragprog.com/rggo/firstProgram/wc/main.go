package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	// calling the count function to count number of words
	// received from the Standard Input and printing it out
	fmt.Println(count(os.Stdin))

}

func count(r io.Reader) int {
	// a scanner is used to read text from a Reader (such as file)
	scanner := bufio.NewScanner(r)
	// define the scanner split type to words since by default splits by lines
	scanner.Split(bufio.ScanWords)
	//defining a counter
	wc := 0
	// increase for every word scanned
	for scanner.Scan() {
		wc += 1
	}

	return wc

}
