package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
)

const (
	defaultCounter = 3
	minCounter     = 0 // Il valore minimo
)

var dataFile = "pngs.json"

// PNG rappresenta la struttura dati per un PNG con il suo contatore.
type PNG struct {
	Name    string
	Counter int
}

func randomPNGName() string {
	adjectives := []string{
		"antico", "arcano", "celestiale", "crepuscolare", "dorato", "draconico",
		"incantato", "lunare", "mistico", "nobile", "ruggente", "sacro",
		"segreto", "tempestoso", "valente", "velato",
	}
	nouns := []string{
		"drago", "grifone", "fenice", "runa", "santuario", "torre", "reliquia",
		"spada", "scudo", "foresta", "regno", "oracolo", "ombra", "stella",
		"valle", "vento",
	}
	suffixes := []string{
		"al", "anor", "dellalba", "delcrepuscolo", "dor", "eld", "fir",
		"gorn", "ion", "kor", "lith", "mir", "nath", "rend", "thor", "vyr",
	}

	adj := capitalizeWord(adjectives[rand.IntN(len(adjectives))])
	noun := capitalizeWord(nouns[rand.IntN(len(nouns))])
	suffix := capitalizeWord(suffixes[rand.IntN(len(suffixes))])
	return fmt.Sprintf("%s %s %s", adj, noun, suffix)
}

func capitalizeWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func uniqueRandomPNGName(existing []PNG) string {
	seen := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		seen[p.Name] = struct{}{}
	}
	for {
		name := randomPNGName()
		if _, ok := seen[name]; !ok {
			return name
		}
	}
}

func loadPNGList(path string) ([]PNG, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PNG{}, "", nil
		}
		return nil, "", err
	}

	var wrapper struct {
		PNGs     []PNG  `json:"pngs"`
		Selected string `json:"selected"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.PNGs != nil {
		return wrapper.PNGs, wrapper.Selected, nil
	}

	var legacy []PNG
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, "", err
	}
	return legacy, "", nil
}

func savePNGList(path string, pngs []PNG, selected string) error {
	payload := struct {
		PNGs     []PNG  `json:"pngs"`
		Selected string `json:"selected"`
	}{
		PNGs:     pngs,
		Selected: selected,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func selectedPNGName(pngs []PNG, idx int) string {
	if idx < 0 || idx >= len(pngs) {
		return ""
	}
	return pngs[idx].Name
}
