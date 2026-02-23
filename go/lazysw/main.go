package main

import (
	"fmt"
	"os"
)

func main() {
	if err := runTViewUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
