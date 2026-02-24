package main

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster        Monster
	Wounds         int
	WoundsMax      int
	InitiativeCard string
	HasInit        bool
	Conditions     map[string]int
	BasePF         int
	Stress         int
	BaseStress     int
}

type encounterConditionDef struct {
	Code string
	Name string
}

var encounterConditionDefs = []encounterConditionDef{
	{Code: "S", Name: "Scosso"},
	{Code: "T", Name: "Stordito"},
	{Code: "D", Name: "Distratto"},
	{Code: "V", Name: "Vulnerabile"},
	{Code: "H", Name: "Impedito"},
	{Code: "F", Name: "Affaticato"},
	{Code: "E", Name: "Intrappolato"},
	{Code: "B", Name: "Vincolato"},
}

type encounterConditionState struct {
	Code   string
	Rounds int
}

func cloneStringIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func orderedEncounterConditions(conditions map[string]int) []encounterConditionState {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]encounterConditionState, 0, len(conditions))
	seen := map[string]struct{}{}
	for _, d := range encounterConditionDefs {
		if n, ok := conditions[d.Code]; ok && n > 0 {
			out = append(out, encounterConditionState{Code: d.Code, Rounds: n})
			seen[d.Code] = struct{}{}
		}
	}
	extra := make([]string, 0, len(conditions))
	extraRounds := map[string]int{}
	for code, rounds := range conditions {
		norm := strings.ToUpper(strings.TrimSpace(code))
		if rounds <= 0 || norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		extra = append(extra, norm)
		extraRounds[norm] = rounds
	}
	sort.Strings(extra)
	for _, code := range extra {
		out = append(out, encounterConditionState{Code: code, Rounds: extraRounds[code]})
	}
	return out
}

func encounterConditionsBadge(entry EncounterEntry) string {
	if len(entry.Conditions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(entry.Conditions))
	for _, d := range encounterConditionDefs {
		if n, ok := entry.Conditions[d.Code]; ok && n > 0 {
			parts = append(parts, d.Code+strconv.Itoa(n))
		}
	}
	if len(parts) == 0 {
		keys := make([]string, 0, len(entry.Conditions))
		for k := range entry.Conditions {
			keys = append(keys, strings.ToUpper(k))
		}
		sort.Strings(keys)
		for _, k := range keys {
			if n := entry.Conditions[k]; n > 0 {
				parts = append(parts, k+strconv.Itoa(n))
			}
		}
	}
	return strings.Join(parts, "")
}

func encounterConditionsLong(entry EncounterEntry) string {
	ordered := orderedEncounterConditions(entry.Conditions)
	if len(ordered) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ordered))
	for _, p := range ordered {
		parts = append(parts, strings.ToUpper(strings.TrimSpace(p.Code))+strconv.Itoa(p.Rounds)+" "+conditionNameByCode(p.Code))
	}
	return strings.Join(parts, ", ")
}

func conditionNameByCode(code string) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	for _, d := range encounterConditionDefs {
		if d.Code == c {
			return d.Name
		}
	}
	if c == "P" {
		return "Prono"
	}
	return c
}

func conditionEffectByCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "S":
		return "Scosso: puo agire solo dopo essere recuperato (Spirit a inizio turno o spendendo Bennie)."
	case "T":
		return "Stordito: deve superare Vigore a inizio turno per riprendersi."
	case "D":
		return "Distratto: -2 a tutte le prove di Tratto."
	case "V":
		return "Vulnerabile: chi lo attacca ha +2 ai tiri di attacco."
	case "H":
		return "Impedito: Passo ridotto di 2 e dado di corsa ridotto di un tipo."
	case "F":
		return "Affaticato: -1 a prove di Tratto per livello (fino a 2), al terzo livello diventa Incapacitato."
	case "E":
		return "Intrappolato: non puo muoversi; bersaglio Vulnerabile; puo liberarsi con prova di Forza."
	case "B":
		return "Vincolato: come Intrappolato, ma piu severo; non puo agire se non tentare di liberarsi."
	case "P":
		return "Prono: -2 agli attacchi in mischia e ai tiri di tratto collegati al movimento; bersaglio con copertura contro attacchi a distanza."
	default:
		return ""
	}
}

func conditionDerivedCodes(code string) []string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "T":
		return []string{"V", "P"}
	default:
		return nil
	}
}

func encounterConditionEffectsLong(entry EncounterEntry) string {
	ordered := orderedEncounterConditions(entry.Conditions)
	if len(ordered) == 0 {
		return ""
	}

	lines := make([]string, 0, len(ordered)*2)
	seen := map[string]struct{}{}
	var appendEffect func(code string)
	appendEffect = func(code string) {
		code = strings.ToUpper(strings.TrimSpace(code))
		if code == "" {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		seen[code] = struct{}{}
		effect := conditionEffectByCode(code)
		if effect == "" {
			effect = conditionNameByCode(code) + ": effetto non codificato."
		}
		lines = append(lines, "- "+effect)
		for _, child := range conditionDerivedCodes(code) {
			appendEffect(child)
		}
	}

	for _, p := range ordered {
		appendEffect(p.Code)
	}
	return strings.Join(lines, "\n")
}

type encounterPersist struct {
	Entries []struct {
		Name             string         `yaml:"name"`
		Wounds           int            `yaml:"wounds"`
		PF               int            `yaml:"pf"`
		InitiativeCard   string         `yaml:"initiative_card,omitempty"`
		LegacyInitiative int            `yaml:"initiative,omitempty"`
		HasInit          bool           `yaml:"has_initiative,omitempty"`
		Conditions       map[string]int `yaml:"conditions,omitempty"`
		Stress           int            `yaml:"stress,omitempty"`
		BaseStress       int            `yaml:"base_stress,omitempty"`
	} `yaml:"entries"`
}

func (m *model) addMonsterToEncounter(mon Monster) {
	woundsMax := monsterWoundsMax(mon)
	m.encounter = append(m.encounter, EncounterEntry{
		Monster:    mon,
		WoundsMax:  woundsMax,
		BasePF:     woundsMax,
		Stress:     0,
		BaseStress: 0,
	})
	m.persistEncounter()
}

func (m *model) removeEncounterAt(idx int) {
	if idx < 0 || idx >= len(m.encounter) {
		return
	}
	m.encounter = append(m.encounter[:idx], m.encounter[idx+1:]...)
	if m.encounterCursor >= len(m.encounter) {
		m.encounterCursor = len(m.encounter) - 1
	}
	if m.encounterCursor < 0 {
		m.encounterCursor = 0
	}
	m.persistEncounter()
}

func (m *model) persistEncounter() {
	var entries []struct {
		Name             string         `yaml:"name"`
		Wounds           int            `yaml:"wounds"`
		PF               int            `yaml:"pf"`
		InitiativeCard   string         `yaml:"initiative_card,omitempty"`
		LegacyInitiative int            `yaml:"initiative,omitempty"`
		HasInit          bool           `yaml:"has_initiative,omitempty"`
		Conditions       map[string]int `yaml:"conditions,omitempty"`
		Stress           int            `yaml:"stress,omitempty"`
		BaseStress       int            `yaml:"base_stress,omitempty"`
	}
	for _, e := range m.encounter {
		basePF := encounterMaxWounds(e)
		entries = append(entries, struct {
			Name             string         `yaml:"name"`
			Wounds           int            `yaml:"wounds"`
			PF               int            `yaml:"pf"`
			InitiativeCard   string         `yaml:"initiative_card,omitempty"`
			LegacyInitiative int            `yaml:"initiative,omitempty"`
			HasInit          bool           `yaml:"has_initiative,omitempty"`
			Conditions       map[string]int `yaml:"conditions,omitempty"`
			Stress           int            `yaml:"stress,omitempty"`
			BaseStress       int            `yaml:"base_stress,omitempty"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: basePF, InitiativeCard: e.InitiativeCard, HasInit: e.HasInit, Conditions: cloneStringIntMap(e.Conditions)})
	}
	_ = saveEncounter(encounterFile, entries)
}

func loadEncounter(path string, monsters []Monster) ([]EncounterEntry, error) {
	rawEntries, err := readEncounter(path)
	if err != nil {
		return nil, err
	}
	if len(rawEntries) == 0 {
		return []EncounterEntry{}, nil
	}

	byName := make(map[string]Monster, len(monsters))
	for _, m := range monsters {
		byName[m.Name] = m
	}

	var entries []EncounterEntry
	for _, e := range rawEntries {
		name := e.Name
		if mon, ok := byName[name]; ok {
			maxWounds := e.PF
			if maxWounds <= 0 {
				maxWounds = monsterWoundsMax(mon)
			}
			if e.Wounds > maxWounds {
				e.Wounds = maxWounds
			}
			entries = append(entries, EncounterEntry{
				Monster:        mon,
				Wounds:         e.Wounds,
				WoundsMax:      maxWounds,
				InitiativeCard: e.InitiativeCard,
				HasInit:        e.HasInit && e.InitiativeCard != "",
				Conditions:     cloneStringIntMap(e.Conditions),
				BasePF:         maxWounds,
			})
		} else {
			maxWounds := e.PF
			if maxWounds <= 0 {
				maxWounds = 3
			}
			entries = append(entries, EncounterEntry{
				Monster:        Monster{Name: name, WoundsMax: maxWounds, PF: maxWounds},
				Wounds:         e.Wounds,
				WoundsMax:      maxWounds,
				InitiativeCard: e.InitiativeCard,
				HasInit:        e.HasInit && e.InitiativeCard != "",
				Conditions:     cloneStringIntMap(e.Conditions),
				BasePF:         maxWounds,
			})
		}
	}
	return entries, nil
}

func monsterWoundsMax(mon Monster) int {
	if mon.WoundsMax > 0 {
		return mon.WoundsMax
	}
	if mon.PF > 0 {
		return mon.PF
	}
	return 3
}

func encounterMaxWounds(e EncounterEntry) int {
	if e.WoundsMax > 0 {
		return e.WoundsMax
	}
	if e.BasePF > 0 {
		return e.BasePF
	}
	if e.Monster.WoundsMax > 0 {
		return e.Monster.WoundsMax
	}
	if e.Monster.PF > 0 {
		return e.Monster.PF
	}
	return 3
}

func saveEncounter(path string, entries []struct {
	Name             string         `yaml:"name"`
	Wounds           int            `yaml:"wounds"`
	PF               int            `yaml:"pf"`
	InitiativeCard   string         `yaml:"initiative_card,omitempty"`
	LegacyInitiative int            `yaml:"initiative,omitempty"`
	HasInit          bool           `yaml:"has_initiative,omitempty"`
	Conditions       map[string]int `yaml:"conditions,omitempty"`
	Stress           int            `yaml:"stress,omitempty"`
	BaseStress       int            `yaml:"base_stress,omitempty"`
}) error {
	payload := encounterPersist{Entries: entries}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readEncounter(path string) ([]struct {
	Name             string         `yaml:"name"`
	Wounds           int            `yaml:"wounds"`
	PF               int            `yaml:"pf"`
	InitiativeCard   string         `yaml:"initiative_card,omitempty"`
	LegacyInitiative int            `yaml:"initiative,omitempty"`
	HasInit          bool           `yaml:"has_initiative,omitempty"`
	Conditions       map[string]int `yaml:"conditions,omitempty"`
	Stress           int            `yaml:"stress,omitempty"`
	BaseStress       int            `yaml:"base_stress,omitempty"`
}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []struct {
				Name             string         `yaml:"name"`
				Wounds           int            `yaml:"wounds"`
				PF               int            `yaml:"pf"`
				InitiativeCard   string         `yaml:"initiative_card,omitempty"`
				LegacyInitiative int            `yaml:"initiative,omitempty"`
				HasInit          bool           `yaml:"has_initiative,omitempty"`
				Conditions       map[string]int `yaml:"conditions,omitempty"`
				Stress           int            `yaml:"stress,omitempty"`
				BaseStress       int            `yaml:"base_stress,omitempty"`
			}{}, nil
		}
		return nil, err
	}
	var payload encounterPersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Entries == nil {
		return []struct {
			Name             string         `yaml:"name"`
			Wounds           int            `yaml:"wounds"`
			PF               int            `yaml:"pf"`
			InitiativeCard   string         `yaml:"initiative_card,omitempty"`
			LegacyInitiative int            `yaml:"initiative,omitempty"`
			HasInit          bool           `yaml:"has_initiative,omitempty"`
			Conditions       map[string]int `yaml:"conditions,omitempty"`
			Stress           int            `yaml:"stress,omitempty"`
			BaseStress       int            `yaml:"base_stress,omitempty"`
		}{}, nil
	}
	return payload.Entries, nil
}
