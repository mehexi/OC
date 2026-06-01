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

	if m.isSplashScreen == true {
		width = m.width / 2
	}

	inputContent := lipgloss.NewStyle().
		Foreground(cyanColor).
		Render(m.inputText.View())

	inputBox := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Background(bgColor).
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(cyanColor).
		Render(inputContent)

	return inputBox
}
