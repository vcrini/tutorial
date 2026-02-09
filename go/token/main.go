package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	ti := textinput.New()
	ti.Placeholder = "Nome PNG..."
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20
	ti.Prompt = "Nome: "

	pngs, selected, err := loadPNGList(dataFile)
	initialMessage := "Benvenuto! Premi Enter per scegliere un'opzione o frecce per navigare."
	if err != nil {
		initialMessage = fmt.Sprintf("Errore nel caricare %s: %v", dataFile, err)
		pngs = []PNG{}
	}
	selectedIndex := -1
	if selected != "" {
		for i, p := range pngs {
			if p.Name == selected {
				selectedIndex = i
				break
			}
		}
	}

	p := tea.NewProgram(model{
		choices:          menuChoices,
		message:          initialMessage,
		pngs:             pngs,
		selectedPNGIndex: selectedIndex,
		appState:         menuState,
		textInput:        ti,
		focusedPanel:     0,
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
