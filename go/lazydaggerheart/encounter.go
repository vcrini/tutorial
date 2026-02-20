package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type EncounterEntry struct {
	Monster    Monster
	Wounds     int
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
	m.encounter = append(m.encounter, EncounterEntry{
		Monster:    mon,
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
	var entries []struct {
		Name       string `yaml:"name"`
		Wounds     int    `yaml:"wounds"`
		PF         int    `yaml:"pf"`
		Stress     int    `yaml:"stress,omitempty"`
		BaseStress int    `yaml:"base_stress,omitempty"`
	}
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
		entries = append(entries, struct {
			Name       string `yaml:"name"`
			Wounds     int    `yaml:"wounds"`
			PF         int    `yaml:"pf"`
			Stress     int    `yaml:"stress,omitempty"`
			BaseStress int    `yaml:"base_stress,omitempty"`
		}{Name: e.Monster.Name, Wounds: e.Wounds, PF: basePF, Stress: currentStress, BaseStress: baseStress})
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
		stress := e.Stress
		baseStress := e.BaseStress
		if mon, ok := byName[name]; ok {
			if baseStress == 0 {
				baseStress = mon.Stress
			}
			if stress == 0 && baseStress > 0 && e.BaseStress == 0 {
				// Backward compatibility for old files without stress fields.
				stress = baseStress
			}
			entries = append(entries, EncounterEntry{Monster: mon, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress})
		} else {
			entries = append(entries, EncounterEntry{Monster: Monster{Name: name, PF: e.PF, Stress: baseStress}, Wounds: e.Wounds, BasePF: e.PF, Stress: stress, BaseStress: baseStress})
		}
	}
	return entries, nil
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
