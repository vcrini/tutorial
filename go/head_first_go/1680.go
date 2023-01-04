package main

import (
  "bufio"
  "fmt"
  "log"
  "os"
)
func main() {
  fmt.Println("Inserisci un grado:")
  reader := bufio.NewReader(os.Stdin)
  input, err  := reader.ReadString('\n')
  log.Fatal(err)
  fmt.Println(input)
}
