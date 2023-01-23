package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	votes := readFile("votes.txt")
	ranks := compare(votes)
	for k, v := range ranks {
		fmt.Printf("%s has been voted %d times\n", k, v)
	}
	votes_alvaro, ok := ranks["Alvaro"]
	fmt.Printf("%d seems votes for Alvaro. But Alvaro is present? %b\n", votes_alvaro, ok)
}
func compare(votes []string) map[string]int {
	ranks := make(map[string]int)
	for _, v := range votes {
		ranks[v]++
	}
	return ranks

}
func readFile(filename string) []string {

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	var elements []string
	for scanner.Scan() {
		elements = append(elements, scanner.Text())
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
