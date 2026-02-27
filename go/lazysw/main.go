package main

import (
	"fmt"
	"os"
)

func main() {
	// Modalità CLI/headless (es. per ambienti senza terminale addressable)
	if len(os.Args) > 1 && (os.Args[1] == "cli" || os.Args[1] == "--cli") {
		if err := runCLI(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Errore modalità CLI: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Modalità TUI (predefinita)
	if err := runTViewUI(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
