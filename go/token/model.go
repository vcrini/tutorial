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
					if len(m.pngs) > 0 {
						if m.selectedPNGIndex == -1 {
							m.selectedPNGIndex = 0
						} else {
							m.selectedPNGIndex = (m.selectedPNGIndex - 1 + len(m.pngs)) % len(m.pngs)
						}
						_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
						m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
					} else {
						m.message = "Nessun PNG disponibile per la selezione."
					}
				} else if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.focusedPanel == 1 {
					if len(m.pngs) > 0 {
						if m.selectedPNGIndex == -1 {
							m.selectedPNGIndex = 0
						} else {
							m.selectedPNGIndex = (m.selectedPNGIndex + 1) % len(m.pngs)
						}
						_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
						m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
					} else {
						m.message = "Nessun PNG disponibile per la selezione."
					}
				} else if m.cursor < len(m.choices)-1 {
					m.cursor++
				}

			case "left", "h":
				if len(m.pngs) > 0 {
					if m.selectedPNGIndex == -1 {
						m.selectedPNGIndex = len(m.pngs) - 1 // Seleziona l'ultimo se nessuno è selezionato
					} else {
						m.selectedPNGIndex = (m.selectedPNGIndex - 1 + len(m.pngs)) % len(m.pngs)
					}
					_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
					m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
				} else {
					m.message = "Nessun PNG disponibile per la selezione."
				}

			case "right", "l":
				if len(m.pngs) > 0 {
					if m.selectedPNGIndex == -1 {
						m.selectedPNGIndex = 0 // Seleziona il primo se nessuno è selezionato
					} else {
						m.selectedPNGIndex = (m.selectedPNGIndex + 1) % len(m.pngs)
					}
					_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
					m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
				} else {
					m.message = "Nessun PNG disponibile per la selezione."
				}

			case "enter":
				switch m.choices[m.cursor] {
				case "Crea PNG":
					m.appState = createPNGState
					m.textInput.Reset()
					m.message = "Inserisci il nome del nuovo PNG:"
					return m, textinput.Blink
				case "Crea PNG casuale":
					name := uniqueRandomPNGName(m.pngs)
					newPNG := PNG{Name: name, Counter: defaultCounter}
					m.pngs = append(m.pngs, newPNG)
					m.selectedPNGIndex = len(m.pngs) - 1
					if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
						m.message = fmt.Sprintf("PNG '%s' creato ma salvataggio fallito: %v", name, err)
					} else {
						m.message = fmt.Sprintf("PNG '%s' creato con contatore %d.", name, defaultCounter)
					}
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
						m.selectPNGCursor = 0
						m.message = fmt.Sprintf("Lista PNG caricata da %s (%d).", dataFile, len(m.pngs))
					}
				case "Salva PNG su disco":
					if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
						m.message = fmt.Sprintf("Errore nel salvataggio su %s: %v", dataFile, err)
					} else {
						m.message = fmt.Sprintf("Lista PNG salvata su %s.", dataFile)
					}
				case "Seleziona PNG":
					if len(m.pngs) == 0 {
						m.message = "Nessun PNG disponibile da selezionare. Creane uno prima!"
					} else {
						m.appState = selectPNGState
						m.selectPNGCursor = 0 // Resetta il cursore di selezione
						m.message = "Seleziona un PNG dalla lista:"
					}
				case "Decrementa Contatore PNG":
					if m.selectedPNGIndex == -1 || len(m.pngs) == 0 {
						m.message = "Nessun PNG selezionato. Seleziona un PNG prima di decrementare."
					} else {
						png := &m.pngs[m.selectedPNGIndex]
						if png.Counter > minCounter {
							png.Counter--
							if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
								m.message = fmt.Sprintf("Contatore di '%s' decrementato, ma salvataggio fallito: %v", png.Name, err)
							} else {
								m.message = fmt.Sprintf("Contatore di '%s' decrementato a %d.", png.Name, png.Counter)
							}
						} else {
							m.message = fmt.Sprintf("Il contatore di '%s' è già al minimo (%d).", png.Name, minCounter)
						}
					}
				case "Resetta Contatore PNG":
					if m.selectedPNGIndex == -1 || len(m.pngs) == 0 {
						m.message = "Nessun PNG selezionato. Seleziona un PNG prima di resettare."
					} else {
						png := &m.pngs[m.selectedPNGIndex]
						png.Counter = defaultCounter
						if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
							m.message = fmt.Sprintf("Contatore di '%s' resettato, ma salvataggio fallito: %v", png.Name, err)
						} else {
							m.message = fmt.Sprintf("Contatore di '%s' resettato a %d.", png.Name, png.Counter)
						}
					}
				case "Resetta Tutti i Contatori PNG":
					if len(m.pngs) == 0 {
						m.message = "Nessun PNG presente per resettare i contatori."
					} else {
						for i := range m.pngs {
							m.pngs[i].Counter = defaultCounter // Resetta al valore di default (3)
						}
						if err := savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex)); err != nil {
							m.message = fmt.Sprintf("Contatori resettati, ma salvataggio fallito: %v", err)
						} else {
							m.message = fmt.Sprintf("Tutti i contatori PNG sono stati resettati a %d.", defaultCounter)
						}
					}
				case "Esci":
					m.quitting = true
					return m, tea.Quit
				}
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
				_ = savePNGList(dataFile, m.pngs, selectedPNGName(m.pngs, m.selectedPNGIndex))
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
