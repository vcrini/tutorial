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
	focusedPanel     int             // 0=pngs, 1=incontro, 2=mostri
	showHelp         bool            // Mostra la finestra di aiuto
	monsters         []Monster
	monsterSearch    textinput.Model
	monsterCursor    int
	monsterHistory   []int
	monsterHistIndex int
	encounter        []EncounterEntry
	encounterCursor  int
	detailsCompact   bool
	helpFilter       textinput.Model
	helpFilterActive bool
}

// Init viene chiamata una volta all'avvio del programma per inizializzare il modello.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update viene chiamato su ogni messaggio (es. pressione di un tasto) per aggiornare il modello.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Help overlay handling
	if m.showHelp {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "q":
				m.showHelp = false
				m.helpFilter.Blur()
				m.helpFilterActive = false
				return m, nil
			case "/":
				m.helpFilterActive = true
				m.helpFilter.SetValue("")
				m.helpFilter.Focus()
				return m, nil
			case "enter":
				if m.helpFilterActive {
					m.helpFilterActive = false
					m.helpFilter.Blur()
					return m, nil
				}
			}
		}
		if m.helpFilterActive {
			m.helpFilter, cmd = m.helpFilter.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch m.appState {
	case menuState:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil
		case tea.KeyMsg:
			handled := false
			switch msg.String() {
			case "esc":
				if m.showHelp {
					m.showHelp = false
					return m, nil
				}
				m.quitting = true
				return m, tea.Quit
			case "q":
				if m.showHelp {
					m.showHelp = false
					return m, nil
				}
				m.quitting = true
				return m, tea.Quit
			case "ctrl+c", "ctrl+q":
				m.quitting = true
				return m, tea.Quit

			case "1":
				m.focusedPanel = 0
				m.monsterSearch.Blur()
				handled = true
			case "2":
				m.focusedPanel = 1
				m.monsterSearch.Blur()
				handled = true
			case "3":
				m.focusedPanel = 2
				m.monsterSearch.Focus()
				handled = true
			case "tab":
				if m.focusedPanel == 0 {
					m.focusedPanel = 1
					m.monsterSearch.Blur()
				} else if m.focusedPanel == 1 {
					m.focusedPanel = 2
					m.monsterSearch.Focus()
				} else {
					m.focusedPanel = 0
					m.monsterSearch.Blur()
				}
				handled = true
			case "?":
				m.showHelp = !m.showHelp
				handled = true

			case "up", "k":
				if m.focusedPanel == 0 {
					m.selectPrevPNG()
				} else if m.focusedPanel == 1 {
					if m.encounterCursor > 0 {
						m.encounterCursor--
					}
				} else {
					m.monsterCursor--
					m.clampMonsterCursor()
					m.pushMonsterHistory()
				}

			case "down", "j":
				if m.focusedPanel == 0 {
					m.selectNextPNG()
				} else if m.focusedPanel == 1 {
					if m.encounterCursor < len(m.encounter)-1 {
						m.encounterCursor++
					}
				} else {
					m.monsterCursor++
					m.clampMonsterCursor()
					m.pushMonsterHistory()
				}

			case "left", "h":
				if m.focusedPanel == 0 {
					m.decrementSelectedToken()
				}
				handled = true

			case "right", "l":
				if m.focusedPanel == 0 {
					m.incrementSelectedToken()
				}
				handled = true

			case "d", "x", "backspace", "delete":
				if m.focusedPanel == 0 {
					m.deleteSelectedPNG()
					handled = true
				} else if m.focusedPanel == 1 {
					if len(m.encounter) > 0 {
						m.removeEncounterAt(m.encounterCursor)
						m.message = "Mostro rimosso dall'incontro."
					}
					handled = true
				}

			case "n":
				if m.focusedPanel == 0 {
					m.appState = createPNGState
					m.textInput.Reset()
					m.message = "Inserisci il nome del nuovo PNG:"
					return m, textinput.Blink
				}
				handled = true

			case "r":
				if m.focusedPanel == 0 {
					return m.handleMenuChoice("Resetta Tutti i Token PNG")
				}

			case "t":
				m.detailsCompact = !m.detailsCompact
				handled = true

			case "enter":
				if m.focusedPanel == 0 {
					return m.handleMenuChoice(m.choices[m.cursor])
				}
				handled = true
			case "a":
				if m.focusedPanel == 2 {
					if mon, ok := m.currentMonster(); ok {
						m.addMonsterToEncounter(mon)
						m.message = "Aggiunto a incontro: " + mon.Name
					}
				}
				handled = true
			case "ctrl+o":
				if m.focusedPanel == 2 {
					m.monsterHistoryBack()
				}
				handled = true
			case "ctrl+i":
				if m.focusedPanel == 2 {
					m.monsterHistoryForward()
				}
				handled = true
			}
			if m.focusedPanel == 2 && !handled {
				var cmd tea.Cmd
				m.monsterSearch, cmd = m.monsterSearch.Update(msg)
				m.clampMonsterCursor()
				m.pushMonsterHistory()
				return m, cmd
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
			case "esc":
				if m.showHelp {
					m.showHelp = false
					return m, nil
				}
				m.appState = menuState
				m.message = "Creazione PNG annullata."
				return m, nil
			case "q":
				if m.showHelp {
					m.showHelp = false
					return m, nil
				}
				m.appState = menuState
				m.message = "Creazione PNG annullata."
				return m, nil
			case "ctrl+c", "ctrl+q":
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
