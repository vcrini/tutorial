package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster    Monster
	Wounds     int
	WoundsMax  int
	BasePF     int
	Stress     int
	BaseStress int
}

type encounterPersist struct {
	Entries []struct {
		Name       string `yaml:"name"`
		Wounds     int    `yaml:"wounds"`
		PF         int    `yaml:"pf"`
		Stress     int    `yaml:"stress,omitempty"`
		BaseStress int    `yaml:"base_stress,omitempty"`
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
		Name       string `yaml:"name"`
		Wounds     int    `yaml:"wounds"`
		PF         int    `yaml:"pf"`
		Stress     int    `yaml:"stress,omitempty"`
		BaseStress int    `yaml:"base_stress,omitempty"`
	}
	for _, e := range m.encounter {
		basePF := encounterMaxWounds(e)
		entries = append(entries, struct {
			Name       string `yaml:"name"`
			Wounds     int    `yaml:"wounds"`
			PF         int    `yaml:"pf"`
			Stress     int    `yaml:"stress,omitempty"`
			BaseStress int    `yaml:"base_stress,omitempty"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: basePF})
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
			entries = append(entries, EncounterEntry{Monster: mon, Wounds: e.Wounds, WoundsMax: maxWounds, BasePF: maxWounds})
		} else {
			maxWounds := e.PF
			if maxWounds <= 0 {
				maxWounds = 3
			}
			entries = append(entries, EncounterEntry{
				Monster:   Monster{Name: name, WoundsMax: maxWounds, PF: maxWounds},
				Wounds:    e.Wounds,
				WoundsMax: maxWounds,
				BasePF:    maxWounds,
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
	Name       string `yaml:"name"`
	Wounds     int    `yaml:"wounds"`
	PF         int    `yaml:"pf"`
	Stress     int    `yaml:"stress,omitempty"`
	BaseStress int    `yaml:"base_stress,omitempty"`
}) error {
	payload := encounterPersist{Entries: entries}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readEncounter(path string) ([]struct {
	Name       string `yaml:"name"`
	Wounds     int    `yaml:"wounds"`
	PF         int    `yaml:"pf"`
	Stress     int    `yaml:"stress,omitempty"`
	BaseStress int    `yaml:"base_stress,omitempty"`
}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []struct {
				Name       string `yaml:"name"`
				Wounds     int    `yaml:"wounds"`
				PF         int    `yaml:"pf"`
				Stress     int    `yaml:"stress,omitempty"`
				BaseStress int    `yaml:"base_stress,omitempty"`
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
			Name       string `yaml:"name"`
			Wounds     int    `yaml:"wounds"`
			PF         int    `yaml:"pf"`
			Stress     int    `yaml:"stress,omitempty"`
			BaseStress int    `yaml:"base_stress,omitempty"`
		}{}, nil
	}
	return payload.Entries, nil
}
