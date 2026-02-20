package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster Monster
	Wounds  int
	BasePF  int
}

type encounterPersist struct {
	Entries []struct {
		Name   string `yaml:"name"`
		Wounds int    `yaml:"wounds"`
		PF     int    `yaml:"pf"`
	} `yaml:"entries"`
}

func (m *model) addMonsterToEncounter(mon Monster) {
	m.encounter = append(m.encounter, EncounterEntry{Monster: mon, BasePF: mon.PF})
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
		Name   string `yaml:"name"`
		Wounds int    `yaml:"wounds"`
		PF     int    `yaml:"pf"`
	}
	for _, e := range m.encounter {
		basePF := e.BasePF
		if basePF == 0 {
			basePF = e.Monster.PF
		}
		entries = append(entries, struct {
			Name   string `yaml:"name"`
			Wounds int    `yaml:"wounds"`
			PF     int    `yaml:"pf"`
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
			entries = append(entries, EncounterEntry{Monster: mon, Wounds: e.Wounds, BasePF: e.PF})
		} else {
			entries = append(entries, EncounterEntry{Monster: Monster{Name: name, PF: e.PF}, Wounds: e.Wounds, BasePF: e.PF})
		}
	}
	return entries, nil
}

func saveEncounter(path string, entries []struct {
	Name   string `yaml:"name"`
	Wounds int    `yaml:"wounds"`
	PF     int    `yaml:"pf"`
}) error {
	payload := encounterPersist{Entries: entries}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readEncounter(path string) ([]struct {
	Name   string `yaml:"name"`
	Wounds int    `yaml:"wounds"`
	PF     int    `yaml:"pf"`
}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []struct {
				Name   string `yaml:"name"`
				Wounds int    `yaml:"wounds"`
				PF     int    `yaml:"pf"`
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
			Name   string `yaml:"name"`
			Wounds int    `yaml:"wounds"`
			PF     int    `yaml:"pf"`
		}{}, nil
	}
	return payload.Entries, nil
}
