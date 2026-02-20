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

var dataFile = "pngs.yml"
var namesFile = "config/names.yaml"
var monstersFile = "config/mostri.yml"
var environmentsFile = "config/ambienti.yml"
var equipmentFile = "config/equipaggiamento.yaml"
var cardsFile = "config/carte.yaml"
var classesFile = "config/classi.yaml"
var encounterFile = "encounter.yml"

type nameLists struct {
	First []string `yaml:"first"`
	Last  []string `yaml:"last"`
}

var namesCache nameLists
var namesLoaded bool

type Thresholds struct {
	Values []int
	Text   string
}

func (t *Thresholds) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		var vals []int
		for i := 0; i < len(value.Content); i++ {
			var v int
			if err := value.Content[i].Decode(&v); err != nil {
				return err
			}
			vals = append(vals, v)
		}
		t.Values = vals
		t.Text = ""
		return nil
	case yaml.ScalarNode:
		t.Text = value.Value
		t.Values = nil
		return nil
	default:
		return nil
	}
}

type Monster struct {
	Name               string     `yaml:"name"`
	Role               string     `yaml:"role"`
	Rank               int        `yaml:"rank"`
	Description        string     `yaml:"description"`
	MotivationsTactics string     `yaml:"motivations_tactics"`
	Difficulty         int        `yaml:"difficulty"`
	Thresholds         Thresholds `yaml:"thresholds"`
	PF                 int        `yaml:"pf"`
	Stress             int        `yaml:"stress"`
	Attack             struct {
		Bonus      string `yaml:"bonus"`
		Name       string `yaml:"name"`
		Range      string `yaml:"range"`
		Damage     string `yaml:"damage"`
		DamageType string `yaml:"damage_type"`
	} `yaml:"attack"`
	Traits []struct {
		Name string `yaml:"name"`
		Kind string `yaml:"kind"`
		Text string `yaml:"text"`
	} `yaml:"traits"`
}

type Environment struct {
	Name                 string `yaml:"name"`
	Kind                 string `yaml:"kind"`
	Rank                 int    `yaml:"rank"`
	Description          string `yaml:"description"`
	Impeti               string `yaml:"impeti"`
	Difficulty           string `yaml:"difficulty"`
	PotentialAdversaries string `yaml:"potential_adversaries"`
	Characteristics      []struct {
		Name string `yaml:"name"`
		Kind string `yaml:"kind"`
		Text string `yaml:"text"`
	} `yaml:"characteristics"`
}

type EquipmentItem struct {
	Name           string `yaml:"name"`
	Category       string `yaml:"category"`
	Type           string `yaml:"type"`
	Rank           int    `yaml:"rank"`
	Levels         string `yaml:"levels"`
	Trait          string `yaml:"trait"`
	Range          string `yaml:"range"`
	Damage         string `yaml:"damage"`
	Grip           string `yaml:"grip"`
	Characteristic string `yaml:"characteristic"`
}

type CardItem struct {
	Name        string   `yaml:"name"`
	Class       string   `yaml:"class"`
	Type        string   `yaml:"type"`
	CasterTrait string   `yaml:"caster_trait"`
	Description string   `yaml:"description"`
	Effects     []string `yaml:"effects"`
}

type ClassItem struct {
	Name            string   `yaml:"name"`
	Subclass        string   `yaml:"subclass"`
	Rank            int      `yaml:"rank"`
	Domains         string   `yaml:"domains"`
	Evasion         int      `yaml:"evasion"`
	HP              int      `yaml:"hp"`
	ClassItem       string   `yaml:"class_item"`
	HopePrivilege   string   `yaml:"hope_privilege"`
	ClassPrivileges []string `yaml:"class_privileges"`
	Description     string   `yaml:"description"`
	CasterTrait     string   `yaml:"caster_trait"`
	BasePrivileges  []string `yaml:"base_privileges"`
	Specialization  string   `yaml:"specialization"`
	Mastery         string   `yaml:"mastery"`
	BackgroundQs    []string `yaml:"background_questions"`
	Bonds           []string `yaml:"bonds"`
}

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

func loadMonsters(path string) ([]Monster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var monsters []Monster
	if err := yaml.Unmarshal(data, &monsters); err != nil {
		return nil, err
	}
	return monsters, nil
}

func loadEnvironments(path string) ([]Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var environments []Environment
	if err := yaml.Unmarshal(data, &environments); err != nil {
		return nil, err
	}
	return environments, nil
}

func loadEquipment(path string) ([]EquipmentItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var items []EquipmentItem
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func loadCards(path string) ([]CardItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cards []CardItem
	if err := yaml.Unmarshal(data, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

func loadClasses(path string) ([]ClassItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var classes []ClassItem
	if err := yaml.Unmarshal(data, &classes); err != nil {
		return nil, err
	}
	return classes, nil
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
		PNGs     []PNG  `yaml:"pngs"`
		Selected string `yaml:"selected"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err == nil && wrapper.PNGs != nil {
		return wrapper.PNGs, wrapper.Selected, nil
	}

	// Legacy JSON support
	var legacyWrapper struct {
		PNGs     []PNG  `json:"pngs"`
		Selected string `json:"selected"`
	}
	if err := json.Unmarshal(data, &legacyWrapper); err == nil && legacyWrapper.PNGs != nil {
		return legacyWrapper.PNGs, legacyWrapper.Selected, nil
	}

	var legacy []PNG
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, "", err
	}
	return legacy, "", nil
}

func savePNGList(path string, pngs []PNG, selected string) error {
	payload := struct {
		PNGs     []PNG  `yaml:"pngs"`
		Selected string `yaml:"selected"`
	}{
		PNGs:     pngs,
		Selected: selected,
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func selectedPNGName(pngs []PNG, idx int) string {
	if idx < 0 || idx >= len(pngs) {
		return ""
	}
	return pngs[idx].Name
}
