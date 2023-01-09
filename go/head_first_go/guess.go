package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	seconds := time.Now().Unix()
	rand.Seed(seconds)
	target := rand.Intn(100) + 1
	fmt.Println(target)

	for guesses := 0; guesses < 3; guesses++ {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Fai un tentativo di indovinare")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input)
		guess, err := strconv.Atoi(input)
		if err != nil {
			log.Fatal(err)
		}
		if guess > target {
			fmt.Println("Alto")
		}
		if guess < target {
			fmt.Println("Basso")
		}
		if guess == target {
			fmt.Println("Indovinato")
      break
		}

	}
}
