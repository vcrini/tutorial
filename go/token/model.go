package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// appState definisce i diversi stati dell'applicazione.
type appState int

const (
	menuState appState = iota
	createPNGState
)

// Model è lo stato della nostra applicazione TUI.
type model struct {
	choices          []string        // Le opzioni del nostro menu principale
	cursor           int             // L'indice dell'opzione attualmente selezionata nel menu principale
	message          string          // Il messaggio da mostrare all'utente
	quitting         bool            // Flag per indicare se l'applicazione sta per chiudersi
	pngs             []PNG           // La lista dei PNG gestiti
	selectedPNGIndex int             // L'indice del PNG attualmente selezionato per le operazioni
	appState         appState        // Lo stato attuale dell'applicazione
	textInput        textinput.Model // Input per il nome del nuovo PNG
	width            int             // Larghezza della finestra
	height           int             // Altezza della finestra
	focusedPanel     int             // 0=menu, 1=pngs
	showHelp         bool            // Mostra la finestra di aiuto
}

// Init viene chiamata una volta all'avvio del programma per inizializzare il modello.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update viene chiamato su ogni messaggio (es. pressione di un tasto) per aggiornare il modello.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.appState {
	case menuState:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc", "q", "ctrl+q":
				m.quitting = true
				return m, tea.Quit

			case "1":
				m.focusedPanel = 0
			case "2":
				m.focusedPanel = 1
			case "tab":
				if m.focusedPanel == 0 {
					m.focusedPanel = 1
				} else {
					m.focusedPanel = 0
				}
			case "?":
				m.showHelp = !m.showHelp

			case "up", "k":
				if m.focusedPanel == 1 {
					m.selectPrevPNG()
				} else if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.focusedPanel == 1 {
					m.selectNextPNG()
				} else if m.cursor < len(m.choices)-1 {
					m.cursor++
				}

			case "left", "h":
				if m.focusedPanel == 1 {
					m.decrementSelectedToken()
				}

			case "right", "l":
				if m.focusedPanel == 1 {
					m.incrementSelectedToken()
				}

			case "d", "x", "backspace", "delete":
				if m.focusedPanel == 1 {
					m.deleteSelectedPNG()
				}

			case "n":
				if m.focusedPanel == 1 {
					m.appState = createPNGState
					m.textInput.Reset()
					m.message = "Inserisci il nome del nuovo PNG:"
					return m, textinput.Blink
				}

			case "r":
				if m.focusedPanel == 1 {
					return m.handleMenuChoice("Resetta Tutti i Token PNG")
				}

			case "enter":
				return m.handleMenuChoice(m.choices[m.cursor])
			}
		}

	case createPNGState:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				name := m.textInput.Value()
				if strings.TrimSpace(name) == "" {
					name = uniqueRandomPNGName(m.pngs)
				}

				// Controlla se il nome esiste già
				found := false
				for _, p := range m.pngs {
					if p.Name == name {
						found = true
						break
					}
				}
				if found {
					m.message = fmt.Sprintf("Un PNG con il nome '%s' esiste già. Scegli un nome diverso.", name)
				} else {
					newPNG := PNG{Name: name, Token: defaultToken}
					m.pngs = append(m.pngs, newPNG)
					m.selectedPNGIndex = len(m.pngs) - 1 // Seleziona il nuovo PNG
					if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
						m.message = fmt.Sprintf("PNG '%s' creato ma salvataggio fallito: %v", name, err)
					} else {
						m.message = fmt.Sprintf("PNG '%s' creato con token %d.", name, defaultToken)
					}
					m.appState = menuState
				}
				return m, nil
			case "esc", "ctrl+c", "q", "ctrl+q":
				m.appState = menuState
				m.message = "Creazione PNG annullata."
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, cmd
}
