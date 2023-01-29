package main

import (
  "fmt"
  "example.com/go/utils/prose"
)

func main() {
  frasi:=[]string{"i miei genitori", "un clown da rodeo"}
  fmt.Println(prose.JoinWithCommas(frasi))
  frasi =[]string{"i miei genitori", "un clown da rodeo", "un toro scatenato"}
  fmt.Println(prose.JoinWithCommas(frasi))
}
