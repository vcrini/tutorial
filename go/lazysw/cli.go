package main

import (
	"fmt"
	"strings"
)

// runCLI esegue la modalità headless/CLI.
// Esempi:
//
//	lazysw cli dice "2d6+1"
//	lazysw cli monsters
//	lazysw cli monsters drago
func runCLI(args []string) error {
	if len(args) == 0 {
		printCLIUsage()
		return nil
	}

	switch args[0] {
	case "dice", "dadi":
		return cliDice(args[1:])
	case "monsters", "mostri":
		return cliMonsters(args[1:])
	default:
		printCLIUsage()
		return fmt.Errorf("comando CLI sconosciuto: %s", args[0])
	}
}

func printCLIUsage() {
	fmt.Println("LazySW modalità CLI")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  lazysw cli dice \"espressione\"      # tiro di dadi headless")
	fmt.Println("  lazysw cli monsters [filtro]        # elenca mostri, opzionale filtro per nome")
	fmt.Println()
}

func cliDice(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("specificare un'espressione di dadi, es: \"2d6+1\" oppure \"D6\"")
	}
	expr := strings.Join(args, " ")
	total, breakdown, err := rollDiceExpression(expr)
	if err != nil {
		return err
	}
	fmt.Printf("Espressione: %s\n", expr)
	fmt.Printf("Totale:     %d\n", total)
	fmt.Println("Dettaglio:")
	fmt.Println(breakdown)
	return nil
}

func cliMonsters(args []string) error {
	monsters, err := loadMonsters(monstersFile)
	if err != nil {
		return fmt.Errorf("errore caricando i mostri da %s: %w", monstersFile, err)
	}

	filter := strings.ToLower(strings.TrimSpace(strings.Join(args, " ")))
	for _, m := range monsters {
		if filter != "" && !strings.Contains(strings.ToLower(m.Name), filter) {
			continue
		}
		wc := "no"
		if m.WildCard {
			wc = "si"
		}
		fmt.Printf("- %s (Ruolo: %s, Wild Card: %s, Taglia: %d, Rank: %d, Ferite max: %d)\n",
			m.Name, m.Role, wc, m.Size, m.Rank, m.WoundsMax)
	}

	return nil
}
