package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
  votes:=readFile("votes.txt")
  ranks:=compare(votes)
  fmt.Printf("elections result: %v", ranks)


}
func compare(votes []string) map[string]int {
  ranks:=make(map[string]int)
  for _,v:=range votes {
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
