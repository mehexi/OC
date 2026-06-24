package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func compactSplash(m Model) string {
	status := lipgloss.NewStyle().Foreground(mutedColor).Render("● Connected")
	if m.Server.healthStatus != nil && m.Server.healthStatus.Healthy {
		status = lipgloss.NewStyle().Foreground(greenColor).Render("● Connected  v" + m.Server.healthStatus.Version)
	} else {
		status = lipgloss.NewStyle().Foreground(orangeColor).Render("● Connecting...")
	}

	info := status
	if m.Server.modelName != "" {
		info = lipgloss.JoinHorizontal(lipgloss.Center, status, "  ", lipgloss.NewStyle().Foreground(whiteColor).Render(m.Server.modelName))
	}

	return lipgloss.NewStyle().
		Width(m.Layout.width).
		Padding(0, 2).
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Render(info)
}

func RenderSplash(m Model) string {
	var infoLines []string
	if m.Server.healthStatus != nil && m.Server.healthStatus.Healthy {
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(greenColor).Render("● Connected  v"+m.Server.healthStatus.Version))
		if m.Chat.multiAgent != nil && *m.Chat.multiAgent {
			label := "⚡ multi-agent"
			if m.Chat.agents > 0 {
				label += fmt.Sprintf(" (%d agents)", m.Chat.agents)
			}
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(orangeColor).Render(label))
			if m.Chat.complexity != "" {
				infoLines = append(infoLines, lipgloss.NewStyle().Foreground(mutedColor).Render(m.Chat.complexity))
			}
		}
		if m.Server.modelName != "" {
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(whiteColor).Render(m.Server.modelName))
		}
		if m.Server.currentPath != "" {
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(mutedColor).Render(m.Server.currentPath))
		}
		if m.Server.contextLimit > 0 {
			remaining := m.Server.contextLimit - m.Server.tokensUsed
			infoLines = append(infoLines, lipgloss.NewStyle().Foreground(mutedColor).Render(fmt.Sprintf("%d / %d tokens  %d remaining", m.Server.tokensUsed, m.Server.contextLimit, remaining)))
		}
	} else {
		infoLines = append(infoLines, lipgloss.NewStyle().Foreground(orangeColor).Render("● Connecting..."))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).
		Width(m.Layout.width).
		Padding(0, 2).
		Render(strings.Join(infoLines, "  |  "))
}
