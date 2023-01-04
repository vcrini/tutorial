package main

import (
  "bufio"
  "fmt"
  "log"
  "os"
)
func main() {
  fmt.Println("Inserisci un voto 0-100:")
  reader := bufio.NewReader(os.Stdin)
  input, err  := reader.ReadString('\n')
  if err != nil {
    log.Fatal(err)
  }
  var  status string
  if input >= 60 {
    status = "passato"
  } else {
    status = "bocciato"
  }
  fmt.Println(status)
}
