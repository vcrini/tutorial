package main

import . "fmt"

const ciao = "Ciao"

var mondo string = "Mondo"

func main() {
	mondo := mondo + "!"
	Printf("%v %v\n", ciao, mondo)
}
