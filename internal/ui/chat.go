package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ChatView(m Model) tea.View {
	inputBox := RenderInputBox(m)
	inputHeight := lipgloss.Height(inputBox)

	vp := m.viewPort
	vp.SetWidth(m.width)
	vp.SetHeight(m.height - inputHeight)

	content := lipgloss.JoinVertical(lipgloss.Top, vp.View(), inputBox)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
