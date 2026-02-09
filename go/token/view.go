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
	if totalWidth < 60 {
		totalWidth = 60
	}
	listWidth := totalWidth / 2
	if listWidth < 30 {
		listWidth = 30
	}
	if listWidth > 60 {
		listWidth = 60
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
		listPanel.WriteString(titleStyle.Render(" PNGs ") + "\n\n")
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

	listBox := panel.Width(listWidth).Render(limitLines(listPanel.String(), bodyContentHeight))

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
	detailsBox := panel.Width(detailWidth).Render(limitLines(details.String(), bodyContentHeight))

	// Barra messaggi con hint contestuale
	message := m.message
	if message == "" {
		message = "Pronto."
	}
	messageBar := panel.Width(listWidth + detailWidth).Render(highlight.Render(" Msg ") + " " + message + "  " + dim.Render("[? help]"))

	// Layout finale
	body := lipgloss.JoinHorizontal(lipgloss.Top, listBox, detailsBox)
	if m.showHelp {
		var help strings.Builder
		help.WriteString(titleStyle.Render(" Help ") + "\n\n")
		help.WriteString("Tab: cambia pannello\n")
		help.WriteString("1/2: focus pannello\n")
		help.WriteString("q/Esc/Ctrl+C: esci\n")
		help.WriteString("n: nuovo PNG\n")
		help.WriteString("d/x/Backspace/Delete: elimina PNG\n")
		help.WriteString("r: reset token di tutti\n")
		help.WriteString("↑↓: seleziona PNG\n")
		help.WriteString("←→: token -/+\n")
		help.WriteString("?: mostra/nasconde help\n")
		helpBox := panel.Width(listWidth + detailWidth).Render(limitLines(help.String(), bodyContentHeight))
		return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, helpBox, messageBar)
	}
	return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, messageBar)
}
