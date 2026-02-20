package main

import "fmt"

func (m *model) selectPrevPNG() {
	if len(m.pngs) == 0 {
		m.message = "Nessun PNG disponibile per la selezione."
		return
	}
	if m.selectedPNGIndex == -1 {
		m.selectedPNGIndex = len(m.pngs) - 1
	} else {
		m.selectedPNGIndex = (m.selectedPNGIndex - 1 + len(m.pngs)) % len(m.pngs)
	}
	m.persistSelection()
	m.message = fmt.Sprintf("PNG selezionato: '%s' (token: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Token)
}

func (m *model) selectNextPNG() {
	if len(m.pngs) == 0 {
		m.message = "Nessun PNG disponibile per la selezione."
		return
	}
	if m.selectedPNGIndex == -1 {
		m.selectedPNGIndex = 0
	} else {
		m.selectedPNGIndex = (m.selectedPNGIndex + 1) % len(m.pngs)
	}
	m.persistSelection()
	m.message = fmt.Sprintf("PNG selezionato: '%s' (token: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Token)
}

func (m *model) decrementSelectedToken() {
	if m.selectedPNGIndex == -1 || len(m.pngs) == 0 {
		m.message = "Nessun PNG selezionato."
		return
	}
	png := &m.pngs[m.selectedPNGIndex]
	if png.Token > minToken {
		png.Token--
		m.persistSelection()
		m.message = fmt.Sprintf("Token di '%s' decrementato a %d (usa ←→ per modificare).", png.Name, png.Token)
	} else {
		m.message = fmt.Sprintf("Token di '%s' è già al minimo (%d).", png.Name, minToken)
	}
}

func (m *model) incrementSelectedToken() {
	if m.selectedPNGIndex == -1 || len(m.pngs) == 0 {
		m.message = "Nessun PNG selezionato."
		return
	}
	png := &m.pngs[m.selectedPNGIndex]
	if png.Token < maxToken {
		png.Token++
		m.persistSelection()
		m.message = fmt.Sprintf("Token di '%s' incrementato a %d (usa ←→ per modificare).", png.Name, png.Token)
	} else {
		m.message = fmt.Sprintf("Token di '%s' è già al massimo (%d).", png.Name, maxToken)
	}
}

func (m *model) deleteSelectedPNG() {
	if m.selectedPNGIndex == -1 || len(m.pngs) == 0 {
		m.message = "Nessun PNG selezionato."
		return
	}
	deleted := m.pngs[m.selectedPNGIndex].Name
	m.pngs = append(m.pngs[:m.selectedPNGIndex], m.pngs[m.selectedPNGIndex+1:]...)
	if len(m.pngs) == 0 {
		m.selectedPNGIndex = -1
	} else if m.selectedPNGIndex >= len(m.pngs) {
		m.selectedPNGIndex = len(m.pngs) - 1
	}
	m.persistSelection()
	m.message = fmt.Sprintf("PNG '%s' eliminato.", deleted)
}

func (m *model) persistSelection() {
	_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
}
