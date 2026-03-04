package main

import (
	"fmt"
	"sort"
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
	fmt.Println("  lazysw cli monsters [filtro] [--source core,iz]   # filtra per nome e source")
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

	nameTerms := make([]string, 0, len(args))
	sourceSet := map[string]struct{}{}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		switch {
		case strings.HasPrefix(arg, "--source="):
			addSourcesToSet(sourceSet, strings.TrimPrefix(arg, "--source="))
		case strings.HasPrefix(arg, "-s="):
			addSourcesToSet(sourceSet, strings.TrimPrefix(arg, "-s="))
		case arg == "--source" || arg == "-s":
			if i+1 >= len(args) {
				return fmt.Errorf("opzione %s richiede un valore, es: %s core,iz", arg, arg)
			}
			i++
			addSourcesToSet(sourceSet, args[i])
		default:
			nameTerms = append(nameTerms, arg)
		}
	}
	filter := strings.ToLower(strings.TrimSpace(strings.Join(nameTerms, " ")))

	count := 0
	for _, m := range monsters {
		if filter != "" && !strings.Contains(strings.ToLower(m.Name), filter) {
			continue
		}
		source := strings.ToLower(strings.TrimSpace(m.Source))
		if source == "" {
			source = "core"
		}
		if len(sourceSet) > 0 {
			if _, ok := sourceSet[source]; !ok {
				continue
			}
		}
		wc := "no"
		if m.WildCard {
			wc = "si"
		}
		fmt.Printf("- %s (Ruolo: %s, Source: %s, Wild Card: %s, Taglia: %d, Rank: %d, Ferite max: %d)\n",
			m.Name, m.Role, source, wc, m.Size, m.Rank, m.WoundsMax)
		count++
	}
	if count == 0 {
		if len(sourceSet) > 0 {
			sources := make([]string, 0, len(sourceSet))
			for s := range sourceSet {
				sources = append(sources, s)
			}
			sort.Strings(sources)
			fmt.Printf("(nessun mostro trovato; nome=%q, source=%s)\n", filter, strings.Join(sources, ","))
		} else {
			fmt.Printf("(nessun mostro trovato; nome=%q)\n", filter)
		}
	}

	return nil
}

func addSourcesToSet(dest map[string]struct{}, raw string) {
	for _, part := range strings.Split(raw, ",") {
		s := strings.ToLower(strings.TrimSpace(part))
		if s == "" {
			continue
		}
		dest[s] = struct{}{}
	}
}
