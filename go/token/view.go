package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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

func fitWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trimmed := ansi.Truncate(line, width, "")
		lines[i] = lipgloss.NewStyle().Width(width).MaxWidth(width).Render(trimmed)
	}
	return strings.Join(lines, "\n")
}

func clampWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(s)
}

func clampFinalWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
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
	helpBorder := lipgloss.Border{
		Top:         "═",
		Bottom:      "═",
		Left:        "║",
		Right:       "║",
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
	}
	helpPanel := lipgloss.NewStyle().Border(helpBorder).Padding(0, 1)
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	highlight := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	selectedPNGStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	totalWidth := m.width
	if totalWidth <= 0 {
		totalWidth = 96
	}
	listWidth := totalWidth / 3
	minPane := 20
	if listWidth < minPane {
		listWidth = minPane
	}
	detailWidth := totalWidth - listWidth
	if detailWidth < minPane {
		if totalWidth < minPane*2 {
			listWidth = totalWidth / 2
			if listWidth < 1 {
				listWidth = 1
			}
			detailWidth = totalWidth - listWidth
		} else {
			detailWidth = minPane
			listWidth = totalWidth - detailWidth
		}
	}
	listContentWidth := listWidth - 2
	if listContentWidth < 1 {
		listContentWidth = 1
	}
	detailContentWidth := detailWidth - 2
	if detailContentWidth < 1 {
		detailContentWidth = 1
	}

	totalHeight := m.height
	if totalHeight <= 0 {
		totalHeight = 24
	}
	// Header
	header := titleStyle.Render(" PNG Manager ")
	headerBar := panel.Width(listWidth + detailWidth - 2).Render(header)

	// Barra messaggi con hint contestuale (servirà per calcolare l'altezza del body)
	message := m.message
	if message == "" {
		message = "Pronto."
	}
	messageBar := panel.Width(listWidth + detailWidth - 2).Render(fitWidth(highlight.Render(" Msg ")+" "+message+"  "+dim.Render("?: help"), listWidth+detailWidth-2))

	bodyHeight := totalHeight - lipgloss.Height(headerBar) - lipgloss.Height(messageBar)
	if bodyHeight < 3 {
		bodyHeight = 3
	}

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

	leftBoxTotal1 := bodyHeight / 3
	leftBoxTotal2 := bodyHeight / 3
	leftBoxTotal3 := bodyHeight - leftBoxTotal1 - leftBoxTotal2
	if leftBoxTotal1 < 3 {
		leftBoxTotal1 = 3
	}
	if leftBoxTotal2 < 3 {
		leftBoxTotal2 = 3
	}
	if leftBoxTotal3 < 3 {
		leftBoxTotal3 = 3
	}
	leftContent1 := leftBoxTotal1 - 2
	leftContent2 := leftBoxTotal2 - 2
	leftContent3 := leftBoxTotal3 - 2
	if leftContent1 < 1 {
		leftContent1 = 1
	}
	if leftContent2 < 1 {
		leftContent2 = 1
	}
	if leftContent3 < 1 {
		leftContent3 = 1
	}

	listBox := panel.Width(listContentWidth).Height(leftContent1).Render(limitLines(fitWidth(listPanel.String(), listContentWidth), leftContent1))

	// Pannello dettagli (PNG o Mostri)
	var details strings.Builder
	modeLabel := "Full"
	if m.detailsCompact {
		modeLabel = "Compact"
	}
	details.WriteString(titleStyle.Render(" Dettagli ") + dim.Render(" ["+modeLabel+"]") + "\n\n")
	if m.encounterEditing {
		var modal strings.Builder
		lines := []string{
			"FERITE",
			"",
			m.encounterInput.View(),
			"",
			"Invio per confermare",
			"Esc per annullare",
		}
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Align(lipgloss.Center).Render("FERITE")
		lines[0] = title
		for i, line := range lines {
			if i == 0 {
				modal.WriteString(line + "\n\n")
				continue
			}
			if line == "Invio per confermare" || line == "Esc per annullare" {
				modal.WriteString(dim.Render(line) + "\n")
			} else {
				modal.WriteString(line + "\n")
			}
		}
		maxLine := 0
		for _, line := range lines {
			if w := lipgloss.Width(line); w > maxLine {
				maxLine = w
			}
		}
		modalWidth := maxLine + 4
		if modalWidth < 20 {
			modalWidth = 20
		}
		modalBox := helpPanel.Width(modalWidth).Render(strings.TrimRight(modal.String(), "\n"))
		details.WriteString(modalBox + "\n\n")
	}
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
			if !m.detailsCompact {
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
			mon := e.Monster
			basePF := e.BasePF
			if basePF == 0 {
				basePF = mon.PF
			}
			details.WriteString(fmt.Sprintf("Nome:  %s #%d\n", mon.Name, seen))
			details.WriteString(fmt.Sprintf("Ruolo: %s  Rango: %d\n", mon.Role, mon.Rank))
			details.WriteString(fmt.Sprintf("Difficoltà: %d\n", mon.Difficulty))
			if len(mon.Thresholds.Values) > 0 {
				details.WriteString(fmt.Sprintf("Soglie: %d/%d\n", mon.Thresholds.Values[0], mon.Thresholds.Values[len(mon.Thresholds.Values)-1]))
			} else if mon.Thresholds.Text != "" {
				details.WriteString(fmt.Sprintf("Soglie: %s\n", mon.Thresholds.Text))
			}
			pf := basePF - e.Wounds
			if pf < 0 {
				pf = 0
			}
			details.WriteString(fmt.Sprintf("PF: %d/%d  Stress: %d\n", pf, basePF, mon.Stress))
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
			if !m.detailsCompact {
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
			line := label
			basePF := e.BasePF
			if basePF == 0 {
				basePF = e.Monster.PF
			}
			if basePF > 0 {
				pf := basePF - e.Wounds
				if pf < 0 {
					pf = 0
				}
				line += fmt.Sprintf(" [%d/%d]", pf, basePF)
			}
			encounter.WriteString(prefix + line + "\n")
		}
	}
	encounterBox := panel.Width(listContentWidth).Height(leftContent2).Render(limitLines(fitWidth(encounter.String(), listContentWidth), leftContent2))

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
	monstersBox := panel.Width(listContentWidth).Height(leftContent3).Render(limitLines(fitWidth(monsters.String(), listContentWidth), leftContent3))

	listStack := lipgloss.JoinVertical(lipgloss.Left, listBox, encounterBox, monstersBox)
	detailsContentHeight := bodyHeight - 2
	if detailsContentHeight < 1 {
		detailsContentHeight = 1
	}
	detailsBox := panel.Width(detailContentWidth).Height(detailsContentHeight).Render(limitLines(fitWidth(details.String(), detailContentWidth), detailsContentHeight))

	// Layout finale
	body := lipgloss.JoinHorizontal(lipgloss.Top, listStack, detailsBox)
	if m.showHelp {
		var help strings.Builder
		helpTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Width(listWidth + detailWidth - 4).Align(lipgloss.Center).Render("HELP")
		help.WriteString(helpTitle + "\n\n")
		help.WriteString(m.helpFilter.View() + "\n\n")

		lines := []string{
			"Tab: cambia pannello",
			"1/2/3: focus pannello (PNGs/Incontro/Mostri)",
			"q/Esc/Ctrl+C: esci",
			"t: toggle dettagli compact/full",
			"?: mostra/nasconde help",
			"/: filtra help",
			"",
		}

		switch m.focusedPanel {
		case 0: // PNGs
			lines = append(lines,
				"PNGs:",
				"↑↓: seleziona PNG",
				"←→: token -/+",
				"n: nuovo PNG",
				"d/x/Backspace/Delete: elimina PNG",
				"r: reset token selezionato",
				"R: reset token di tutti",
			)
		case 1: // Incontro
			lines = append(lines,
				"Incontro:",
				"↑↓: seleziona mostro",
				"d/x/Backspace/Delete: rimuovi",
				"←→: aggiungi/togli ferite",
			)
		case 2: // Mostri
			lines = append(lines,
				"Mostri:",
				"digita per cercare",
				"↑↓: seleziona mostro",
				"a: aggiungi all'incontro",
				"Ctrl+O/Ctrl+I: cronologia indietro/avanti",
			)
		}

		filter := strings.ToLower(strings.TrimSpace(m.helpFilter.Value()))
		for _, line := range lines {
			if filter == "" || strings.Contains(strings.ToLower(line), filter) {
				help.WriteString(line + "\n")
			}
		}
		helpBox := helpPanel.Width(listWidth + detailWidth - 2).Render(limitLines(help.String(), totalHeight-4))
		return clampFinalWidth(helpBox, totalWidth)
	}
	return clampFinalWidth(lipgloss.JoinVertical(lipgloss.Left, headerBar, body, messageBar), totalWidth)
}
