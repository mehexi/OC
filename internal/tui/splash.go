package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func RenderSplash(m Model) string {

	logoIcon := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Render(`
⠀⠀⠀⠀⠀⠀⣀⣤⡤⠀⠀⠀
⠀⠀⠀⠀⢀⣾⣿⠋⠀
⠀⠀⠀⣠⣾⣿⡟⠀⠀
⠀⠀⢸⠛⠉⢹⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡠⠄⠠⣀⠀⠀
⠀⠀⡘⠀⠀⠀⡀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠖⠉⠀⠀⠀⣾⣿⣦
⠀⠀⡇⠀⠀⠀⢡⠄⠀⠀⣀⣀⣀⣠⠊⠀⠀⠀⠀⡠⠞⠛⠛⠛
⠀⠀⢃⠀⠀⠀⠀⠗⠚⠉⠉⠀⠈⠁⠀⠀⠀⢀⡔⠁⠀
⠀⠀⠸⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣴⣶⣄⠲⡎⠀⠀
⠀⠀⠀⠃⠀⠀⢠⣤⡀⠀⠀⠀⠀⣿⣿⣿⠀⠘⡄
⠀⠀⠀⡆⠀⠀⣿⣿⡇⠀⠀⠀⠀⠈⠛⠉⣴⣆⢹⡄⠀⠀⠀⠀⠀
⠀⠀⠀⣇⢰⡧⣉⡉⠀⠀⢀⡀⠀⣀⣀⣠⣿⡷⢠⡇⠀⠀⠀⠀⠀
⠀⠀⠀⢻⠘⠃⠈⠻⢦⠞⠋⠙⠺⠋⠉⠉⠉⢡⠟⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠳⢄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠋
`)

	tips := []string{
		"  • Use /sessions to browse history",
		"  • Use /load <n> to resume a past session",
	}

	var tipLines []string
	for _, t := range tips {
		tipLines = append(tipLines, lipgloss.NewStyle().Foreground(mutedColor).Render(t))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		logoIcon,
		strings.Join(tipLines, "\n"),
	)

	splash := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Width(m.width).
		Align(lipgloss.Left).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left,
		splash,
	)
}
