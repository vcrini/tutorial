package main

import "strings"

func (m model) filteredMonsters() []Monster {
	if strings.TrimSpace(m.monsterSearch.Value()) == "" {
		return m.monsters
	}
	query := strings.ToLower(strings.TrimSpace(m.monsterSearch.Value()))
	var out []Monster
	for _, mon := range m.monsters {
		if strings.Contains(strings.ToLower(mon.Name), query) {
			out = append(out, mon)
		}
	}
	return out
}

func (m model) currentMonster() (Monster, bool) {
	list := m.filteredMonsters()
	if len(list) == 0 {
		return Monster{}, false
	}
	idx := m.monsterCursor
	if idx < 0 {
		idx = 0
	}
	if idx >= len(list) {
		idx = len(list) - 1
	}
	return list[idx], true
}

func (m *model) clampMonsterCursor() {
	list := m.filteredMonsters()
	if len(list) == 0 {
		m.monsterCursor = 0
		return
	}
	if m.monsterCursor < 0 {
		m.monsterCursor = 0
	}
	if m.monsterCursor >= len(list) {
		m.monsterCursor = len(list) - 1
	}
}
