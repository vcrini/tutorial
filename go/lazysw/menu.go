package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var menuChoices = []string{
	"Crea PNG",
	"Esci",
}

func (m model) handleMenuChoice(choice string) (model, tea.Cmd) {
	switch choice {
	case "Crea PNG":
		m.appState = createPNGState
		m.textInput.Reset()
		m.message = "Inserisci il nome del nuovo PNG:"
		return m, textinput.Blink
	case "Esci":
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}
