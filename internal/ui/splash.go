package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func splashView(m Model) tea.View {
	logoO := lipgloss.NewStyle().Foreground(whiteColor).Bold(true).Render(`
▄████▄
██  ██
▀████▀
`)

	logoC := lipgloss.NewStyle().Foreground(cyanColor).Bold(true).Render(`
 ▄▄▄▄
██▀▀▀
▀████
`)

	logo := lipgloss.JoinHorizontal(lipgloss.Top, logoO, " ", logoC)

	// Input box
	inputBox := RenderInputBox(m)

	content := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-1).
		Align(lipgloss.Center, lipgloss.Center).
		Render(logo + "\n\n" + inputBox + "\n")

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
