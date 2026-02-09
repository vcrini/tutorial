package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var menuChoices = []string{
	"Crea PNG",
	"Ricarica PNG da disco",
	"Salva PNG su disco",
	"Resetta Tutti i Token PNG",
	"Esci",
}

func (m model) handleMenuChoice(choice string) (model, tea.Cmd) {
	switch choice {
	case "Crea PNG":
		m.appState = createPNGState
		m.textInput.Reset()
		m.message = "Inserisci il nome del nuovo PNG:"
		return m, textinput.Blink
	case "Ricarica PNG da disco":
		selectedName := ""
		if m.selectedPNGIndex >= 0 && m.selectedPNGIndex < len(m.pngs) {
			selectedName = m.pngs[m.selectedPNGIndex].Name
		}
		pngs, selected, err := loadPNGList(dataFile)
		if err != nil {
			m.message = fmt.Sprintf("Errore nel caricare %s: %v", dataFile, err)
		} else {
			m.pngs = pngs
			m.selectedPNGIndex = -1
			if len(m.pngs) > 0 {
				if selected != "" {
					for i, p := range m.pngs {
						if p.Name == selected {
							m.selectedPNGIndex = i
							break
						}
					}
				} else {
					for i, p := range m.pngs {
						if p.Name == selectedName && selectedName != "" {
							m.selectedPNGIndex = i
							break
						}
					}
				}
				if m.selectedPNGIndex == -1 {
					m.selectedPNGIndex = 0
				}
			}
			m.message = fmt.Sprintf("Lista PNG caricata da %s (%d).", dataFile, len(m.pngs))
		}
	case "Salva PNG su disco":
		if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
			m.message = fmt.Sprintf("Errore nel salvataggio su %s: %v", dataFile, err)
		} else {
			m.message = fmt.Sprintf("Lista PNG salvata su %s.", dataFile)
		}
	case "Resetta Tutti i Token PNG":
		if len(m.pngs) == 0 {
			m.message = "Nessun PNG presente per resettare i token."
		} else {
			for i := range m.pngs {
				m.pngs[i].Token = defaultToken
			}
			if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
				m.message = fmt.Sprintf("Token resettati, ma salvataggio fallito: %v", err)
			} else {
				m.message = fmt.Sprintf("Tutti i token PNG sono stati resettati a %d.", defaultToken)
			}
		}
	case "Esci":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}
