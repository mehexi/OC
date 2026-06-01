package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ChatView(m Model) tea.View {
	chatBubble := RenderChatBubble(m.inputText.Value(), m)
	inputBox := RenderInputBox(m)

	content := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - 1).
		Render(chatBubble + "\n\n" + chatBubble + "\n" + inputBox)

	v := tea.NewView(content)
	v.AltScreen = true

	return v
}
