package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var menuChoices = []string{
	"Crea PNG",
	"Crea PNG casuale",
	"Ricarica PNG da disco",
	"Salva PNG su disco",
	"Seleziona PNG",
	"Decrementa Contatore PNG",
	"Resetta Contatore PNG",
	"Resetta Tutti i Contatori PNG",
	"Esci",
}

func (m model) handleMenuChoice(choice string) (model, tea.Cmd) {
	switch choice {
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
			m.selectPNGCursor = 0
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
				m.pngs[i].Counter = defaultCounter
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

	return m, nil
}
