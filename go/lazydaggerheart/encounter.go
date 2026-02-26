package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster    Monster
	Seq        int
	Wounds     int
	BasePF     int
	Stress     int
	BaseStress int
}

type encounterPersistEntry struct {
	Name       string `yaml:"name"`
	Seq        int    `yaml:"seq,omitempty"`
	Wounds     int    `yaml:"wounds"`
	PF         int    `yaml:"pf"`
	Stress     int    `yaml:"stress,omitempty"`
	BaseStress int    `yaml:"base_stress,omitempty"`
}

type encounterPersist struct {
	Entries []encounterPersistEntry `yaml:"entries"`
}

func nextEncounterSeq(entries []EncounterEntry, name string) int {
	maxSeq := 0
	fallbackCount := 0
	for _, e := range entries {
		if !strings.EqualFold(strings.TrimSpace(e.Monster.Name), strings.TrimSpace(name)) {
			continue
		}
		fallbackCount++
		if e.Seq > maxSeq {
			maxSeq = e.Seq
		}
	}
	if maxSeq > 0 {
		return maxSeq + 1
	}
	return fallbackCount + 1
}

func (m *model) addMonsterToEncounter(mon Monster) {
	m.encounter = append(m.encounter, EncounterEntry{
		Monster:    mon,
		Seq:        nextEncounterSeq(m.encounter, mon.Name),
		BasePF:     mon.PF,
		Stress:     mon.Stress,
		BaseStress: mon.Stress,
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
	var entries []encounterPersistEntry
	for _, e := range m.encounter {
		basePF := e.BasePF
		if basePF == 0 {
			basePF = e.Monster.PF
		}
		baseStress := e.BaseStress
		if baseStress == 0 {
			baseStress = e.Monster.Stress
		}
		currentStress := e.Stress
		if currentStress < 0 {
			currentStress = 0
		}
		if baseStress > 0 && currentStress > baseStress {
			currentStress = baseStress
		}
		entries = append(entries, encounterPersistEntry{Name: e.Monster.Name, Seq: e.Seq, Wounds: e.Wounds, PF: basePF, Stress: currentStress, BaseStress: baseStress})
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
	assigned := map[string]int{}
	for _, e := range rawEntries {
		name := e.Name
		stress := e.Stress
		baseStress := e.BaseStress
		seq := e.Seq
		if seq <= 0 {
			assigned[name]++
			seq = assigned[name]
		} else if seq > assigned[name] {
			assigned[name] = seq
		}
		if mon, ok := byName[name]; ok {
			if baseStress == 0 {
				baseStress = mon.Stress
			}
			if stress == 0 && baseStress > 0 && e.BaseStress == 0 {
				// Backward compatibility for old files without stress fields.
				stress = baseStress
			}
			entries = append(entries, EncounterEntry{Monster: mon, Seq: seq, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress})
		} else {
			entries = append(entries, EncounterEntry{Monster: Monster{Name: name, PF: e.PF, Stress: baseStress}, Seq: seq, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress})
		}
	}
	return entries, nil
}

func saveEncounter(path string, entries []encounterPersistEntry) error {
	payload := encounterPersist{Entries: entries}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readEncounter(path string) ([]encounterPersistEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []encounterPersistEntry{}, nil
		}
		return nil, err
	}
	var payload encounterPersist
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.Entries == nil {
		return []encounterPersistEntry{}, nil
	}
	return payload.Entries, nil
}
