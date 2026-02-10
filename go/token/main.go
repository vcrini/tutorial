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

	mi := textinput.New()
	mi.Placeholder = "Cerca mostro..."
	mi.CharLimit = 40
	mi.Width = 24
	mi.Prompt = "Cerca: "

	hf := textinput.New()
	hf.Placeholder = "filtra help..."
	hf.CharLimit = 40
	hf.Width = 30
	hf.Prompt = "/ "

	pngs, selected, err := loadPNGList(dataFile)
	initialMessage := "Benvenuto! Premi Enter per scegliere un'opzione o frecce per navigare."
	if err != nil {
		initialMessage = fmt.Sprintf("Errore nel caricare %s: %v", dataFile, err)
		pngs = []PNG{}
	}
	monsters, errMon := loadMonsters(monstersFile)
	if errMon != nil {
		initialMessage = fmt.Sprintf("Errore nel caricare %s: %v", monstersFile, errMon)
		monsters = []Monster{}
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
		monsters:         monsters,
		monsterSearch:    mi,
		monsterHistory:   []int{},
		monsterHistIndex: 0,
		encounter:        []EncounterEntry{},
		encounterCursor:  0,
		helpFilter:       hf,
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
