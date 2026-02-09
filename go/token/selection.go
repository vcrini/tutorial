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
	m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
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
	m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
}

func (m *model) selectFirstPNG() {
	if len(m.pngs) == 0 {
		m.selectedPNGIndex = -1
		return
	}
	m.selectedPNGIndex = 0
	m.persistSelection()
}

func (m *model) persistSelection() {
	_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
}
