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
		m.Modes.mode = modeInsert
		m.Layout.inputText.Focus()
		s := m.Layout.inputText.Styles()
		s.Cursor.Color = cyanColor
		s.Cursor.Blink = false
		m.Layout.inputText.SetStyles(s)
		return m, nil

	case "j":
		m.Layout.viewPort.ScrollDown(3)
		return m, nil

	case "k":
		m.Layout.viewPort.ScrollUp(3)
		return m, nil

	case "g":
		if m.Modes.awaitingGG {
			m.Modes.awaitingGG = false
			m.Layout.viewPort.GotoTop()
		} else {
			m.Modes.awaitingGG = true
		}
		return m, nil

	case "G":
		m.Modes.awaitingGG = false
		m.Layout.viewPort.GotoBottom()
		return m, nil

	case "v":
		m.Modes.awaitingGG = false
		if len(m.Chat.messages) == 0 {
			return m, nil
		}
		m.Modes.mode = modeVisual
		last := len(m.Chat.messages) - 1
		m.Modes.visualAnchor = last
		m.Modes.visualCursor = last
		return m, nil

	case "/":
		m.Modes.awaitingGG = false
		m.Layout.inputText.SetValue("/")
		m.Layout.inputText.SetCursor(len("/"))
		m.Layout.inputText.Focus()
		return m.showCmdList(), nil

	case "enter":
		m.Modes.awaitingGG = false
		input := m.Layout.inputText.Value()
		if input == "" {
			return m, nil
		}
		if !m.Chat.loading && !m.Chat.streaming {
			if strings.HasPrefix(input, "/") {
				m.Layout.inputText.SetValue("")
				parts := strings.Fields(input)
				if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
					m.Chat.sessionId = ""
					m.Chat.messages = nil
					m = m.refreshMessages()
					return m, commands.AddAssistantMsg("Started a new session.")
				}
				return m.handleCommand(input)
			}
			m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleUser, Content: input})
			if m.Chat.sessionId != "" {
				history.AppendMessage(m.Chat.sessionId, string(RoleUser), input)
			}
			m = m.refreshMessages()
			m.Layout.inputText.SetValue("")
			m.Chat.loading = true
			return m, commands.SendChat(m.Server.client, m.Chat.sessionId, input)
		}
		return m, nil

	case "ctrl+c":
		if m.Layout.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.Layout.inputText.SetValue("")
		return m, nil

	default:
		m.Modes.awaitingGG = false
		var vpCmd tea.Cmd
		m.Layout.viewPort, vpCmd = m.Layout.viewPort.Update(msg)
		return m, vpCmd
	}
}

func (m Model) onInsertKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.Modes.mode = modeNormal
		m.Layout.inputText.Blur()
		return m, nil

	case "enter":
		input := m.Layout.inputText.Value()
		if input == "" {
			return m, nil
		}
		if !m.Chat.loading && !m.Chat.streaming {
			if strings.HasPrefix(input, "/") {
				m.Layout.inputText.SetValue("")
				parts := strings.Fields(input)
				if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
					m.Chat.sessionId = ""
					m.Chat.messages = nil
					m = m.refreshMessages()
					return m, commands.AddAssistantMsg("Started a new session.")
				}
				if input == "/" {
					return m.showCmdList(), nil
				}
				return m.handleCommand(input)
			}
			m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleUser, Content: input})
			if m.Chat.sessionId != "" {
				history.AppendMessage(m.Chat.sessionId, string(RoleUser), input)
			}
			m = m.refreshMessages()
			m.Layout.inputText.SetValue("")
			m.Chat.loading = true
			m.Layout.viewPort.GotoBottom()
			return m, commands.SendChat(m.Server.client, m.Chat.sessionId, input)
		}
		return m, nil

	case "ctrl+c":
		if m.Layout.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.Layout.inputText.SetValue("")
		return m, nil

	default:
		var cmd tea.Cmd
		m.Layout.inputText, cmd = m.Layout.inputText.Update(msg)
		if strings.HasPrefix(m.Layout.inputText.Value(), "/") {
			return m.showCmdList(), cmd
		}
		return m, cmd
	}
}

func (m Model) onVisualKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j":
		if m.Modes.visualCursor < len(m.Chat.messages)-1 {
			m.Modes.visualCursor++
			m = m.refreshMessages()
			m.Layout.viewPort.ScrollDown(3)
		}
		return m, nil

	case "k":
		if m.Modes.visualCursor > 0 {
			m = m.refreshMessages()
			m.Modes.visualCursor--
			m.Layout.viewPort.ScrollUp(3)
		}
		return m, nil

	case "y":
		lo, hi := m.Modes.visualAnchor, m.Modes.visualCursor
		if lo > hi {
			lo, hi = hi, lo
		}
		var b strings.Builder
		for i := lo; i <= hi && i < len(m.Chat.messages); i++ {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(m.Chat.messages[i].Content)
		}
		text := b.String()
		if text != "" {
			clipboard.WriteAll(text)
		}
		m.Modes.mode = modeNormal
		m.Layout.inputText.SetValue(fmt.Sprintf("Yanked %d message(s)", hi-lo+1))
		time.AfterFunc(3*time.Second, func() {
			m.Layout.inputText.SetValue("")
		})
		m.Layout.inputText.SetCursor(len(m.Layout.inputText.Value()))
		return m, nil

	case "esc":
		m.Modes.mode = modeNormal
		return m, nil

	default:
		var vpCmd tea.Cmd
		m.Layout.viewPort, vpCmd = m.Layout.viewPort.Update(msg)
		return m, vpCmd
	}
}

func (m Model) showQusList() Model {
	q := m.Flow.pendingControl.Data.Questions[m.Flow.currentQuestionIdx]
	m.Modes.qusItems = make([]qusItem, len(q.Options))
	for i, opt := range q.Options {
		m.Modes.qusItems[i] = qusItem{label: opt.Label, desc: opt.Description}
	}
	m.Modes.qusCursor = 0

	const compactHeaderHeight = 3
	available := m.Layout.termHeight - compactHeaderHeight - inputBoxHeight
	qusHeight := 2 + len(m.Modes.qusItems)
	if qusHeight > available-3 {
		qusHeight = available - 3
	}
	m.Modes.qusHeight = qusHeight

	m.Modes.mode = modeQus
	m.Layout.inputText.Blur()
	return m.syncLayout()
}

func (m Model) handleQusAnswer() (Model, tea.Cmd) {
	if m.Modes.qusCursor >= len(m.Modes.qusItems) {
		return m, nil
	}
	answer := m.Modes.qusItems[m.Modes.qusCursor].label

	m.Flow.questionAnswers = append(m.Flow.questionAnswers, answer)
	m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleUser, Content: answer})
	m = m.refreshMessages()

	m.Flow.currentQuestionIdx++
	if m.Flow.currentQuestionIdx < len(m.Flow.pendingControl.Data.Questions) {
		return m.showQusList(), nil
	}

	m.Flow.awaitingResponse = false
	m.Modes.qusItems = nil
	m.Modes.qusCursor = 0
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m.Layout.inputText.Placeholder = "Ask anything ..."
	m.Modes.qusHeight = 0
	m.Chat.loading = true
	return m.syncLayout(), commands.SendControlResponse(m.Server.client, m.Flow.pendingControl, m.Flow.questionAnswers)
}

func (m Model) handleQusCancel() (Model, tea.Cmd) {
	m.Flow.pendingControl = nil
	m.Flow.currentQuestionIdx = 0
	m.Flow.questionAnswers = nil
	m.Flow.awaitingResponse = false
	m.Modes.qusItems = nil
	m.Modes.qusCursor = 0
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m.Layout.inputText.Placeholder = "Ask anything ..."
	m.Modes.qusHeight = 0
	m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Question cancelled."})
	m = m.refreshMessages()
	return m.syncLayout(), nil
}

func (m Model) onQusKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.Modes.qusCursor < len(m.Modes.qusItems)-1 {
			m.Modes.qusCursor++
		}
		return m, nil
	case "k", "up":
		if m.Modes.qusCursor > 0 {
			m.Modes.qusCursor--
		}
		return m, nil
	case "enter":
		if len(m.Modes.qusItems) == 0 {
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
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "No past sessions."})
		m = m.refreshMessages()
		return m
	}
	m.Modes.sessions = sessions
	m.Modes.sessionPage = 0
	m.Modes.sessionCursor = 0

	m.Modes.mode = modeSession
	m.Layout.inputText.Blur()
	return m.syncLayout()
}

func (m Model) onSessionKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	itemsPerPage := 5
	total := len(m.Modes.sessions)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	itemsOnPage := itemsPerPage
	start := m.Modes.sessionPage * itemsPerPage
	if end := start + itemsPerPage; end > total {
		itemsOnPage = total - start
	}

	switch msg.String() {
	case "j", "down":
		if m.Modes.sessionCursor < itemsOnPage-1 {
			m.Modes.sessionCursor++
		} else if m.Modes.sessionPage < totalPages-1 {
			m.Modes.sessionPage++
			m.Modes.sessionCursor = 0
		}
		return m, nil
	case "k", "up":
		if m.Modes.sessionCursor > 0 {
			m.Modes.sessionCursor--
		} else if m.Modes.sessionPage > 0 {
			m.Modes.sessionPage--
			itemsOnPrev := itemsPerPage
			if end := m.Modes.sessionPage*itemsPerPage + itemsPerPage; end > total {
				itemsOnPrev = total - m.Modes.sessionPage*itemsPerPage
			}
			m.Modes.sessionCursor = itemsOnPrev - 1
		}
		return m, nil
	case "enter":
		idx := m.Modes.sessionPage*itemsPerPage + m.Modes.sessionCursor
		if idx >= total {
			return m, nil
		}
		selected := m.Modes.sessions[idx]
		m.Modes.mode = modeInsert
		m.Layout.inputText.Focus()
		m.Layout.inputText.Placeholder = "Ask anything ..."
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
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m.Layout.inputText.Placeholder = "Ask anything ..."
	return m.syncLayout(), nil
}

func filteredCmdList(m Model) []cmdItem {
	input := strings.TrimPrefix(m.Layout.inputText.Value(), "/")
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
	{Name: "/model", Category: "model", Description: "Toogle between models"},
	{Name: "/retry", Category: "chat", Description: "Re-send last user message"},
	{Name: "/load <n>", Category: "history", Description: "Load session by number"},
	{Name: "/tokens", Category: "info", Description: "Show token usage"},
	{Name: "/exit", Category: "exit", Description: "Quit the app"},
}

func filteredModelList(m Model) []api.ModelList {
	input := m.Layout.inputText.Value()
	if input == "" {
		return m.Modes.models
	}
	var result []api.ModelList
	for _, model := range m.Modes.models {
		if strings.Contains(strings.ToLower(model.Name), strings.ToLower(input)) {
			result = append(result, model)
		}
	}
	return result
}

func (m Model) showModelList() Model {
	m.Modes.modelCursor = 0
	m.Modes.modelPage = 0
	m.Modes.mode = modeModel
	m.Layout.inputText.Focus()
	return m.syncLayout()
}

func (m Model) onModelKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	models := filteredModelList(m)
	const itemsPerPage = 5
	total := len(models)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	itemsOnPage := itemsPerPage
	start := m.Modes.modelPage * itemsPerPage
	if end := start + itemsPerPage; end > total {
		itemsOnPage = total - start
	}

	switch msg.String() {
	case "j", "down":
		if m.Modes.modelCursor < itemsOnPage-1 {
			m.Modes.modelCursor++
		} else if m.Modes.modelPage < totalPages-1 {
			m.Modes.modelPage++
			m.Modes.modelCursor = 0
		}
		return m, nil
	case "k", "up":
		if m.Modes.modelCursor > 0 {
			m.Modes.modelCursor--
		} else if m.Modes.modelPage > 0 {
			m.Modes.modelPage--
			itemsOnPrev := itemsPerPage
			if end := m.Modes.modelPage*itemsPerPage + itemsPerPage; end > total {
				itemsOnPrev = total - m.Modes.modelPage*itemsPerPage
			}
			m.Modes.modelCursor = itemsOnPrev - 1
		}
		return m, nil
	case "esc", "ctrl+c":
		m.Modes.mode = modeInsert
		m.Layout.inputText.Focus()
		return m.syncLayout(), nil
	case "enter":
		if total > 0 {
			idx := m.Modes.modelPage*itemsPerPage + m.Modes.modelCursor
			if idx < total {
				selected := models[idx]
				m.Server.modelName = selected.Name
				m.Server.modelID = selected.ID
				m.Server.modelProviderID = selected.ProviderID
				m.Server.client.ModelID = selected.ID
				m.Server.client.ModelProviderID = selected.ProviderID
				if m.Chat.sessionId != "" {
					m.Chat.sessionId = ""
				}
				m.Modes.mode = modeInsert
				m.Layout.inputText.Focus()
				m.Layout.inputText.SetValue("")
				return m.syncLayout(), func() tea.Msg {
					m.Server.client.SetModel(selected.ID)
					return nil
				}
			}
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.Layout.inputText, cmd = m.Layout.inputText.Update(msg)
		var vpCmd tea.Cmd
		m.Layout.viewPort, vpCmd = m.Layout.viewPort.Update(msg)
		m.Modes.modelPage = 0
		m.Modes.modelCursor = 0
		return m.syncLayout(), tea.Batch(cmd, vpCmd)
	}
}

func (m Model) showCmdList() Model {
	m.Modes.cmdPage = 0
	m.Modes.cmdCursor = 0

	m.Modes.mode = modeCmd
	return m.syncLayout()
}

func (m Model) onCmdKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	cmds := filteredCmdList(m)
	const itemsPerPage = 5
	total := len(cmds)
	totalPages := (total + itemsPerPage - 1) / itemsPerPage
	itemsOnPage := itemsPerPage
	start := m.Modes.cmdPage * itemsPerPage
	if end := start + itemsPerPage; end > total {
		itemsOnPage = total - start
	}

	switch msg.String() {
	case "j", "down":
		if m.Modes.cmdCursor < itemsOnPage-1 {
			m.Modes.cmdCursor++
		} else if m.Modes.cmdPage < totalPages-1 {
			m.Modes.cmdPage++
			m.Modes.cmdCursor = 0
		}
		return m, nil
	case "k", "up":
		if m.Modes.cmdCursor > 0 {
			m.Modes.cmdCursor--
		} else if m.Modes.cmdPage > 0 {
			m.Modes.cmdPage--
			itemsOnPrev := itemsPerPage
			if end := m.Modes.cmdPage*itemsPerPage + itemsPerPage; end > total {
				itemsOnPrev = total - m.Modes.cmdPage*itemsPerPage
			}
			m.Modes.cmdCursor = itemsOnPrev - 1
		}
		return m, nil
	case "tab":
		if total > 0 {
			idx := m.Modes.cmdPage*itemsPerPage + m.Modes.cmdCursor
			if idx < total {
				m.Modes.mode = modeInsert
				m.Layout.inputText.Focus()
				m.Layout.inputText.SetValue(cmds[idx].Name)
				m.Layout.inputText.SetCursor(len(cmds[idx].Name))
				return m.syncLayout(), nil
			}
		}
		return m, nil
	case "enter":
		if total > 0 {
			idx := m.Modes.cmdPage*itemsPerPage + m.Modes.cmdCursor
			if idx < total {
				return m.executeCommand(cmds[idx].Name)
			}
		}
		return m, nil
	case "esc", "ctrl+c":
		return m.handleCmdCancel()
	default:
		var cmd tea.Cmd
		m.Layout.inputText, cmd = m.Layout.inputText.Update(msg)
		var vpCmd tea.Cmd
		m.Layout.viewPort, vpCmd = m.Layout.viewPort.Update(msg)
		if !strings.HasPrefix(m.Layout.inputText.Value(), "/") {
			return m.handleCmdCancel()
		}
		m.Modes.cmdPage = 0
		m.Modes.cmdCursor = 0
		return m.syncLayout(), tea.Batch(cmd, vpCmd)
	}
}

func (m Model) executeCommand(input string) (Model, tea.Cmd) {
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m.Layout.inputText.Placeholder = "Ask anything ..."
	m.Layout.inputText.SetValue("")
	m = m.syncLayout()
	parts := strings.Fields(input)

	switch {
	case len(parts) == 2 && parts[0] == "/session" && parts[1] == "new":
		m.Chat.sessionId = ""
		m.Chat.messages = nil
		m = m.refreshMessages()
		return m, commands.AddAssistantMsg("Started a new session.")

	case parts[0] == "/clear":
		m.Chat.messages = nil
		m = m.refreshMessages()
		return m, commands.AddAssistantMsg("Chat cleared.")

	case parts[0] == "/retry":
		if len(m.Chat.messages) == 0 {
			return m, commands.AddAssistantMsg("Nothing to retry.")
		}
		lastUserIdx := -1
		for i := len(m.Chat.messages) - 1; i >= 0; i-- {
			if m.Chat.messages[i].Role == RoleUser {
				lastUserIdx = i
				break
			}
		}
		if lastUserIdx == -1 {
			return m, commands.AddAssistantMsg("No user message to retry.")
		}
		lastInput := m.Chat.messages[lastUserIdx].Content
		m.Chat.messages = m.Chat.messages[:lastUserIdx+1]
		m = m.refreshMessages()
		m.Chat.loading = true
		return m, commands.SendChat(m.Server.client, m.Chat.sessionId, lastInput)

	}

	return m.handleCommand(input)
}

func (m Model) handleCmdSelect(cmd string) (Model, tea.Cmd) {
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m.Layout.inputText.Placeholder = "Ask anything ..."
	m.Layout.inputText.SetValue(cmd)
	m.Layout.inputText.SetCursor(len(cmd))
	return m.syncLayout(), nil
}

func (m Model) handleCmdCancel() (Model, tea.Cmd) {
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m.Layout.inputText.Placeholder = "Ask anything ..."
	m.Layout.inputText.SetValue("")
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
	id := m.Flow.pendingPermission.ID
	m.Flow.pendingPermission = nil
	m.Modes.mode = modeInsert
	m.Layout.inputText.Focus()
	m = m.syncLayout()
	r := reply
	return m, func() tea.Msg {
		err := m.Server.client.ReplyToPermission(id, r)
		if err != nil {
			return PermissionRequestMsg{Err: err}
		}
		return PermissionRequestMsg{Reply: r}
	}
}
