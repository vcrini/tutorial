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
	bodyContentHeight := maxInt(totalHeight-6, 4)

	// Header
	header := titleStyle.Render(" PNG Manager ") + dim.Render(" • Lazy style UI")
	headerBar := panel.Width(leftWidth + rightWidth).Render(header)

	// Menu (sinistra)
	var menu strings.Builder
	menu.WriteString(titleStyle.Render(" Comandi ") + "\n\n")
	menu.WriteString(dim.Render("1=Comandi  2=PNGs  Tab=Switch  q/Esc/Ctrl+C=Esci") + "\n")
	menu.WriteString(dim.Render("Enter: esegui  ↑↓: naviga") + "\n\n")
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
		rightTop.WriteString(dim.Render("Enter: conferma  Esc/q: annulla"))
	default:
		rightTop.WriteString(titleStyle.Render(" PNGs ") + "\n\n")
		rightTop.WriteString(dim.Render("↑↓: seleziona PNG  ←→: token -/+  n: nuovo  r: reset all  d/x: elimina  (focus su PNGs)") + "\n\n")
		if len(m.pngs) == 0 {
			rightTop.WriteString(dim.Render("Nessun PNG creato."))
		} else {
			for i, png := range m.pngs {
				line := fmt.Sprintf("%s (Token: %d)", png.Name, png.Token)
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

	// Barra messaggi con hint contestuale
	message := m.message
	if message == "" {
		message = "Pronto."
	}
	helpText := "1/2/Tab focus  •  q/Esc/Ctrl+C esci  •  Enter esegui  •  ↑↓ menu"
	if m.appState == createPNGState {
		helpText = "Enter conferma  •  Esc/q annulla"
	} else if m.focusedPanel == 1 {
		helpText = "↑↓ seleziona PNG  •  ←→ token  •  n nuovo  •  r reset all  •  d/x elimina  •  1 menu  2 PNGs"
	}
	if totalWidth < 80 {
		if m.appState == createPNGState {
			helpText = "Enter conferma  •  Esc/q annulla"
		} else if m.focusedPanel == 1 {
			helpText = "↑↓ seleziona  •  ←→ token  •  n nuovo  •  r reset all  •  d/x elimina  •  1 menu  2 PNGs"
		} else {
			helpText = "1/2/Tab focus  •  q/Esc esci  •  Enter"
		}
	}
	messageBar := panel.Width(leftWidth + rightWidth).Render(highlight.Render(" Msg ") + " " + message + "  " + dim.Render("["+helpText+"]"))

	// Layout finale
	body := lipgloss.JoinHorizontal(lipgloss.Top, menuPanel, rightTopPanel)
	return lipgloss.JoinVertical(lipgloss.Left, headerBar, body, messageBar)
}
