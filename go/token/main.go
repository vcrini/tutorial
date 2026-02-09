package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultCounter = 3
	minCounter     = 0 // Il valore minimo
	dataFile       = "pngs.json"
)

// PNG rappresenta la struttura dati per un PNG con il suo contatore.
type PNG struct {
	Name    string
	Counter int
}

func randomPNGName() string {
	adjectives := []string{
		"antico", "arcano", "celestiale", "crepuscolare", "dorato", "draconico",
		"incantato", "lunare", "mistico", "nobile", "ruggente", "sacro",
		"segreto", "tempestoso", "valente", "velato",
	}
	nouns := []string{
		"drago", "grifone", "fenice", "runa", "santuario", "torre", "reliquia",
		"spada", "scudo", "foresta", "regno", "oracolo", "ombra", "stella",
		"valle", "vento",
	}
	suffixes := []string{
		"al", "anor", "dellalba", "delcrepuscolo", "dor", "eld", "fir",
		"gorn", "ion", "kor", "lith", "mir", "nath", "rend", "thor", "vyr",
	}

	adj := capitalizeWord(adjectives[rand.IntN(len(adjectives))])
	noun := capitalizeWord(nouns[rand.IntN(len(nouns))])
	suffix := capitalizeWord(suffixes[rand.IntN(len(suffixes))])
	return fmt.Sprintf("%s %s %s", adj, noun, suffix)
}

func capitalizeWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func uniqueRandomPNGName(existing []PNG) string {
	seen := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		seen[p.Name] = struct{}{}
	}
	for {
		name := randomPNGName()
		if _, ok := seen[name]; !ok {
			return name
		}
	}
}

func loadPNGList(path string) ([]PNG, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PNG{}, nil
		}
		return nil, err
	}

	var pngs []PNG
	if err := json.Unmarshal(data, &pngs); err != nil {
		return nil, err
	}
	return pngs, nil
}

func savePNGList(path string, pngs []PNG) error {
	data, err := json.MarshalIndent(pngs, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

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
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}

			case "left", "h":
				if len(m.pngs) > 0 {
					if m.selectedPNGIndex == -1 {
						m.selectedPNGIndex = len(m.pngs) - 1 // Seleziona l'ultimo se nessuno è selezionato
					} else {
						m.selectedPNGIndex = (m.selectedPNGIndex - 1 + len(m.pngs)) % len(m.pngs)
					}
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
					m.message = fmt.Sprintf("PNG selezionato: '%s' (contatore: %d).", m.pngs[m.selectedPNGIndex].Name, m.pngs[m.selectedPNGIndex].Counter)
				} else {
					m.message = "Nessun PNG disponibile per la selezione."
				}

			case "enter":
				switch m.choices[m.cursor] {
				// case "Saluta":
				// 	m.message = "Ciao! Benvenuto nell'applicazione Bubble Tea."
				// case "Mostra Data/Ora":
				// 	currentTime := time.Now().Format("02/01/2006 15:04:05")
				// 	m.message = fmt.Sprintf("La data e l'ora attuali sono: %s", currentTime)
				// case "Numero Casuale":
				// 	randomNumber := rand.IntN(1000) + 1
				// 	m.message = fmt.Sprintf("Il tuo numero casuale è: %d", randomNumber)
				case "Crea PNG":
					m.appState = createPNGState
					m.textInput.Reset()
					m.message = "Inserisci il nome del nuovo PNG:"
					return m, textinput.Blink
				case "Crea PNG casuale":
					name := uniqueRandomPNGName(m.pngs)
					newPNG := PNG{Name: name, Counter: defaultCounter}
					m.pngs = append(m.pngs, newPNG)
					if err := savePNGList(dataFile, m.pngs); err != nil {
						m.message = fmt.Sprintf("PNG '%s' creato ma salvataggio fallito: %v", name, err)
					} else {
						m.message = fmt.Sprintf("PNG '%s' creato con contatore %d.", name, defaultCounter)
					}
					m.selectedPNGIndex = len(m.pngs) - 1
				case "Ricarica PNG da disco":
					pngs, err := loadPNGList(dataFile)
					if err != nil {
						m.message = fmt.Sprintf("Errore nel caricare %s: %v", dataFile, err)
					} else {
						m.pngs = pngs
						if len(m.pngs) == 0 {
							m.selectedPNGIndex = -1
						} else {
							m.selectedPNGIndex = 0
						}
						m.selectPNGCursor = 0
						m.message = fmt.Sprintf("Lista PNG caricata da %s (%d).", dataFile, len(m.pngs))
					}
				case "Salva PNG su disco":
					if err := savePNGList(dataFile, m.pngs); err != nil {
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
							if err := savePNGList(dataFile, m.pngs); err != nil {
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
						if err := savePNGList(dataFile, m.pngs); err != nil {
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
						if err := savePNGList(dataFile, m.pngs); err != nil {
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
					if err := savePNGList(dataFile, m.pngs); err != nil {
						m.message = fmt.Sprintf("PNG '%s' creato ma salvataggio fallito: %v", name, err)
					} else {
						m.message = fmt.Sprintf("PNG '%s' creato con contatore %d.", name, defaultCounter)
					}
					m.selectedPNGIndex = len(m.pngs) - 1 // Seleziona il nuovo PNG
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

// View rende l'interfaccia utente.
func (m model) View() string {
	if m.quitting {
		return "Arrivederci!\n"
	}

	s := strings.Builder{}

	// Stile per il titolo
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).PaddingBottom(1)
	// Stile per il menu selezionato
	selectedItemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	// Stile per i messaggi
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).PaddingTop(1)
	// Stile per l'elenco PNG selezionato
	selectedPNGStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)

	// Titolo
	s.WriteString(titleStyle.Render("Gestione PNG con Contatore (Bubble Tea)\n"))
	s.WriteString(strings.Repeat("─", 50) + "\n\n")

	// Contenuto basato sullo stato dell'applicazione
	switch m.appState {
	case menuState:
		s.WriteString("Seleziona un'opzione:\n\n")
		for i, choice := range m.choices {
			cursor := "  "
			if m.cursor == i {
				cursor = selectedItemStyle.Render("->") + " "
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, choice))
		}

		s.WriteString("\n--- PNGs Disponibili ---\n")
		if len(m.pngs) == 0 {
			s.WriteString("Nessun PNG creato. Scegli 'Crea PNG' per aggiungerne uno.\n")
		} else {
			for i, png := range m.pngs {
				pngLine := fmt.Sprintf("  %s (Contatore: %d)", png.Name, png.Counter)
				if i == m.selectedPNGIndex {
					s.WriteString(selectedPNGStyle.Render(pngLine) + " <- Selezionato\n")
				} else {
					s.WriteString(pngLine + "\n")
				}
			}
			if m.selectedPNGIndex == -1 {
				s.WriteString("\nNessun PNG selezionato. Scegli 'Seleziona PNG' per sceglierne uno.\n")
			}
		}

	case createPNGState:
		s.WriteString(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			m.message,
			m.textInput.View(),
			"(Premi Enter per confermare, Esc per annullare)",
		))

	case selectPNGState:
		s.WriteString("Seleziona un PNG:\n\n")
		if len(m.pngs) == 0 {
			s.WriteString("Nessun PNG disponibile.\n")
		} else {
			for i, png := range m.pngs {
				cursor := "  "
				if m.selectPNGCursor == i {
					cursor = selectedItemStyle.Render("->") + " "
				}
				s.WriteString(fmt.Sprintf("%s%s (Contatore: %d)\n", cursor, png.Name, png.Counter))
			}
			s.WriteString("\n(Premi Enter per selezionare, Esc per annullare)")
		}
	}

	// Messaggio globale
	s.WriteString(messageStyle.Render(fmt.Sprintf("\n%s\n", m.message)))

	return s.String()
}

func main() {
	ti := textinput.New()
	ti.Placeholder = "Nome PNG..."
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 20
	ti.Prompt = "Nome: "

	pngs, err := loadPNGList(dataFile)
	initialMessage := "Benvenuto! Premi Enter per scegliere un'opzione o frecce per navigare."
	if err != nil {
		initialMessage = fmt.Sprintf("Errore nel caricare %s: %v", dataFile, err)
		pngs = []PNG{}
	}

	p := tea.NewProgram(model{
		choices: []string{
			// "Saluta",
			// "Mostra Data/Ora",
			// "Numero Casuale",
			"Crea PNG",
			"Crea PNG casuale",
			"Ricarica PNG da disco",
			"Salva PNG su disco",
			"Seleziona PNG",
			"Decrementa Contatore PNG",
			"Resetta Contatore PNG",
			"Resetta Tutti i Contatori PNG",
			"Esci",
		},
		message:          initialMessage,
		pngs:             pngs,
		selectedPNGIndex: -1, // Nessun PNG selezionato inizialmente
		appState:         menuState,
		textInput:        ti,
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
