package main

import (
	"embed"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
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
var fearStateFile = "state.yml"
var notesFile = "notes.yml"
var appStateDir = ""

//go:embed config/names.yaml config/mostri.yml config/ambienti.yml config/equipaggiamento.yaml config/carte.yaml config/classi.yaml
var embeddedConfigFS embed.FS

type nameLists struct {
	First []string `yaml:"first"`
	Last  []string `yaml:"last"`
}

var namesCache nameLists
var namesLoaded bool

func initStoragePaths() error {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return fmt.Errorf("impossibile risolvere HOME: %w", err)
	}
	appStateDir = filepath.Join(home, ".lazydaggerheart")
	if err := os.MkdirAll(appStateDir, 0o755); err != nil {
		return fmt.Errorf("impossibile creare dir stato %s: %w", appStateDir, err)
	}
	dataFile = filepath.Join(appStateDir, "pngs.yml")
	encounterFile = filepath.Join(appStateDir, "encounter.yml")
	fearStateFile = filepath.Join(appStateDir, "state.yml")
	notesFile = filepath.Join(appStateDir, "notes.yml")
	return nil
}

func readData(path string) ([]byte, error) {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if strings.HasPrefix(normalized, "config/") {
		data, err := embeddedConfigFS.ReadFile(normalized)
		if err == nil {
			return data, nil
		}
	}
	return os.ReadFile(path)
}

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
	Name        string `yaml:"name"`
	Token       int    `yaml:"token"`
	PF          int    `yaml:"pf,omitempty"`
	Stress      int    `yaml:"stress,omitempty"`
	ArmorScore  int    `yaml:"armor_score,omitempty"`
	Hope        int    `yaml:"hope,omitempty"`
	Class       string `yaml:"class,omitempty"`
	Subclass    string `yaml:"subclass,omitempty"`
	Level       int    `yaml:"level,omitempty"`
	Rank        int    `yaml:"rank,omitempty"`
	CompBonus   int    `yaml:"comp_bonus,omitempty"`
	ExpBonus    int    `yaml:"exp_bonus,omitempty"`
	Description string `yaml:"description,omitempty"`
	Traits      string `yaml:"traits,omitempty"`
	Primary     string `yaml:"primary,omitempty"`
	Secondary   string `yaml:"secondary,omitempty"`
	Armor       string `yaml:"armor,omitempty"`
	Look        string `yaml:"look,omitempty"`
	Inventory   string `yaml:"inventory,omitempty"`
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

	data, err := readData(namesFile)
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
	data, err := readData(path)
	if err != nil {
		return nil, err
	}

	var monsters []Monster
	if err := yaml.Unmarshal(data, &monsters); err != nil {
		return nil, err
	}
	for i := range monsters {
		monsters[i].Name = sanitizeMonsterText(monsters[i].Name)
		monsters[i].Role = sanitizeMonsterText(monsters[i].Role)
		monsters[i].Description = sanitizeMonsterText(monsters[i].Description)
		monsters[i].MotivationsTactics = sanitizeMonsterText(monsters[i].MotivationsTactics)
		monsters[i].Attack.Name = sanitizeMonsterText(monsters[i].Attack.Name)
		monsters[i].Attack.Range = sanitizeMonsterText(monsters[i].Attack.Range)
		monsters[i].Attack.Damage = sanitizeMonsterText(monsters[i].Attack.Damage)
		monsters[i].Attack.DamageType = sanitizeMonsterText(monsters[i].Attack.DamageType)
		for j := range monsters[i].Traits {
			monsters[i].Traits[j].Name = sanitizeMonsterText(monsters[i].Traits[j].Name)
			monsters[i].Traits[j].Kind = sanitizeMonsterText(monsters[i].Traits[j].Kind)
			monsters[i].Traits[j].Text = sanitizeMonsterText(monsters[i].Traits[j].Text)
		}
	}
	return monsters, nil
}

func sanitizeMonsterText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"Seguace(4)", "Seguace (4)",
		"AttaccoinMassa", "Attacco in Massa",
		"IlGrovigliovienesconfitto", "Il Groviglio viene sconfitto",
		"ilgrovigliovienesconfitto", "il Groviglio viene sconfitto",
		"SpendeteunaPaura", "Spendete una Paura",
		"spendeteunaPaura", "spendete una Paura",
		"MarcateunoStress", "Marcate uno Stress",
		"marcateunoStress", "marcate uno Stress",
		"dannifisici", "danni fisici",
		"dannimagici", "danni magici",
		" untiro ", " un tiro ",
	)
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

type fearPersist struct {
	Paure int `yaml:"paure"`
}

func loadFearState(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var st fearPersist
	if err := yaml.Unmarshal(data, &st); err != nil {
		return 0, err
	}
	return clampFear(st.Paure), nil
}

func saveFearState(path string, paure int) error {
	payload := fearPersist{Paure: clampFear(paure)}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func clampFear(v int) int {
	if v < 0 {
		return 0
	}
	if v > 12 {
		return 12
	}
	return v
}

type notesPersist struct {
	Notes []string `yaml:"notes"`
}

func loadNotes(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload notesPersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Notes == nil {
		return []string{}, nil
	}
	return payload.Notes, nil
}

func saveNotes(path string, notes []string) error {
	payload := notesPersist{Notes: notes}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadEnvironments(path string) ([]Environment, error) {
	data, err := readData(path)
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
	data, err := readData(path)
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
	data, err := readData(path)
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
	data, err := readData(path)
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
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, "", err
	}
	return wrapper.PNGs, wrapper.Selected, nil
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
