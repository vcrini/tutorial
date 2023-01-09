// guess challenge
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
	success := false
	retries := 3
	for guesses := 0; guesses < retries; guesses++ {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Fai un tentativo di indovinare disponibili(", retries-guesses, " di ", retries, ")")
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
			success = true
			break
		}

	}
	if success {
		fmt.Println("Complimenti!")

	} else {
		fmt.Println("Spiacente non hai indovinato il numero era", target, ", ritenta")

	}
}
