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

func limitLines(s string, max int) string {
	if max <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	return strings.Join(lines[:max], "\n")
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

func loadPNGList(path string) ([]PNG, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PNG{}, "", nil
		}
		return nil, "", err
	}

	var wrapper struct {
		PNGs     []PNG  `json:"pngs"`
		Selected string `json:"selected"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.PNGs != nil {
		return wrapper.PNGs, wrapper.Selected, nil
	}

	var legacy []PNG
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, "", err
	}
	return legacy, "", nil
}

func savePNGList(path string, pngs []PNG, selected string) error {
	payload := struct {
		PNGs     []PNG  `json:"pngs"`
		Selected string `json:"selected"`
	}{
		PNGs:     pngs,
		Selected: selected,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func selectedPNGName(pngs []PNG, idx int) string {
	if idx < 0 || idx >= len(pngs) {
		return ""
	}
	return pngs[idx].Name
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

// View rende l'interfaccia utente.
func (m model) View() string {
	if m.quitting {
		return "Arrivederci!\n"
	}

	// Stili ispirati a lazygit/lazydocker
	border := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}
	panel := lipgloss.NewStyle().Border(border).Padding(0, 1)
	panelFocused := lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color("10")).Padding(0, 1)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	highlight := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	selectedItemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	selectedPNGStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	totalWidth := m.width
	if totalWidth <= 0 {
		totalWidth = 96
	}
	if totalWidth < 60 {
		totalWidth = 60
	}
	leftWidth := totalWidth / 3
	if leftWidth > 32 {
		leftWidth = 32
	}
	if leftWidth < 24 {
		leftWidth = 24
	}
	rightWidth := totalWidth - leftWidth

	totalHeight := m.height
	if totalHeight <= 0 {
		totalHeight = 24
	}
	if totalHeight < 12 {
		totalHeight = 12
	}
	bodyContentHeight := max(totalHeight-6, 4)

	// Header
	header := titleStyle.Render(" PNG Manager ") + dim.Render(" • Lazy style UI")
	headerBar := panel.Width(leftWidth + rightWidth).Render(header)

	// Menu (sinistra)
	var menu strings.Builder
	menu.WriteString(titleStyle.Render(" Comandi ") + "\n\n")
	menu.WriteString(dim.Render("Shortcut: 1=Comandi 2=PNGs") + "\n\n")
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i && m.appState == menuState {
			cursor = selectedItemStyle.Render("➜") + " "
		}
		menu.WriteString(fmt.Sprintf("%s%s\n", cursor, choice))
	}
	menuPanelStyle := panel
	if m.focusedPanel == 0 {
		menuPanelStyle = panelFocused
	}
	menuPanel := menuPanelStyle.Width(leftWidth).Render(limitLines(menu.String(), bodyContentHeight))

	// Pannello destro superiore
	var rightTop strings.Builder
	switch m.appState {
	case createPNGState:
		rightTop.WriteString(titleStyle.Render(" Nuovo PNG ") + "\n\n")
		rightTop.WriteString(m.message + "\n\n")
		rightTop.WriteString(m.textInput.View() + "\n\n")
		rightTop.WriteString(dim.Render("(Enter per confermare, Esc per annullare)"))
	case selectPNGState:
		rightTop.WriteString(titleStyle.Render(" Seleziona PNG ") + "\n\n")
		if len(m.pngs) == 0 {
			rightTop.WriteString(dim.Render("Nessun PNG disponibile."))
		} else {
			for i, png := range m.pngs {
				cursor := "  "
				if m.selectPNGCursor == i {
					cursor = selectedItemStyle.Render("➜") + " "
				}
				rightTop.WriteString(fmt.Sprintf("%s%s (Contatore: %d)\n", cursor, png.Name, png.Counter))
			}
			rightTop.WriteString("\n" + dim.Render("(Enter per selezionare, Esc per annullare)"))
		}
	default:
		rightTop.WriteString(titleStyle.Render(" PNGs ") + "\n\n")
		if len(m.pngs) == 0 {
			rightTop.WriteString(dim.Render("Nessun PNG creato."))
		} else {
			for i, png := range m.pngs {
				line := fmt.Sprintf("%s (Contatore: %d)", png.Name, png.Counter)
				if i == m.selectedPNGIndex {
					rightTop.WriteString(selectedPNGStyle.Render("• "+line) + "\n")
				} else {
					rightTop.WriteString("  " + line + "\n")
				}
			}
			if m.selectedPNGIndex == -1 {
				rightTop.WriteString("\n" + dim.Render("Nessun PNG selezionato."))
			}
		}
	}

	rightPanelStyle := panel
	if m.focusedPanel == 1 {
		rightPanelStyle = panelFocused
	}
	rightTopPanel := rightPanelStyle.Width(rightWidth).Render(limitLines(rightTop.String(), bodyContentHeight))

	// Barra messaggi
	message := m.message
	if message == "" {
		message = "Pronto."
	}
	messageBar := panel.Width(leftWidth + rightWidth).Render(highlight.Render(" Msg ") + " " + message)

	// Layout finale
	body := lipgloss.JoinHorizontal(lipgloss.Top, menuPanel, rightTopPanel)
	return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, messageBar)
}

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
		selectedPNGIndex: selectedIndex, // Nessun PNG selezionato inizialmente
		appState:         menuState,
		textInput:        ti,
		focusedPanel:     0,
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
