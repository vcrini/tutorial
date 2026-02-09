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
	selectPNGState
	// Aggiungi altri stati se necessario
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
	selectPNGCursor  int             // Il cursore per la selezione del PNG
	width            int             // Larghezza della finestra
	height           int             // Altezza della finestra
	focusedPanel     int             // 0=menu, 1=pngs
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
			case "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit

			case "1":
				m.focusedPanel = 0
			case "2":
				m.focusedPanel = 1

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
				m.selectPrevPNG()

			case "right", "l":
				m.selectNextPNG()

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
					newPNG := PNG{Name: name, Counter: defaultCounter}
					m.pngs = append(m.pngs, newPNG)
					m.selectedPNGIndex = len(m.pngs) - 1 // Seleziona il nuovo PNG
					if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
						m.message = fmt.Sprintf("PNG '%s' creato ma salvataggio fallito: %v", name, err)
					} else {
						m.message = fmt.Sprintf("PNG '%s' creato con contatore %d.", name, defaultCounter)
					}
					m.appState = menuState
				}
				return m, nil
			case "esc", "ctrl+c":
				m.appState = menuState
				m.message = "Creazione PNG annullata."
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case selectPNGState:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.selectPNGCursor > 0 {
					m.selectPNGCursor--
				}
			case "down", "j":
				if m.selectPNGCursor < len(m.pngs)-1 {
					m.selectPNGCursor++
				}
			case "enter":
				m.selectedPNGIndex = m.selectPNGCursor
				m.persistSelection()
				m.message = fmt.Sprintf("PNG '%s' selezionato (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
				m.appState = menuState
			case "esc", "ctrl+c":
				m.appState = menuState
				m.message = "Selezione PNG annullata."
			}
		}
	}

	return m, cmd
}
