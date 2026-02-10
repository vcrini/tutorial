package main

type EncounterEntry struct {
	Monster Monster
}

func (m *model) addMonsterToEncounter(mon Monster) {
	m.encounter = append(m.encounter, EncounterEntry{Monster: mon})
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
}
