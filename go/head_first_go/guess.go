package main

import (
  "fmt"
  "math/rand"
  "time"
)
func main() {
  seconds := time.Now().Unix()
  rand.Seed(seconds)
  target := rand.Intn(100)+1
  fmt.Println(target)
}

