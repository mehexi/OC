package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func renderModelView(m Model) string {
	if len(m.models) == 0 {
		return "Loading models..."
	}
	const itemsPerPage = 5
	models := filteredModelList(m)
	total := len(models)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	start := m.modelPage * itemsPerPage

	var lines []string
	lines = append(lines, fmt.Sprintf("%d models", total))
	lines = append(lines, strings.Repeat("-", 10))

	for i := range itemsPerPage {
		idx := start + i
		if idx >= total {
			lines = append(lines, "")
			continue
		}
		item := models[idx]
		prefix := "  "
		style := lipgloss.NewStyle()
		if i == m.modelCursor {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(orangeColor)
		}
		costStr := ""
		if item.CostInput > 0 || item.CostOutput > 0 {
			costStr = fmt.Sprintf(" $%.2f/%.2f", item.CostInput, item.CostOutput)
		}
		line := fmt.Sprintf("%s%-27s (%s)%s", prefix, item.Name, item.ProviderID, costStr)
		lines = append(lines, style.Render(line))
	}

	if totalPages > 1 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Page %d/%d", m.modelPage+1, totalPages))
	}

	return strings.Join(lines, "\n")
}

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
	case modeModel:
		body = lipgloss.JoinVertical(
			lipgloss.Top,
			m.viewPort.View(),
			renderModelView(m),
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

	for i := range itemsPerPage {
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

	for i := range itemsPerPage {
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

	tagColor := cyanColor
	tag := "you >"
	switch msg.Role {
	case RoleAssistant:
		tagColor = mutedColor
		tag = "oc >"
	case RolePermission:
		tagColor = orangeColor
		tag = "perm"
	}

	tagRendered := lipgloss.NewStyle().
		Foreground(tagColor).
		Render(tag)

	var body string

	content := msg.Content

	if msg.Role == RoleSystem {
		style := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		return style.Render("──── " + content + " ────")
	}

	if msg.Role == RoleJudge {
		boxWidth := m.width - 6
		judgeStyle := lipgloss.NewStyle().
			Foreground(orangeColor).
			Bold(true)
		bordered := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(orangeColor).
			Padding(0, 1).
			Width(boxWidth)
		judgeBlock := bordered.Render(judgeStyle.Render("⚖️  " + content))
		return judgeBlock
	}

	if msg.Reasoning != "" {
		boxWidth := m.width - 6
		reasoningStyle := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)
		bordered := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1).
			Width(boxWidth)
		body = bordered.Render(reasoningStyle.Render("💭 "+msg.Reasoning)) + "\n\n"
	}

	rendered := RenderMarkdown(content)
	body += lipgloss.JoinHorizontal(lipgloss.Top, tagRendered, " ", lipgloss.NewStyle().Render(rendered))
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
