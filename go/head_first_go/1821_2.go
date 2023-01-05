package main

import (
  "bufio"
  "fmt"
  "log"
  "os"
  "strconv"
  "strings"
)
func main() {
  fmt.Println("Inserisci un voto 0-100:")
  reader := bufio.NewReader(os.Stdin)
  input, err  := reader.ReadString('\n')
  input = strings.TrimSpace(input)
  if err != nil {
    log.Fatal(err)
  }
  grade, errg := strconv.ParseFloat(input, 64)
  if errg != nil {
    log.Fatal(err)
  }
  var  status string
  if grade >= 60 {
    status = "passato"
  } else {
    status = "bocciato"
  }
  fmt.Println(status)
}
