package tui

import (
	"fmt"
	"oc/internal/api"
	"oc/internal/history"
	"oc/internal/server"
	"oc/internal/tui/commands"
	"strings"
	"time"

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

	case "v":
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
					return m, commands.AddAssistantMsg("Started a new session.")
				}
				return m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			if m.sessionId != "" {
				history.AppendMessage(m.sessionId, "user", input)
			}
			m = m.refreshMessages()
			m.inputText.SetValue("")
			m.loading = true
			return m, commands.SendChat(m.client, m.sessionId, input)
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
					return m, commands.AddAssistantMsg("Started a new session.")
				}
				if input == "/" {
					return m.showCmdList(), nil
				}
				return m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			if m.sessionId != "" {
				history.AppendMessage(m.sessionId, "user", input)
			}
			m = m.refreshMessages()
			m.inputText.SetValue("")
			m.loading = true
			m.viewPort.GotoBottom()
			return m, commands.SendChat(m.client, m.sessionId, input)
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
		if strings.HasPrefix(m.inputText.Value(), "/") {
			return m.showCmdList(), cmd
		}
		return m, cmd
	}
}

func (m Model) onVisualKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j":
		if m.visualCursor < len(m.messages)-1 {
			m.visualCursor++
			m = m.refreshMessages()
			m.viewPort.ScrollDown(3)
		}
		return m, nil

	case "k":
		if m.visualCursor > 0 {
			m = m.refreshMessages()
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
		time.AfterFunc(3*time.Second, func() {
			m.inputText.SetValue("")
		})
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

	m.mode = modeQus
	m.inputText.Blur()
	return m.syncLayout()
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
	m.qusHeight = 0
	m.loading = true
	return m.syncLayout(), commands.SendControlResponse(m.client, m.pendingControl, m.questionAnswers)
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
	m.qusHeight = 0
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Question cancelled."})
	m = m.refreshMessages()
	return m.syncLayout(), nil
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

	m.mode = modeSession
	m.inputText.Blur()
	return m.syncLayout()
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
		m = m.syncLayout()
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
	return m.syncLayout(), nil
}

func filteredCmdList(m Model) []cmdItem {
	input := strings.TrimPrefix(m.inputText.Value(), "/")
	if input == "" {
		return cmdList
	}
	var result []cmdItem
	for _, c := range cmdList {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(input)) {
			result = append(result, c)
		}
	}
	return result
}

var cmdList = []cmdItem{
	{Name: "/help", Category: "help", Description: "Show available commands"},
	{Name: "/sessions", Category: "history", Description: "List and load past sessions"},
	{Name: "/session new", Category: "history", Description: "Start a fresh session"},
	{Name: "/clear", Category: "chat", Description: "Clear chat messages"},
	{Name: "/multiagent", Category: "Agent", Description: "Toggle multi-agent mode — spawns sub-agents that work on tasks in parallel to solve complex problems faster"},
	{Name: "/model", Category: "model", Description: "Toogle between models"},
	{Name: "/retry", Category: "chat", Description: "Re-send last user message"},
	{Name: "/load <n>", Category: "history", Description: "Load session by number"},
	{Name: "/tokens", Category: "info", Description: "Show token usage"},
	{Name: "/exit", Category: "exit", Description: "Quit the app"},
}

func filteredModelList(m Model) []api.ModelList {
	input := m.inputText.Value()
	if input == "" {
		return m.models
	}
	var result []api.ModelList
	for _, model := range m.models {
		if strings.Contains(strings.ToLower(model.Name), strings.ToLower(input)) {
			result = append(result, model)
		}
	}
	return result
}

func (m Model) showModelList() Model {
	m.modelCursor = 0
	m.modelPage = 0
	m.mode = modeModel
	m.inputText.Focus()
	return m.syncLayout()
}

func (m Model) onModelKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	models := filteredModelList(m)
	const itemsPerPage = 5
	total := len(models)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	itemsOnPage := itemsPerPage
	start := m.modelPage * itemsPerPage
	if end := start + itemsPerPage; end > total {
		itemsOnPage = total - start
	}

	switch msg.String() {
	case "j", "down":
		if m.modelCursor < itemsOnPage-1 {
			m.modelCursor++
		} else if m.modelPage < totalPages-1 {
			m.modelPage++
			m.modelCursor = 0
		}
		return m, nil
	case "k", "up":
		if m.modelCursor > 0 {
			m.modelCursor--
		} else if m.modelPage > 0 {
			m.modelPage--
			itemsOnPrev := itemsPerPage
			if end := m.modelPage*itemsPerPage + itemsPerPage; end > total {
				itemsOnPrev = total - m.modelPage*itemsPerPage
			}
			m.modelCursor = itemsOnPrev - 1
		}
		return m, nil
	case "esc", "ctrl+c":
		m.mode = modeInsert
		m.inputText.Focus()
		return m.syncLayout(), nil
	case "enter":
		if total > 0 {
			idx := m.modelPage*itemsPerPage + m.modelCursor
			if idx < total {
				selected := models[idx]
				m.modelName = selected.Name
				m.modelID = selected.ID
				m.modelProviderID = selected.ProviderID
				m.client.ModelID = selected.ID
				m.client.ModelProviderID = selected.ProviderID
				if m.sessionId != "" {
					m.sessionId = ""
				}
				m.mode = modeInsert
				m.inputText.Focus()
				m.inputText.SetValue("")
				return m.syncLayout(), func() tea.Msg {
					m.client.SetModel(selected.ID)
					return nil
				}
			}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.inputText, cmd = m.inputText.Update(msg)
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		m.modelPage = 0
		m.modelCursor = 0
		return m.syncLayout(), tea.Batch(cmd, vpCmd)
	}
}

func (m Model) showCmdList() Model {
	m.cmdPage = 0
	m.cmdCursor = 0

	m.mode = modeCmd
	return m.syncLayout()
}

func (m Model) onCmdKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	cmds := filteredCmdList(m)
	const itemsPerPage = 5
	total := len(cmds)
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
	case "tab":
		if total > 0 {
			idx := m.cmdPage*itemsPerPage + m.cmdCursor
			if idx < total {
				m.mode = modeInsert
				m.inputText.Focus()
				m.inputText.SetValue(cmds[idx].Name)
				m.inputText.SetCursor(len(cmds[idx].Name))
				return m.syncLayout(), nil
			}
		}
		return m, nil
	case "enter":
		if total > 0 {
			idx := m.cmdPage*itemsPerPage + m.cmdCursor
			if idx < total {
				return m.executeCommand(cmds[idx].Name)
			}
		}
		return m, nil
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
		m.cmdPage = 0
		m.cmdCursor = 0
		return m.syncLayout(), tea.Batch(cmd, vpCmd)
	}
}

func (m Model) executeCommand(input string) (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.inputText.SetValue("")
	m = m.syncLayout()
	parts := strings.Fields(input)

	switch {
	case len(parts) == 2 && parts[0] == "/session" && parts[1] == "new":
		m.sessionId = ""
		m.messages = nil
		m = m.refreshMessages()
		return m, commands.AddAssistantMsg("Started a new session.")

	case parts[0] == "/clear":
		m.messages = nil
		m = m.refreshMessages()
		return m, commands.AddAssistantMsg("Chat cleared.")

	case parts[0] == "/retry":
		if len(m.messages) == 0 {
			return m, commands.AddAssistantMsg("Nothing to retry.")
		}
		lastUserIdx := -1
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == "user" {
				lastUserIdx = i
				break
			}
		}
		if lastUserIdx == -1 {
			return m, commands.AddAssistantMsg("No user message to retry.")
		}
		lastInput := m.messages[lastUserIdx].Content
		m.messages = m.messages[:lastUserIdx+1]
		m = m.refreshMessages()
		m.loading = true
		return m, commands.SendChat(m.client, m.sessionId, lastInput)

	}

	return m.handleCommand(input)
}

func (m Model) handleCmdSelect(cmd string) (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.inputText.SetValue(cmd)
	m.inputText.SetCursor(len(cmd))
	return m.syncLayout(), nil
}

func (m Model) handleCmdCancel() (Model, tea.Cmd) {
	m.mode = modeInsert
	m.inputText.Focus()
	m.inputText.Placeholder = "Ask anything ..."
	m.inputText.SetValue("")
	return m.syncLayout(), nil
}

func (m Model) onPermKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	var reply string
	switch msg.String() {
	case "y":
		reply = "once"
	case "a":
		reply = "always"
	case "n", "esc", "ctrl+c":
		reply = "reject"
	default:
		return m, nil
	}
	id := m.pendingPermission.ID
	m.pendingPermission = nil
	m.mode = modeInsert
	m.inputText.Focus()
	m = m.syncLayout()
	r := reply
	return m, func() tea.Msg {
		err := m.client.ReplyToPermission(id, r)
		if err != nil {
			return PermissionRequestMsg{Err: err}
		}
		return PermissionRequestMsg{Reply: r}
	}
}
