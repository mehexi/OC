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

	info := status
	if m.modelName != "" {
		info = lipgloss.JoinHorizontal(lipgloss.Center, status, "  ", lipgloss.NewStyle().Foreground(whiteColor).Render(m.modelName))
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 2).
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Render(info)
}

func RenderSplash(m Model) string {
	var infoLines []string
	if m.healthStatus != nil && m.healthStatus.Healthy {
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(greenColor).Render("● Connected  v"+m.healthStatus.Version))
		if m.multiAgent != nil && *m.multiAgent {
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(orangeColor).Render("⚡ multi-agent"))
		}
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

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Width(m.width).
		Padding(0, 2).
		Render(strings.Join(infoLines, "  |  "))
}
