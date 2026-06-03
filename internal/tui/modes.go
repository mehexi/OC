package tui

import (
	"fmt"
	"oc/internal/server"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
)

func (m Model) onNormalKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		m.mode = modeInsert
		m.inputText.Focus()
		s := m.inputText.Styles()
		s.Cursor.Color = cyanColor
		s.Cursor.Blink = false
		m.inputText.SetStyles(s)
		return m, nil

	case "j":
		m.viewPort.ScrollDown(3)
		return m, nil

	case "k":
		m.viewPort.ScrollUp(3)
		return m, nil

	case "g":
		if m.awaitingGG {
			m.awaitingGG = false
			m.viewPort.GotoTop()
		} else {
			m.awaitingGG = true
		}
		return m, nil

	case "G":
		m.awaitingGG = false
		m.viewPort.GotoBottom()
		return m, nil

	case "V":
		m.awaitingGG = false
		if len(m.messages) == 0 {
			return m, nil
		}
		m.mode = modeVisual
		last := len(m.messages) - 1
		m.visualAnchor = last
		m.visualCursor = last
		return m, nil

	case "/":
		m.awaitingGG = false
		m.inputText.SetValue("/")
		m.inputText.SetCursor(len("/"))
		m.mode = modeInsert
		m.inputText.Focus()
		return m, nil

	case "enter":
		m.awaitingGG = false
		input := m.inputText.Value()
		if input == "" {
			return m, nil
		}
		if !m.loading && !m.streaming {
			if strings.HasPrefix(input, "/") {
				m.inputText.SetValue("")
				parts := strings.Fields(input)
				if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
					m.sessionId = ""
					m.messages = nil
					m = m.refreshMessages()
					return m, m.addAssistantMsg("Started a new session.")
				}
				return m, m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			m = m.refreshMessages()
			m.inputText.SetValue("")
			m.loading = true
			return m, m.sendChat(input)
		}
		return m, nil

	case "ctrl+c":
		if m.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.inputText.SetValue("")
		return m, nil

	default:
		m.awaitingGG = false
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		return m, vpCmd
	}
}

func (m Model) onInsertKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.inputText.Blur()
		return m, nil

	case "enter":
		input := m.inputText.Value()
		if input == "" {
			return m, nil
		}
		if !m.loading && !m.streaming {
			if strings.HasPrefix(input, "/") {
				m.inputText.SetValue("")
				parts := strings.Fields(input)
				if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
					m.sessionId = ""
					m.messages = nil
					m = m.refreshMessages()
					return m, m.addAssistantMsg("Started a new session.")
				}
				return m, m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			m = m.refreshMessages()
			m.inputText.SetValue("")
			m.loading = true
			return m, m.sendChat(input)
		}
		return m, nil

	case "ctrl+c":
		if m.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.inputText.SetValue("")
		return m, nil

	default:
		var cmd tea.Cmd
		m.inputText, cmd = m.inputText.Update(msg)
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		return m, tea.Batch(cmd, vpCmd)
	}
}

func (m Model) onVisualKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j":
		if m.visualCursor < len(m.messages)-1 {
			m.visualCursor++
			m.viewPort.ScrollDown(3)
		}
		return m, nil

	case "k":
		if m.visualCursor > 0 {
			m.visualCursor--
			m.viewPort.ScrollUp(3)
		}
		return m, nil

	case "y":
		lo, hi := m.visualAnchor, m.visualCursor
		if lo > hi {
			lo, hi = hi, lo
		}
		var b strings.Builder
		for i := lo; i <= hi && i < len(m.messages); i++ {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(m.messages[i].Content)
		}
		text := b.String()
		if text != "" {
			clipboard.WriteAll(text)
		}
		m.mode = modeNormal
		m.inputText.SetValue(fmt.Sprintf("Yanked %d message(s)", hi-lo+1))
		m.inputText.SetCursor(len(m.inputText.Value()))
		return m, nil

	case "esc":
		m.mode = modeNormal
		return m, nil

	default:
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		return m, vpCmd
	}
}

func (m Model) showQusList() Model {
	q := m.pendingControl.Data.Questions[m.currentQuestionIdx]
	items := make([]list.Item, len(q.Options))
	for i, opt := range q.Options {
		items[i] = qusItem{label: opt.Label, desc: opt.Description}
	}

	del := list.NewDefaultDelegate()
	del.SetHeight(1)
	del.SetSpacing(0)
	del.ShowDescription = false
	del.Styles.NormalTitle = lipgloss.NewStyle().Foreground(whiteColor).Padding(0, 2, 0, 2)
	del.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(cyanColor).Bold(true).Padding(0, 2, 0, 2)
	del.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(mutedColor).Padding(0, 2, 0, 2)

	m.qusList.SetDelegate(del)
	m.qusList.SetItems(items)
	m.qusList.Title = q.Header + "  (" + q.Question + ")"
	m.qusList.SetFilteringEnabled(false)
	m.qusList.SetShowTitle(true)
	m.qusList.SetShowPagination(false)
	m.qusList.SetShowHelp(false)
	m.qusList.SetShowStatusBar(false)
	m.qusList.KeyMap.Quit.SetEnabled(false)
	m.qusList.Select(0)
	m.qusList.Styles.Title = lipgloss.NewStyle().Foreground(cyanColor).Bold(true)
	m.qusList.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 2).Border(lipgloss.NormalBorder()).BorderBottom(true).BorderLeft(false).BorderRight(false).BorderTop(false).BorderForeground(cyanColor)

	const compactHeaderHeight = 3
	available := m.termHeight - compactHeaderHeight - inputBoxHeight
	qusHeight := min(len(items)+2, available-3)
	m.qusList.SetSize(m.width, qusHeight)
	m.qusHeight = qusHeight
	m.viewPort.SetHeight(available - qusHeight)

	m.mode = modeQus
	m.inputText.Blur()
	return m
}

func (m Model) handleQusAnswer() (Model, tea.Cmd) {
	selected := m.qusList.SelectedItem().(qusItem)
	answer := selected.label

	m.questionAnswers = append(m.questionAnswers, answer)
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: answer})
	m = m.refreshMessages()

	m.currentQuestionIdx++
	if m.currentQuestionIdx < len(m.pendingControl.Data.Questions) {
		return m.showQusList(), nil
	}

	m.awaitingResponse = false
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
	m.qusHeight = 0
	m.loading = true
	return m, m.sendControlResponse()
}

func (m Model) handleQusCancel() (Model, tea.Cmd) {
	m.pendingControl = nil
	m.currentQuestionIdx = 0
	m.questionAnswers = nil
	m.awaitingResponse = false
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
	m.qusHeight = 0
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Question cancelled."})
	return m.refreshMessages(), nil
}

func (m Model) onQusKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.qusList, _ = m.qusList.Update(msg)
		return m, nil
	case "k", "up":
		m.qusList, _ = m.qusList.Update(msg)
		return m, nil
	case "enter":
		selected := m.qusList.SelectedItem()
		if selected == nil {
			return m, nil
		}
		return m.handleQusAnswer()
	case "esc", "ctrl+c":
		return m.handleQusCancel()
	default:
		var cmd tea.Cmd
		m.qusList, cmd = m.qusList.Update(msg)
		return m, cmd
	}
}
