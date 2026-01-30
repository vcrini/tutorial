package main

import (
	"fmt"
	"math/rand/v2" // Usiamo rand/v2 per una migliore generazione di numeri casuali
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Model è lo stato della nostra applicazione TUI.
type model struct {
	choices  []string // Le opzioni del nostro menu
	cursor   int      // L'indice dell'opzione attualmente selezionata
	message  string   // Il messaggio da mostrare all'utente
	quitting bool     // Flag per indicare se l'applicazione sta per chiudersi
}

// Init viene chiamata una volta all'avvio del programma per inizializzare il modello.
func (m model) Init() tea.Cmd {
	return nil // Non ci sono comandi iniziali da eseguire
}

// Update viene chiamato su ogni messaggio (es. pressione di un tasto) per aggiornare il modello.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc": // Esci dall'applicazione con Ctrl+C o Esc
			m.quitting = true
			return m, tea.Quit

		case "up", "k": // Naviga in alto nel menu
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j": // Naviga in basso nel menu
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter": // Esegui l'azione del pulsante selezionato
			switch m.choices[m.cursor] {
			case "Saluta":
				m.message = "Ciao! Benvenuto nell'applicazione Bubble Tea."
			case "Mostra Data/Ora":
				currentTime := time.Now().Format("02/01/2006 15:04:05")
				m.message = fmt.Sprintf("La data e l'ora attuali sono: %s", currentTime)
			case "Numero Casuale":
				randomNumber := rand.IntN(1000) + 1 // Un numero tra 1 e 1000
				m.message = fmt.Sprintf("Il tuo numero casuale è: %d", randomNumber)
			case "Esci":
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View rende l'interfaccia utente.
func (m model) View() string {
	if m.quitting {
		return "Arrivederci!\n"
	}

	// Costruisce la parte dei pulsanti/menu
	s := strings.Builder{}
	s.WriteString("Seleziona un'opzione:\n\n")

	for i, choice := range m.choices {
		cursor := "  " // Default senza cursore
		if m.cursor == i {
			cursor = "-> " // Opzione selezionata
		}
		s.WriteString(fmt.Sprintf("%s%s\n", cursor, choice))
	}

	// Ora, mettiamo il menu e il messaggio fianco a fianco.
	// Questo è un layout molto semplice per il terminale.
	// Potresti usare librerie come `lipgloss` per layout più complessi e stili.
	menuLines := strings.Split(s.String(), "\n")
	messageLines := strings.Split(m.message, "\n")

	maxLines := len(menuLines)
	if len(messageLines) > maxLines {
		maxLines = len(messageLines)
	}

	output := strings.Builder{}
	// Aggiunge un titolo
	output.WriteString("Applicazione Bubble Tea\n")
	output.WriteString(strings.Repeat("─", 30) + "\n\n")

	for i := 0; i < maxLines; i++ {
		menuLine := ""
		if i < len(menuLines) {
			menuLine = menuLines[i]
		}

		messageLine := ""
		if i < len(messageLines) {
			messageLine = messageLines[i]
		}

		// Formatta per avere due colonne.
		// `%-25s` alloca 25 caratteri per la colonna del menu, con allineamento a sinistra.
		output.WriteString(fmt.Sprintf("%-25s %s\n", menuLine, messageLine))
	}
	output.WriteString("\n")

	return output.String()
}

func main() {
	p := tea.NewProgram(model{
		choices: []string{"Saluta", "Mostra Data/Ora", "Numero Casuale", "Esci"},
		message: "Premi Enter per scegliere un'opzione o frecce per navigare.",
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Errore nell'esecuzione dell'applicazione: %v\n", err)
		os.Exit(1)
	}
}
