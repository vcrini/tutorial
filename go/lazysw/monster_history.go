package main

func (m *model) currentMonsterIndex() int {
	list := m.filteredMonsters()
	if len(list) == 0 {
		return -1
	}
	if m.monsterCursor < 0 {
		return 0
	}
	if m.monsterCursor >= len(list) {
		return len(list) - 1
	}
	return m.monsterCursor
}

func (m *model) pushMonsterHistory() {
	idx := m.currentMonsterIndex()
	if idx < 0 {
		return
	}
	if len(m.monsterHistory) > 0 && m.monsterHistory[m.monsterHistIndex] == idx {
		return
	}
	if m.monsterHistIndex < len(m.monsterHistory)-1 {
		m.monsterHistory = m.monsterHistory[:m.monsterHistIndex+1]
	}
	m.monsterHistory = append(m.monsterHistory, idx)
	m.monsterHistIndex = len(m.monsterHistory) - 1
}

func (m *model) monsterHistoryBack() {
	if len(m.monsterHistory) == 0 || m.monsterHistIndex <= 0 {
		return
	}
	m.monsterHistIndex--
	m.monsterCursor = m.monsterHistory[m.monsterHistIndex]
	m.clampMonsterCursor()
}

func (m *model) monsterHistoryForward() {
	if len(m.monsterHistory) == 0 || m.monsterHistIndex >= len(m.monsterHistory)-1 {
		return
	}
	m.monsterHistIndex++
	m.monsterCursor = m.monsterHistory[m.monsterHistIndex]
	m.clampMonsterCursor()
}
