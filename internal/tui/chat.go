package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) renderHeader() string {
	switch m.mode {
	case modeQus, modeSession, modeCmd:
		return compactSplash(m)
	default:
		if m.termHeight < 30 {
			return compactSplash(m)
		}
		return RenderSplash(m)
	}
}

func ChatView(m Model) tea.View {
	inputBox := RenderInputBox(m)
	header := m.renderHeader()

	var body string

	switch m.mode {
	case modeQus:
		body = lipgloss.JoinVertical(
			lipgloss.Top,
			m.viewPort.View(),
			renderQusView(m),
		)
	case modeSession:
		body = lipgloss.JoinVertical(
			lipgloss.Top,
			m.viewPort.View(),
			renderSessionView(m),
		)
	case modeCmd:
		body = lipgloss.JoinVertical(
			lipgloss.Top,
			m.viewPort.View(),
			renderCmdView(m),
		)
	default:
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
	if len(m.qusItems) == 0 {
		return ""
	}
	q := m.pendingControl.Data.Questions[m.currentQuestionIdx]
	title := q.Header + "  (" + q.Question + ")"

	var lines []string
	lines = append(lines, title)
	lines = append(lines, strings.Repeat("-", len(title)))

	for i, item := range m.qusItems {
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == m.qusCursor {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(orangeColor)
		}
		line := fmt.Sprintf("%s%-30s", prefix, item.label)
		if item.desc != "" {
			line += fmt.Sprintf(" (%s) %s", "option", item.desc)
		}
		lines = append(lines, style.Render(line))
	}

	return strings.Join(lines, "\n")
}

func renderCmdView(m Model) string {
	const itemsPerPage = 5
	cmds := filteredCmdList(m)
	total := len(cmds)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	start := m.cmdPage * itemsPerPage

	var lines []string
	lines = append(lines, "commands")
	lines = append(lines, strings.Repeat("-", 10))

	for i := 0; i < itemsPerPage; i++ {
		idx := start + i
		if idx >= total {
			lines = append(lines, "")
			continue
		}
		item := cmds[idx]
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == m.cmdCursor {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(orangeColor)
		}
		line := fmt.Sprintf("%s%-30s (%s) %s", prefix, item.Name, item.Category, item.Description)
		lines = append(lines, style.Render(line))
	}

	if totalPages > 1 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Page %d/%d", m.cmdPage+1, totalPages))
	}

	return strings.Join(lines, "\n")
}

func renderSessionView(m Model) string {
	const itemsPerPage = 5
	total := len(m.sessions)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	start := m.sessionPage * itemsPerPage

	var lines []string
	lines = append(lines, fmt.Sprintf("%d sessions", total))
	lines = append(lines, strings.Repeat("-", 12))

	for i := 0; i < itemsPerPage; i++ {
		idx := start + i
		if idx >= total {
			lines = append(lines, "")
			continue
		}
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == m.sessionCursor {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(orangeColor)
		}
		title := strings.ReplaceAll(m.sessions[idx].Title, "\n", " ")
		line := fmt.Sprintf("%s%02d  %-26s", prefix, idx+1, title)
		lines = append(lines, style.Render(line))
	}

	if totalPages > 1 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Page %d/%d", m.sessionPage+1, totalPages))
	}

	return strings.Join(lines, "\n")
}

func RenderChatBubble(msg ChatMessage, m Model) string {
	color := cyanColor
	prefix := "You >"
	switch msg.Role {
	case "assistant":
		color = whiteColor
		prefix = "oc >"
	case "permission":
		color = orangeColor
		prefix = "🔑"
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

func inputModeTag(m Model) string {
	label := " NORMAL "
	fg := lipgloss.Color("#888888")

	switch m.mode {
	case modeInsert:
		label = " INSERT"
		fg = cyanColor
	case modeVisual:
		label = " VISUAL "
		fg = orangeColor
	case modeQus:
		label = " QUESTION "
		fg = greenColor
	case modeSession:
		label = " SESSIONS "
		fg = cyanColor
	case modeCmd:
		label = " COMMANDS "
		fg = cyanColor
	case modePerm:
		label = " PERMISSION "
		fg = orangeColor
	}

	return lipgloss.NewStyle().
		Foreground(fg).
		Background(lipgloss.Color("#333333")).
		Padding(0, 0).
		Render(label)
}

func RenderInputBox(m Model) string {

	mode := inputModeTag(m)
	input := m.inputText.View()

	if m.loading {
		input = nextSpinner() + " thinking"
	}

	content := lipgloss.JoinHorizontal(lipgloss.Center, mode, " ", input)

	return lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.NormalBorder()).
		BorderLeft(false).BorderRight(false).
		Padding(0, 1).
		Render(content)
}
