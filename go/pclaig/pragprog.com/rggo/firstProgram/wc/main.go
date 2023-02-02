package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	//using a boolean flag -l to count lines, not words
	lines := flag.Bool("l", false, "Conta le linee")
	bytes := flag.Bool("b", false, "Conta i bytes")
	flag.Parse()
	// calling the count function to count number of words
	// received from the Standard Input and printing it out
	fmt.Println(count(os.Stdin, *lines, *bytes))

}

func count(r io.Reader, countLines bool, countBytes bool) int {
	// a scanner is used to read text from a Reader (such as file)
	scanner := bufio.NewScanner(r)
	// define the scanner split type to words if count lines flag is not set
	if !countLines && !countBytes {
		scanner.Split(bufio.ScanWords)
	} else if countBytes {
		scanner.Split(bufio.ScanBytes)

	}
	//defining a counter
	wc := 0
	// increase for every word scanned
	for scanner.Scan() {
		wc += 1
	}

	return wc

}
