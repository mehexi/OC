package tui

import (
	"fmt"
	"oc/internal/api"
	"oc/internal/history"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// onServerStarted initialises the API client and triggers a health check.
func (m Model) onServerStarted(msg ServerStartedMsg) (Model, tea.Cmd) {
	m.serverAddr = msg.Address
	m.client = api.New(msg.Address)
	return m, m.checkHealth()
}

func (m Model) refreshMessages() Model {
	var chatBubbles []string
	for i, msg := range m.messages {
		bubble := RenderChatBubble(msg, m)
		if m.mode == modeVisual {
			lo, hi := m.visualAnchor, m.visualCursor
			if lo > hi {
				lo, hi = hi, lo
			}
			if i >= lo && i <= hi {
				bubble = lipgloss.NewStyle().Background(selectBgColor).Render(bubble)
			}
		}
		chatBubbles = append(chatBubbles, bubble)
	}
	m.viewPort.SetContent(strings.Join(chatBubbles, "\n\n"))
	return m
}

// onServerErr appends a server-error message to the chat.
func (m Model) onServerErr(msg ServerErrMsg) (Model, tea.Cmd) {
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Server error: " + msg.err.Error()})
	return m.refreshMessages(), nil
}

// onHealthCheck records health status and shows a welcome or error message.
func (m Model) onHealthCheck(msg HealthCheckMsg) (Model, tea.Cmd) {
	m.healthChecked = true
	if msg.Err != nil {
		m.healthErr = msg.Err
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Server error: " + msg.Err.Error()})
	} else {
		m.healthStatus = msg.Status
		welcome := fmt.Sprintf("Server v%s connected. Type /sessions for history.", msg.Status.Version)
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: welcome})
		return m.refreshMessages(), tea.Batch(m.fetchProviders(), m.fetchPath())
	}
	return m.refreshMessages(), nil
}

// onProvidersInfo stores the default model name.
func (m Model) onProvidersInfo(msg ProvidersInfoMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.modelName = msg.ModelName
	}
	return m, nil
}

// onPath stores the current working directory path and starts SSE listener.
func (m Model) onPath(msg PathMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.currentPath = msg.Path
		m.client.Directory = msg.Path
		return m, m.startSSEListener()
	}
	return m, nil
}

// onSessionUsage stores token usage info from the current session.
func (m Model) onSessionUsage(msg SessionUsageMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.tokensUsed = msg.TokensUsed
		m.contextLimit = msg.ContextLimit
		if msg.ModelName != "" {
			m.modelName = msg.ModelName
		}
	}
	return m, nil
}

// onControlRequest handles incoming questions from the question tool.
func (m Model) onControlRequest(msg ControlRequestMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.loading = false
		m.awaitingResponse = false
		m.pendingControl = nil
		m.currentQuestionIdx = 0
		m.questionAnswers = nil
		m.inputText.Placeholder = "Ask anything ..."
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Control request error: " + msg.Err.Error()})
		return m.refreshMessages(), nil
	}
	if msg.Request == nil {
		m.loading = false
		m.inputText.Placeholder = "Ask anything ..."
		if m.awaitingResponse {
			return m, nil
		}
		m.awaitingResponse = false
		if m.pendingControl != nil {
			m = m.syncLayout()
			var sb strings.Builder
			sb.WriteString("Answers:\n")
			for i, q := range m.pendingControl.Data.Questions {
				a := ""
				if i < len(m.questionAnswers) {
					a = m.questionAnswers[i]
				}
				fmt.Fprintf(&sb, "- %s: %s\n", q.Header, a)
			}
			m.pendingControl = nil
			m.currentQuestionIdx = 0
			m.questionAnswers = nil
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: strings.TrimSpace(sb.String())})
			m = m.refreshMessages()
			m.inputText.SetValue("")
			if m.streaming {
				return m, nil
			}
			m.loading = true
			return m, m.sendChat(strings.TrimSpace(sb.String()))
		}
		if m.streaming {
			return m, nil
		}
		return m, nil
	}

	if m.pendingControl != nil {
		return m, nil
	}
	m.pendingControl = msg.Request
	m.currentQuestionIdx = 0
	m.questionAnswers = nil
	m.awaitingResponse = true
	m.loading = false

	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.Request.Data.Questions[0].Header})
	m = m.refreshMessages()
	return m.showQusList(), nil
}

// onStreamMsg handles SSE streaming chunks from the AI response.
func (m Model) onStreamMsg(msg ChatStreamMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.loading = false
		m.streaming = false
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Error: " + msg.Err.Error()})
		return m.refreshMessages(), nil
	}

	// First message carries the session ID — persist user message
	if msg.SessionID != "" && m.sessionId == "" {
		m.sessionId = msg.SessionID
		if len(m.messages) > 0 {
			history.AppendMessage(m.sessionId, "user", m.messages[len(m.messages)-1].Content)
		}
	}

	if msg.Done {
		m.streaming = false
		if msg.ModelName != "" {
			m.modelName = msg.ModelName
		}
		// Persist final assistant message
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == "assistant" {
				history.AppendMessage(m.sessionId, "assistant", m.messages[i].Content)
				break
			}
		}
		return m.refreshMessages(), m.fetchSessionUsage()
	}

	m.loading = false
	firstStream := !m.streaming
	m.streaming = true

	if msg.Text == "" {
		return m, nil
	}

	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
		m.messages[len(m.messages)-1].Content += msg.Text
	} else {
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.Text})
	}

	m = m.refreshMessages()
	m.viewPort.GotoBottom()
	if firstStream {
		return m, nil
	}
	return m, nil
}

// onChatResponse handles an incoming chat response or error and persists history.
func (m Model) onChatResponse(msg ChatResponseMsg) (Model, tea.Cmd) {
	m.loading = false
	if msg.Err != nil {
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Error: " + msg.Err.Error()})
	} else {
		if msg.SessionID != "" {
			isNew := m.sessionId == ""
			m.sessionId = msg.SessionID
			if isNew {
				history.AppendMessage(msg.SessionID, "user", m.messages[len(m.messages)-1].Content)
			}
		}
		if msg.ModelName != "" {
			m.modelName = msg.ModelName
		}
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.Response})
		if m.sessionId != "" {
			history.AppendMessage(m.sessionId, "assistant", msg.Response)
		}
	}
	m = m.refreshMessages()
	m.viewPort.GotoBottom()
	return m, m.fetchSessionUsage()
}

// onLoadSession populates messages and session ID from a loaded history session.
func (m Model) onLoadSession(msg LoadSessionMsg) (Model, tea.Cmd) {
	m.sessionId = msg.Session.ID
	m.messages = make([]ChatMessage, len(msg.Session.Messages))
	for i, msg := range msg.Session.Messages {
		m.messages[i] = ChatMessage{Role: msg.Role, Content: msg.Content}
	}
	m = m.refreshMessages()
	m.viewPort.GotoBottom()
	return m, m.fetchSessionUsage()
}

const inputBoxHeight = 3

func (m Model) viewportHeight() int {
	headerHeight := lipgloss.Height(m.renderHeader())
	available := m.termHeight - headerHeight - inputBoxHeight
	if available < 1 {
		available = 1
	}
	switch m.mode {
	case modeQus:
		available -= m.qusHeight
	case modeSession:
		sessionLines := 2 + 5
		total := len(m.sessions)
		totalPages := (total + 5 - 1) / 5
		if totalPages > 1 {
			sessionLines += 2
		}
		available -= sessionLines
	case modeCmd:
		cmds := filteredCmdList(m)
		cmdLines := 2 + 5
		total := len(cmds)
		totalPages := (total + 5 - 1) / 5
		if totalPages > 1 {
			cmdLines += 2
		}
		available -= cmdLines
	}
	if available < 1 {
		available = 1
	}
	return available
}

func (m Model) syncLayout() Model {
	m.viewPort.SetWidth(m.width)
	m.viewPort.SetHeight(m.viewportHeight())
	m.inputText.SetWidth(m.width - 6)
	return m
}

// onWindowSize updates layout dimensions when the terminal is resized.
func (m Model) onWindowSize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.width = msg.Width
	m.termHeight = msg.Height
	m = m.syncLayout()
	return m, nil
}

// onKeyPress dispatches key events to the active mode handler.
func (m Model) onKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch m.mode {
	case modeNormal:
		return m.onNormalKey(msg)
	case modeInsert:
		return m.onInsertKey(msg)
	case modeVisual:
		return m.onVisualKey(msg)
	case modeQus:
		return m.onQusKey(msg)
	case modeSession:
		return m.onSessionKey(msg)
	case modeCmd:
		return m.onCmdKey(msg)
	case modePerm:
		return m.onPermKey(msg)
	default:
		return m.onInsertKey(msg)
	}
}

// rebuildView refreshes viewport content and propagates component updates.
func (m Model) rebuildView(msg tea.Msg) (Model, tea.Cmd) {
	m = m.refreshMessages()

	var cmd tea.Cmd
	m.inputText, cmd = m.inputText.Update(msg)
	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)
	return m, tea.Batch(cmd, vpCmd)
}

// Update dispatches messages to typed handler methods.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ServerStartedMsg:
		return m.onServerStarted(msg)
	case ServerErrMsg:
		return m.onServerErr(msg)
	case HealthCheckMsg:
		return m.onHealthCheck(msg)
	case ChatStreamMsg:
		return m.onStreamMsg(msg)
	case ControlRequestMsg:
		return m.onControlRequest(msg)
	case PermissionRequestMsg:
		if msg.Err != nil {
			m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Permission error: " + msg.Err.Error()})
			return m.refreshMessages(), nil
		}
		if msg.Reply != "" {
			m.pendingPermission = nil
			var label string
			switch msg.Reply {
			case "once":
				label = "Permission granted (once)"
			case "always":
				label = "Permission granted (always)"
			case "reject":
				label = "Permission rejected"
			}
			if m.permissionMsgIndex >= 0 && m.permissionMsgIndex < len(m.messages) {
				m.messages[m.permissionMsgIndex].Content = label
			}
			m.permissionMsgIndex = -1
			return m.refreshMessages(), nil
		}
		m.pendingPermission = msg.Request
		m.mode = modePerm
		m.inputText.Blur()
		patterns := strings.Join(msg.Request.Patterns, ", ")
		m.permissionMsgIndex = len(m.messages)
		m.messages = append(m.messages, ChatMessage{Role: "permission", Content: "Permission: " + msg.Request.Permission + " on " + patterns + "\n  y=once  a=always  n=reject  esc=cancel"})
		return m.refreshMessages(), nil
	case ChatResponseMsg:
		return m.onChatResponse(msg)
	case LoadSessionMsg:
		return m.onLoadSession(msg)
	case ProvidersInfoMsg:
		return m.onProvidersInfo(msg)
	case PathMsg:
		return m.onPath(msg)
	case SessionUsageMsg:
		return m.onSessionUsage(msg)
	case tea.WindowSizeMsg:
		return m.onWindowSize(msg)
	case ShowSessionListMsg:
		return m.showSessionList(), nil
	case tea.KeyPressMsg:
		return m.onKeyPress(msg)
	}
	return m.rebuildView(msg)
}
