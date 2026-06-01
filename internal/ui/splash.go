package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func splashView(m Model) tea.View {

	inputBox := RenderInputBox(m)

	content := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-1).
		Align(lipgloss.Center, lipgloss.Center).
		Render(inputBox)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
