package main

import (
	"encoding/json"
	"math/rand/v2"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultToken = 3
	minToken     = 0 // Il valore minimo
	maxToken     = 3 // Il valore massimo
)

var dataFile = "pngs.json"
var namesFile = "config/names.yaml"

type nameLists struct {
	First []string `yaml:"first"`
	Last  []string `yaml:"last"`
}

var namesCache nameLists
var namesLoaded bool

// PNG rappresenta la struttura dati per un PNG con il suo token.
type PNG struct {
	Name  string `json:"Name"`
	Token int    `json:"Token"`
}

func (p *PNG) UnmarshalJSON(data []byte) error {
	var aux struct {
		Name         string `json:"Name"`
		Token        *int   `json:"Token"`
		Counter      *int   `json:"Counter"`
		TokenLower   *int   `json:"token"`
		CounterLower *int   `json:"counter"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.Name = aux.Name
	switch {
	case aux.Token != nil:
		p.Token = *aux.Token
	case aux.TokenLower != nil:
		p.Token = *aux.TokenLower
	case aux.Counter != nil:
		p.Token = *aux.Counter
	case aux.CounterLower != nil:
		p.Token = *aux.CounterLower
	default:
		p.Token = 0
	}
	return nil
}

func (p PNG) MarshalJSON() ([]byte, error) {
	out := struct {
		Name  string `json:"Name"`
		Token int    `json:"Token"`
	}{
		Name:  p.Name,
		Token: p.Token,
	}
	return json.Marshal(out)
}

func randomPNGName() string {
	first, last := loadNameLists()
	if len(last) == 0 {
		return first[rand.IntN(len(first))]
	}
	return first[rand.IntN(len(first))] + " " + last[rand.IntN(len(last))]
}

func capitalizeWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func loadNameLists() ([]string, []string) {
	if namesLoaded {
		if len(namesCache.First) > 0 {
			return namesCache.First, namesCache.Last
		}
		return []string{"Unknown"}, nil
	}
	namesLoaded = true

	data, err := os.ReadFile(namesFile)
	if err != nil {
		namesCache = defaultNameLists()
		return namesCache.First, namesCache.Last
	}

	var names nameLists
	if err := yaml.Unmarshal(data, &names); err != nil {
		namesCache = defaultNameLists()
		return namesCache.First, namesCache.Last
	}
	for _, name := range names.First {
		name = strings.TrimSpace(name)
		if name != "" {
			namesCache.First = append(namesCache.First, name)
		}
	}
	for _, name := range names.Last {
		name = strings.TrimSpace(name)
		if name != "" {
			namesCache.Last = append(namesCache.Last, name)
		}
	}
	if len(namesCache.First) == 0 {
		namesCache = defaultNameLists()
	}
	return namesCache.First, namesCache.Last
}

func defaultNameLists() nameLists {
	return nameLists{
		First: []string{
			"Alucard", "Ambrose", "Ash", "Bellamy", "Calder",
			"Calypso", "Chartreuse", "Clover", "Dahlia",
			"Darrow", "Deacon", "Elowen", "Emrys", "Fable",
			"Fiorella", "Flynn", "Gatlin", "Gerard", "Hadron",
			"Harlow", "Indigo", "Isla", "Jaden", "Kai", "Kismet",
			"Leo", "Mika", "Moon", "Nyx", "Orna", "Phaedra",
			"Quill", "Rani", "Raphael", "Reza", "Roux", "Saffron",
			"Sierra", "Skye", "Talon", "Thea", "Triton", "Vala",
			"Velo", "Wisteria", "Yanelle", "Zahara",
		},
		Last: []string{
			"Abbot", "Advani", "Agoston", "Baptiste", "Belgarde",
			"Blossom", "Chance", "Covault", "Dawn", "Dennison",
			"Drayer", "Emrick", "Foley", "Fury", "Grove",
			"Hartley", "Humfleet", "Hyland", "Ikeda", "Jones",
			"Jordon", "Kaan", "Knoth", "Lagrange", "Lockamy",
			"Lyon", "Marche", "Merrell", "Newland", "Novak",
			"Orwick", "Overholt", "Pray", "Rathbone", "Rose",
			"Seagrave", "Spurlock", "Thorn", "Tringle", "Vasquez",
			"Warren", "Worth", "York",
		},
	}
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
