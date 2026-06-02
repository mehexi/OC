package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ChatView(m Model) tea.View {
	inputBox := RenderInputBox(m)
	welcomeMsg := RenderSplash(m)

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		welcomeMsg,
		m.viewPort.View(),
		inputBox,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func RenderChatBubble(msg ChatMessage, m Model) string {
	color := cyanColor
	prefix := "You >"
	if msg.Role == "assistant" {
		color = whiteColor
		prefix = "oc >"
	}

	prefixRendered := lipgloss.NewStyle().Foreground(color).Render(prefix)
	rendered := RenderMarkdown(msg.Content, m.width-8)

	body := lipgloss.NewStyle().
		PaddingLeft(2).
		PaddingRight(2).
		Width(m.width - 4).
		Render(prefixRendered + " " + rendered)

	return body
}

func RenderInputBox(m Model) string {
	input := m.inputText.View()

	if m.loading {
		input = "… " + input
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.NormalBorder()).
		BorderLeft(false).BorderRight(false).
		Padding(0, 1).
		Render(
			input,
		)
}
