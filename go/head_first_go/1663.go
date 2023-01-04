package main

import (
  "bufio"
  "fmt"
  "os"
)
func main() {
  fmt.Println("Inserisci un grado:")
  reader := bufio.NewReader(os.Stdin)
  input, _ := reader.ReadString('\n')
  fmt.Println(input)
}
