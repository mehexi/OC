package ui

import "charm.land/lipgloss/v2"

func RenderChatBubble(text string, m Model) string {
	return lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(cyanColor).
		Background(chatBgColor).
		Padding(1, 1).
		Render(text)
}

func RenderInputBox(m Model) string {
	width := m.width
	if m.isSplashScreen {
		width = m.width * 8 / 10
	}

	inputBox := lipgloss.NewStyle().
		Width(width).
		Padding(1, 0).
		Background(inputBgColor).
		BorderStyle(lipgloss.ThickBorder()).
		BorderLeft(true).
		BorderForeground(cyanColor).
		Foreground(cyanColor).
		Render(m.inputText.View())

	return inputBox
}
