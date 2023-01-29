package main

import "fmt"


func main() {
  defer fmt.Println("Finito")
  fmt.Println("Iniziato")
}
