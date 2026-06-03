package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func compactSplash(m Model) string {
	status := lipgloss.NewStyle().Foreground(mutedColor).Render("● Connected")
	if m.healthStatus != nil && m.healthStatus.Healthy {
		status = lipgloss.NewStyle().Foreground(greenColor).Render("● Connected  v" + m.healthStatus.Version)
	} else {
		status = lipgloss.NewStyle().Foreground(orangeColor).Render("● Connecting...")
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		modeTag(m),
		lipgloss.NewStyle().Width(m.width).Padding(0, 2).Render(status),
	)
}

func modeTag(m Model) string {
	label := "  NORMAL  "
	fg := lipgloss.Color("#888888")

	switch m.mode {
	case modeInsert:
		label = "  INSERT  iiii"
		fg = cyanColor
	case modeVisual:
		label = "  VISUAL  "
		fg = orangeColor
	case modeQus:
		label = "  QUESTION  "
		fg = greenColor
	}

	return lipgloss.NewStyle().
		Foreground(fg).
		Width(m.width).
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Render(label)
}

func RenderSplash(m Model) string {
	logoIcon := lipgloss.NewStyle().Render(`
⣡⣴⣶⣶⡀⠄⠄⠙⢿⣿⣿⣿⣿⣿⣴⣿⣿⣿⢃⣤⣄⣀⣥⣿
⢸⣇⠻⣿⣿⣿⣧⣀⢀⣠⡌⢻⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠿⣿⣿
⢸⣿⣷⣤⣤⣤⣬⣙⣛⢿⣿⣿⣿⣿⣿⣿⡿⣿⣿⡍⠄⠄⢀⣤⣄⠉
⣖⣿⣿⣿⣿⣿⣿⣿⣿⣿⢿⣿⣿⣿⣿⣿⢇⣿⣿⡷⠶⠶⢿⣿⣿⠇⢀
⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣽⣿⣿⣿⡇⣿⣿⣿⣿⣿⣿⣷⣶⣥⣴
⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣦⣌⣛⣻⣿⣿⣧⠙⠛⠛⡭⠅⠒⠦⠭⣭⡻⣿⣿⣿⣿⣿⣿⣿⣿⡿⠃⠄
⣿⣿⣿⣿⣿⣿⣿⡆⠄⠄⠄⠄⠄⠄⠄⠄⠹⠈⢋⣽⣿⣿⣿⣿⣵⣾
⣿⣿⣿⣿⣿⣿⣿⣿⠄⣴⣿⣶⣄⠄⣴⣶⠄⢀⣾⣿⣿⣿⣿⣿⣿⠃⠄⠄
⠈⠻⣿⣿⣿⣿⣿⣿⡄⢻⣿⣿⣿⠄⣿⣿⡀⣾⣿⣿⣿⣿⣛⠛⠁
⠄⠄⠈⠛⢿⣿⣿⣿⠁⠞⢿⣿⣿⡄⢿⣿⡇⣸⣿⣿⠿⠛⠁⠄
⠄⠄⠄⠄⠄⠉⠻⣿⣿⣾⣦⡙⠻⣷⣾⣿⠃⠿⠋⠁⠄
`)

	borderedLogo := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderRight(true).BorderLeft(false).BorderTop(false).BorderBottom(false).
		Render(logoIcon)

	var infoLines []string
	if m.healthStatus != nil && m.healthStatus.Healthy {
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(greenColor).Render("● Connected  v"+m.healthStatus.Version))
		if m.modelName != "" {
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(whiteColor).Render(m.modelName))
		}
		if m.currentPath != "" {
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(mutedColor).Render(m.currentPath))
		}
		if m.contextLimit > 0 {
			remaining := m.contextLimit - m.tokensUsed
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(mutedColor).Render(fmt.Sprintf("%d / %d tokens  %d remaining", m.tokensUsed, m.contextLimit, remaining)))
		}
	} else {
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(orangeColor).Render("● Connecting..."))
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		borderedLogo,
		lipgloss.NewStyle().PaddingLeft(2).Render(strings.Join(infoLines, "\n")),
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		modeTag(m),
		body,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Width(m.width).
		Align(lipgloss.Left).
		Render(content)
}
