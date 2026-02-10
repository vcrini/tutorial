package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	highlight := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	selectedPNGStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	totalWidth := m.width
	if totalWidth <= 0 {
		totalWidth = 96
	}
	if totalWidth < 80 {
		totalWidth = 80
	}
	listWidth := totalWidth / 3
	if listWidth < 26 {
		listWidth = 26
	}
	detailWidth := totalWidth - listWidth

	totalHeight := m.height
	if totalHeight <= 0 {
		totalHeight = 24
	}
	if totalHeight < 12 {
		totalHeight = 12
	}
	bodyContentHeight := maxInt(totalHeight-6, 4)

	// Header
	header := titleStyle.Render(" PNG Manager ")
	headerBar := panel.Width(listWidth + detailWidth).Render(header)

	// Pannello lista PNG
	var listPanel strings.Builder
	switch m.appState {
	case createPNGState:
		listPanel.WriteString(titleStyle.Render(" Nuovo PNG ") + "\n\n")
		listPanel.WriteString(m.message + "\n\n")
		listPanel.WriteString(m.textInput.View() + "\n\n")
		listPanel.WriteString(dim.Render("Enter: conferma  Esc/q: annulla"))
	default:
		listPanel.WriteString(titleStyle.Render(" [1]-PNGs ") + "\n\n")
		listPanel.WriteString("\n")
		if len(m.pngs) == 0 {
			listPanel.WriteString(dim.Render("Nessun PNG creato."))
		} else {
			for i, png := range m.pngs {
				line := fmt.Sprintf("%s (Token: %d)", png.Name, png.Token)
				if i == m.selectedPNGIndex {
					listPanel.WriteString(selectedPNGStyle.Render("• "+line) + "\n")
				} else {
					listPanel.WriteString("  " + line + "\n")
				}
			}
			if m.selectedPNGIndex == -1 {
				listPanel.WriteString("\n" + dim.Render("Nessun PNG selezionato."))
			}
		}
	}

	leftPanelHeight := bodyContentHeight / 3
	if leftPanelHeight < 4 {
		leftPanelHeight = 4
	}
	listBox := panel.Width(listWidth).Render(limitLines(listPanel.String(), leftPanelHeight))

	// Pannello dettagli (PNG o Mostri)
	var details strings.Builder
	details.WriteString(titleStyle.Render(" Dettagli ") + "\n\n")
	if m.focusedPanel == 2 {
		filtered := m.filteredMonsters()
		if len(filtered) == 0 {
			details.WriteString(dim.Render("Nessun mostro trovato."))
		} else {
			idx := m.monsterCursor
			if idx < 0 {
				idx = 0
			}
			if idx >= len(filtered) {
				idx = len(filtered) - 1
			}
			mon := filtered[idx]
			details.WriteString(fmt.Sprintf("Nome:  %s\n", mon.Name))
			details.WriteString(fmt.Sprintf("Ruolo: %s  Rango: %d\n", mon.Role, mon.Rank))
			details.WriteString(fmt.Sprintf("Difficoltà: %d\n", mon.Difficulty))
			if len(mon.Thresholds.Values) > 0 {
				details.WriteString(fmt.Sprintf("Soglie: %d/%d\n", mon.Thresholds.Values[0], mon.Thresholds.Values[len(mon.Thresholds.Values)-1]))
			} else if mon.Thresholds.Text != "" {
				details.WriteString(fmt.Sprintf("Soglie: %s\n", mon.Thresholds.Text))
			}
			details.WriteString(fmt.Sprintf("PF: %d  Stress: %d\n", mon.PF, mon.Stress))
			if mon.Attack.Name != "" {
				bonus := strings.TrimSpace(mon.Attack.Bonus)
				if bonus != "" && !strings.HasPrefix(bonus, "+") && !strings.HasPrefix(bonus, "-") {
					bonus = "+" + bonus
				}
				if bonus != "" {
					details.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s (%s)\n", mon.Attack.Name, mon.Attack.Range, mon.Attack.Damage, mon.Attack.DamageType, bonus))
				} else {
					details.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s\n", mon.Attack.Name, mon.Attack.Range, mon.Attack.Damage, mon.Attack.DamageType))
				}
			}
			if mon.Description != "" {
				details.WriteString("\n" + mon.Description + "\n")
			}
			if mon.MotivationsTactics != "" {
				details.WriteString("\nMotivazioni: " + mon.MotivationsTactics + "\n")
			}
			if len(mon.Traits) > 0 {
				details.WriteString("\nTratti:\n")
				for _, t := range mon.Traits {
					details.WriteString(fmt.Sprintf("- %s (%s): %s\n", t.Name, t.Kind, t.Text))
				}
			}
		}
	} else if m.focusedPanel == 1 {
		if len(m.encounter) == 0 {
			details.WriteString(dim.Render("Nessun mostro in incontro."))
		} else {
			idx := m.encounterCursor
			if idx < 0 {
				idx = 0
			}
			if idx >= len(m.encounter) {
				idx = len(m.encounter) - 1
			}
			e := m.encounter[idx]
			seen := 0
			for i := 0; i <= idx; i++ {
				if m.encounter[i].Monster.Name == e.Monster.Name {
					seen++
				}
			}
			details.WriteString(fmt.Sprintf("Nome:  %s #%d\n", e.Monster.Name, seen))
			details.WriteString(fmt.Sprintf("Ruolo: %s  Rango: %d\n", e.Monster.Role, e.Monster.Rank))
			details.WriteString(fmt.Sprintf("Difficoltà: %d\n", e.Monster.Difficulty))
		}
	} else {
		if m.selectedPNGIndex >= 0 && m.selectedPNGIndex < len(m.pngs) {
			png := m.pngs[m.selectedPNGIndex]
			details.WriteString(fmt.Sprintf("Nome:  %s\n", png.Name))
			details.WriteString(fmt.Sprintf("Token: %d\n", png.Token))
			details.WriteString(fmt.Sprintf("Index: %d/%d\n", m.selectedPNGIndex+1, len(m.pngs)))
		} else {
			details.WriteString(dim.Render("Seleziona un PNG per vedere i dettagli."))
		}
	}
	// Pannello incontro (sotto PNGs)
	var encounter strings.Builder
	encounter.WriteString(titleStyle.Render(" [2]-Incontro ") + "\n\n")
	if len(m.encounter) == 0 {
		encounter.WriteString(dim.Render("Nessun mostro in incontro."))
	} else {
		seen := map[string]int{}
		for i, e := range m.encounter {
			seen[e.Monster.Name]++
			label := fmt.Sprintf("%s #%d", e.Monster.Name, seen[e.Monster.Name])
			prefix := "  "
			if i == m.encounterCursor && m.focusedPanel == 1 {
				prefix = selectedPNGStyle.Render("•") + " "
			}
			encounter.WriteString(prefix + label + "\n")
		}
	}
	encounterBox := panel.Width(listWidth).Render(limitLines(encounter.String(), leftPanelHeight))

	// Pannello mostri (sotto Incontro)
	var monsters strings.Builder
	monsters.WriteString(titleStyle.Render(" [3]-Mostri ") + "\n\n")
	monsters.WriteString(m.monsterSearch.View() + "\n\n")
	filtered := m.filteredMonsters()
	if len(filtered) == 0 {
		monsters.WriteString(dim.Render("Nessun mostro trovato."))
	} else {
		for i, mon := range filtered {
			prefix := "  "
			if i == m.monsterCursor {
				prefix = selectedPNGStyle.Render("•") + " "
			}
			monsters.WriteString(prefix + mon.Name + "\n")
		}
		monsters.WriteString("\n")
		idx := m.monsterCursor
		if idx < 0 {
			idx = 0
		}
		if idx >= len(filtered) {
			idx = len(filtered) - 1
		}
		mon := filtered[idx]
		monsters.WriteString(dim.Render("Dettagli") + "\n")
		monsters.WriteString(fmt.Sprintf("Ruolo: %s  Rango: %d\n", mon.Role, mon.Rank))
		monsters.WriteString(fmt.Sprintf("Difficoltà: %d\n", mon.Difficulty))
		if len(mon.Thresholds.Values) > 0 {
			monsters.WriteString(fmt.Sprintf("Soglie: %d/%d\n", mon.Thresholds.Values[0], mon.Thresholds.Values[len(mon.Thresholds.Values)-1]))
		} else if mon.Thresholds.Text != "" {
			monsters.WriteString(fmt.Sprintf("Soglie: %s\n", mon.Thresholds.Text))
		}
		monsters.WriteString(fmt.Sprintf("PF: %d  Stress: %d\n", mon.PF, mon.Stress))
		if mon.Attack.Name != "" {
			bonus := strings.TrimSpace(mon.Attack.Bonus)
			if bonus != "" && !strings.HasPrefix(bonus, "+") && !strings.HasPrefix(bonus, "-") {
				bonus = "+" + bonus
			}
			if bonus != "" {
				monsters.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s (%s)\n", mon.Attack.Name, mon.Attack.Range, mon.Attack.Damage, mon.Attack.DamageType, bonus))
			} else {
				monsters.WriteString(fmt.Sprintf("Attacco: %s (%s) %s %s\n", mon.Attack.Name, mon.Attack.Range, mon.Attack.Damage, mon.Attack.DamageType))
			}
		}
	}
	monstersBox := panel.Width(listWidth).Render(limitLines(monsters.String(), leftPanelHeight))

	listStack := lipgloss.JoinVertical(lipgloss.Left, listBox, encounterBox, monstersBox)

	detailsBox := panel.Width(detailWidth).Render(limitLines(details.String(), bodyContentHeight*2))

	// Barra messaggi con hint contestuale
	message := m.message
	if message == "" {
		message = "Pronto."
	}
	messageBar := panel.Width(listWidth + detailWidth).Render(highlight.Render(" Msg ") + " " + message + "  " + dim.Render("?: help"))

	// Layout finale
	body := lipgloss.JoinHorizontal(lipgloss.Top, listStack, detailsBox)
	if m.showHelp {
		var help strings.Builder
		help.WriteString(titleStyle.Render(" Help ") + "\n\n")
		help.WriteString("Tab: cambia pannello\n")
		help.WriteString("1/2/3: focus pannello (PNGs/Incontro/Mostri)\n")
		help.WriteString("q/Esc/Ctrl+C: esci\n")
		help.WriteString("n: nuovo PNG\n")
		help.WriteString("d/x/Backspace/Delete: elimina PNG\n")
		help.WriteString("r: reset token di tutti\n")
		help.WriteString("↑↓: seleziona PNG\n")
		help.WriteString("←→: token -/+\n")
		help.WriteString("Mostri: digita per cercare, ↑↓ per selezionare, a: aggiungi\n")
		help.WriteString("Incontro: d/x/backspace: rimuovi\n")
		help.WriteString("?: mostra/nasconde help\n")
		helpBox := panel.Width(listWidth + detailWidth).Render(limitLines(help.String(), bodyContentHeight))
		return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, helpBox, messageBar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, messageBar)
}
