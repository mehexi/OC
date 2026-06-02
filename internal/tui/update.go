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

// onPath stores the current working directory path.
func (m Model) onPath(msg PathMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.currentPath = msg.Path
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

const splashHeight = 17
const inputBoxHeight = 3

// onWindowSize updates layout dimensions when the terminal is resized.
func (m Model) onWindowSize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.width = msg.Width
	m.viewPort.SetWidth(msg.Width)
	m.viewPort.SetHeight(msg.Height - splashHeight - inputBoxHeight)
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
	case tea.KeyPressMsg:
		return m.onKeyPress(msg)
	}
	return m.rebuildView(msg)
}
