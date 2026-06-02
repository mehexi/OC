package tui

import (
	"fmt"
	"oc/internal/api"
	"oc/internal/history"
	"oc/internal/server"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// onServerStarted initialises the API client and triggers a health check.
func (m Model) onServerStarted(msg ServerStartedMsg) (Model, tea.Cmd) {
	m.serverAddr = msg.Address
	m.client = api.New(msg.Address)
	return m, m.checkHealth()
}

func (m Model) refreshMessages() Model {
	var chatBubbles []string
	for _, msg := range m.messages {
		chatBubbles = append(chatBubbles, RenderChatBubble(msg, m))
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
	}
	return m.refreshMessages(), nil
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
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.Response})
		if m.sessionId != "" {
			history.AppendMessage(m.sessionId, "assistant", msg.Response)
		}
	}
	m = m.refreshMessages()
	m.viewPort.GotoBottom()
	return m, nil
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
	return m, nil
}

const splashHeight = 19
const inputBoxHeight = 3

// onWindowSize updates layout dimensions when the terminal is resized.
func (m Model) onWindowSize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.width = msg.Width
	m.viewPort.SetWidth(msg.Width)
	m.viewPort.SetHeight(msg.Height - splashHeight - inputBoxHeight)
	return m, nil
}

// onKeyPress handles ctrl+c to quit/clear and enter to submit a message or command.
func (m Model) onKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.inputText.SetValue("")
		return m, nil

	case "enter":
		input := m.inputText.Value()
		if input != "" && !m.loading {
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
	}

	var cmd tea.Cmd
	m.inputText, cmd = m.inputText.Update(msg)
	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)
	return m, tea.Batch(cmd, vpCmd)
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
	case tea.WindowSizeMsg:
		return m.onWindowSize(msg)
	case tea.KeyPressMsg:
		return m.onKeyPress(msg)
	}
	return m.rebuildView(msg)
}
