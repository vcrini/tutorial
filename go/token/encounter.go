package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster Monster
}

type encounterPersist struct {
	Names []string `yaml:"names"`
}

func (m *model) addMonsterToEncounter(mon Monster) {
	m.encounter = append(m.encounter, EncounterEntry{Monster: mon})
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
	var names []string
	for _, e := range m.encounter {
		names = append(names, e.Monster.Name)
	}
	_ = saveEncounter(encounterFile, names)
}

func loadEncounter(path string, monsters []Monster) ([]EncounterEntry, error) {
	names, err := readEncounter(path)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return []EncounterEntry{}, nil
	}

	byName := make(map[string]Monster, len(monsters))
	for _, m := range monsters {
		byName[m.Name] = m
	}

	var entries []EncounterEntry
	for _, name := range names {
		if mon, ok := byName[name]; ok {
			entries = append(entries, EncounterEntry{Monster: mon})
		} else {
			entries = append(entries, EncounterEntry{Monster: Monster{Name: name}})
		}
	}
	return entries, nil
}

func saveEncounter(path string, names []string) error {
	payload := encounterPersist{Names: names}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readEncounter(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var payload encounterPersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.Names, nil
}
