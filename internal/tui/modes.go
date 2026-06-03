package tui

import (
	"fmt"
	"oc/internal/history"
	"oc/internal/server"
	"strings"

	tea "charm.land/bubbletea/v2"
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
		m.inputText.Focus()
		return m.showCmdList(), nil

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
			if m.sessionId != "" {
				history.AppendMessage(m.sessionId, "user", input)
			}
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
				if input == "/" {
					return m.showCmdList(), nil
				}
				return m, m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			if m.sessionId != "" {
				history.AppendMessage(m.sessionId, "user", input)
			}
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
		if strings.HasPrefix(m.inputText.Value(), "/") {
			return m.showCmdList(), tea.Batch(cmd, vpCmd)
		}
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
	m.qusItems = make([]qusItem, len(q.Options))
	for i, opt := range q.Options {
		m.qusItems[i] = qusItem{label: opt.Label, desc: opt.Description}
	}
	m.qusCursor = 0

	const compactHeaderHeight = 3
	available := m.termHeight - compactHeaderHeight - inputBoxHeight
	qusHeight := 2 + len(m.qusItems)
	if qusHeight > available-3 {
		qusHeight = available - 3
	}
	m.qusHeight = qusHeight
	m.viewPort.SetHeight(available - qusHeight)

	m.mode = modeQus
	m.inputText.Blur()
	return m
}

func (m Model) handleQusAnswer() (Model, tea.Cmd) {
	if m.qusCursor >= len(m.qusItems) {
		return m, nil
	}
	answer := m.qusItems[m.qusCursor].label

	m.questionAnswers = append(m.questionAnswers, answer)
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: answer})
	m = m.refreshMessages()

	m.currentQuestionIdx++
	if m.currentQuestionIdx < len(m.pendingControl.Data.Questions) {
		return m.showQusList(), nil
	}

	m.awaitingResponse = false
	m.qusItems = nil
	m.qusCursor = 0
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
	m.qusItems = nil
	m.qusCursor = 0
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
		if m.qusCursor < len(m.qusItems)-1 {
			m.qusCursor++
		}
		return m, nil
	case "k", "up":
		if m.qusCursor > 0 {
			m.qusCursor--
		}
		return m, nil
	case "enter":
		if len(m.qusItems) == 0 {
			return m, nil
		}
		return m.handleQusAnswer()
	case "esc", "ctrl+c":
		return m.handleQusCancel()
	default:
		return m, nil
	}
}

func (m Model) showSessionList() Model {
	sessions, err := history.ListSessions()
	if err != nil || len(sessions) == 0 {
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "No past sessions."})
		m = m.refreshMessages()
		return m
	}
	m.sessions = sessions
	m.sessionPage = 0
	m.sessionCursor = 0

	const compactHeaderHeight = 3
	itemsPerPage := 5
	total := len(sessions)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	sessionLines := 2 + itemsPerPage
	if totalPages > 1 {
		sessionLines += 2
	}
	available := m.termHeight - compactHeaderHeight - inputBoxHeight
	m.viewPort.SetHeight(available - sessionLines)

	m.mode = modeSession
	m.inputText.Blur()
	return m
}

func (m Model) onSessionKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	itemsPerPage := 5
	total := len(m.sessions)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	itemsOnPage := itemsPerPage
	start := m.sessionPage * itemsPerPage
	if end := start + itemsPerPage; end > total {
		itemsOnPage = total - start
	}

	switch msg.String() {
	case "j", "down":
		if m.sessionCursor < itemsOnPage-1 {
			m.sessionCursor++
		} else if m.sessionPage < totalPages-1 {
			m.sessionPage++
			m.sessionCursor = 0
		}
		return m, nil
	case "k", "up":
		if m.sessionCursor > 0 {
			m.sessionCursor--
		} else if m.sessionPage > 0 {
			m.sessionPage--
			itemsOnPrev := itemsPerPage
			if end := m.sessionPage*itemsPerPage + itemsPerPage; end > total {
				itemsOnPrev = total - m.sessionPage*itemsPerPage
			}
			m.sessionCursor = itemsOnPrev - 1
		}
		return m, nil
	case "enter":
		idx := m.sessionPage*itemsPerPage + m.sessionCursor
		if idx >= total {
			return m, nil
		}
		selected := m.sessions[idx]
		m.mode = modeInsert
		m.inputText.Focus()
		m.inputText.Placeholder = "Ask anything ..."
		m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
		return m, func() tea.Msg {
			s, err := history.LoadSession(selected.ID)
			if err != nil {
				return ChatResponseMsg{Err: fmt.Errorf("load session: %w", err)}
			}
			return LoadSessionMsg{Session: s}
		}
	case "esc", "ctrl+c":
		return m.handleSessionCancel()
	}
	return m, nil
}

func (m Model) handleSessionCancel() (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
	return m, nil
}

var cmdList = []cmdItem{
	{Name: "/sessions", Category: "history", Description: "List and load past sessions"},
	{Name: "/session new", Category: "history", Description: "Start a fresh session"},
	{Name: "/load <n>", Category: "history", Description: "Load session by number"},
}

func (m Model) showCmdList() Model {
	m.cmdPage = 0
	m.cmdCursor = 0

	const compactHeaderHeight = 3
	const itemsPerPage = 5
	total := len(cmdList)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	cmdLines := 2 + itemsPerPage
	if totalPages > 1 {
		cmdLines += 2
	}
	available := m.termHeight - compactHeaderHeight - inputBoxHeight
	m.viewPort.SetHeight(available - cmdLines)

	m.mode = modeCmd
	return m
}

func (m Model) onCmdKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	const itemsPerPage = 5
	total := len(cmdList)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	itemsOnPage := itemsPerPage
	start := m.cmdPage * itemsPerPage
	if end := start + itemsPerPage; end > total {
		itemsOnPage = total - start
	}

	switch msg.String() {
	case "j", "down":
		if m.cmdCursor < itemsOnPage-1 {
			m.cmdCursor++
		} else if m.cmdPage < totalPages-1 {
			m.cmdPage++
			m.cmdCursor = 0
		}
		return m, nil
	case "k", "up":
		if m.cmdCursor > 0 {
			m.cmdCursor--
		} else if m.cmdPage > 0 {
			m.cmdPage--
			itemsOnPrev := itemsPerPage
			if end := m.cmdPage*itemsPerPage + itemsPerPage; end > total {
				itemsOnPrev = total - m.cmdPage*itemsPerPage
			}
			m.cmdCursor = itemsOnPrev - 1
		}
		return m, nil
	case "enter":
		input := m.inputText.Value()
		if input != "/" && input != "" {
			return m.executeCommand(input)
		}
		idx := m.cmdPage*itemsPerPage + m.cmdCursor
		if idx >= total {
			return m, nil
		}
		return m.handleCmdSelect(cmdList[idx].Name)
	case "esc", "ctrl+c":
		return m.handleCmdCancel()
	default:
		var cmd tea.Cmd
		m.inputText, cmd = m.inputText.Update(msg)
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		if !strings.HasPrefix(m.inputText.Value(), "/") {
			return m.handleCmdCancel()
		}
		return m, tea.Batch(cmd, vpCmd)
	}
}

func (m Model) executeCommand(input string) (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.inputText.SetValue("")
	m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
	parts := strings.Fields(input)
	if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
		m.sessionId = ""
		m.messages = nil
		m = m.refreshMessages()
		return m, m.addAssistantMsg("Started a new session.")
	}
	return m, m.handleCommand(input)
}

func (m Model) handleCmdSelect(cmd string) (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.inputText.SetValue(cmd)
	m.inputText.SetCursor(len(cmd))
	m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
	return m, nil
}

func (m Model) handleCmdCancel() (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.inputText.SetValue("")
	m.viewPort.SetHeight(m.termHeight - splashHeight - inputBoxHeight)
	return m, nil
}
