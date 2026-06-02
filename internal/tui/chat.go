package tui

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
	return tea.NewView(content)
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
	prompt := lipgloss.NewStyle().Foreground(cyanColor).Render("╰─ $")

	var inputLine string
	if m.loading {
		loading := lipgloss.NewStyle().Foreground(mutedColor).Render("…")
		inputLine = prompt + " " + loading + " " + m.inputText.View()
	} else {
		inputLine = prompt + " " + m.inputText.View()
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 2).
		Render(inputLine)
}
