package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ChatView(m Model) tea.View {
	inputBox := RenderInputBox(m)

	var header string
	if m.mode == modeQus {
		header = compactSplash(m)
	} else {
		header = RenderSplash(m)
	}

	var body string
	if m.mode == modeQus {
		qusView := renderQusView(m)
		body = lipgloss.JoinVertical(
			lipgloss.Top,
			m.viewPort.View(),
			qusView,
		)
	} else {
		body = m.viewPort.View()
	}

	content := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		body,
		inputBox,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func renderQusView(m Model) string {
	return lipgloss.NewStyle().
		Padding(0, 2).
		Render(m.qusList.View())
}

func RenderChatBubble(msg ChatMessage, m Model) string {
	color := cyanColor
	prefix := "You >"
	switch msg.Role {
	case "assistant":
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
