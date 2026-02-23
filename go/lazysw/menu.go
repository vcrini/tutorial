package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var menuChoices = []string{
	"Crea PNG",
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
