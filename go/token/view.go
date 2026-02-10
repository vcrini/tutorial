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
	header := titleStyle.Render(" PNG Manager ") + dim.Render(" • Lazy style UI")
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
		listPanel.WriteString(dim.Render("?") + "\n\n")
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

	leftPanelHeight := bodyContentHeight / 2
	if leftPanelHeight < 4 {
		leftPanelHeight = 4
	}
	listBox := panel.Width(listWidth).Render(limitLines(listPanel.String(), leftPanelHeight))

	// Pannello dettagli PNG
	var details strings.Builder
	details.WriteString(titleStyle.Render(" Dettagli ") + "\n\n")
	if m.selectedPNGIndex >= 0 && m.selectedPNGIndex < len(m.pngs) {
		png := m.pngs[m.selectedPNGIndex]
		details.WriteString(fmt.Sprintf("Nome:  %s\n", png.Name))
		details.WriteString(fmt.Sprintf("Token: %d\n", png.Token))
		details.WriteString(fmt.Sprintf("Index: %d/%d\n", m.selectedPNGIndex+1, len(m.pngs)))
	} else {
		details.WriteString(dim.Render("Seleziona un PNG per vedere i dettagli."))
	}
	// Pannello mostri (sotto PNGs)
	var monsters strings.Builder
	monsters.WriteString(titleStyle.Render(" [2]-Mostri ") + "\n\n")
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

	listStack := lipgloss.JoinVertical(lipgloss.Left, listBox, monstersBox)

	detailsBox := panel.Width(detailWidth).Render(limitLines(details.String(), bodyContentHeight*2))

	// Barra messaggi con hint contestuale
	message := m.message
	if message == "" {
		message = "Pronto."
	}
	messageBar := panel.Width(listWidth + detailWidth).Render(highlight.Render(" Msg ") + " " + message + "  " + dim.Render("[? help]"))

	// Layout finale
	body := lipgloss.JoinHorizontal(lipgloss.Top, listStack, detailsBox)
	if m.showHelp {
		var help strings.Builder
		help.WriteString(titleStyle.Render(" Help ") + "\n\n")
		help.WriteString("Tab: cambia pannello\n")
		help.WriteString("1/2: focus pannello (PNGs/Mostri)\n")
		help.WriteString("q/Esc/Ctrl+C: esci\n")
		help.WriteString("n: nuovo PNG\n")
		help.WriteString("d/x/Backspace/Delete: elimina PNG\n")
		help.WriteString("r: reset token di tutti\n")
		help.WriteString("↑↓: seleziona PNG\n")
		help.WriteString("←→: token -/+\n")
		help.WriteString("Mostri: digita per cercare, ↑↓ per selezionare\n")
		help.WriteString("?: mostra/nasconde help\n")
		helpBox := panel.Width(listWidth + detailWidth).Render(limitLines(help.String(), bodyContentHeight))
		return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, helpBox, messageBar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, messageBar)
}
