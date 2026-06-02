package tui

import (
	"fmt"
	"oc/internal/api"
	"oc/internal/history"
	"oc/internal/server"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func IntialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything ..."
	styles := ti.Styles()
	styles.Focused.Placeholder = lipgloss.NewStyle().Background(inputBgColor)
	styles.Focused.Text = lipgloss.NewStyle().Background(inputBgColor)
	ti.SetStyles(styles)
	ti.SetWidth(50)
	ti.Focus()

	vp := viewport.New()
	vp.SoftWrap = true

	return Model{
		viewPort:  vp,
		inputText: ti,
		messages:  []ChatMessage{},
		sessionId: "",
		loading:   false,
		width:     0,
		height:    0,
	}
}

func initServer() tea.Cmd {
	return func() tea.Msg {
		addr, err := server.EnsureRunning()
		if err != nil {
			return ServerErrMsg{err}
		}
		return ServerStartedMsg{Address: addr}
	}
}

func (m Model) checkHealth() tea.Cmd {
	return func() tea.Msg {
		status, err := m.client.Health()
		return HealthCheckMsg{Status: status, Err: err}
	}
}

func (m Model) sendChat(text string) tea.Cmd {
	return func() tea.Msg {
		sessionID := m.sessionId
		if sessionID == "" {
			id, err := m.client.CreateSession(text)
			if err != nil {
				return ChatResponseMsg{Err: err}
			}
			sessionID = id
		}
		resp, err := m.client.SendMessage(sessionID, text)
		if err != nil {
			return ChatResponseMsg{Err: err, SessionID: sessionID}
		}
		return ChatResponseMsg{Response: resp, SessionID: sessionID}
	}
}

func (m Model) Init() tea.Cmd {
	return initServer()
}

func (m Model) handleCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/sessions":
		sessions, err := history.ListSessions()
		if err != nil {
			return m.addAssistantMsg("Error listing sessions: " + err.Error())
		}
		if len(sessions) == 0 {
			return m.addAssistantMsg("No past sessions.")
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("─── %d Past Sessions ───\n\n", len(sessions)))
		for i, s := range sessions {
			t := s.CreatedAt.Format("Jan 02 15:04")
			b.WriteString(fmt.Sprintf("  %d. %s\n     %s · %d msgs\n\n", i+1, s.Title, t, s.Count))
		}
		b.WriteString("Load one with  /load <number>")
		return m.addAssistantMsg(b.String())

	case "/load":
		if len(parts) < 2 {
			return m.addAssistantMsg("Usage: /load <number>")
		}
		sessions, err := history.ListSessions()
		if err != nil {
			return m.addAssistantMsg("Error: " + err.Error())
		}
		var n int
		if _, err := fmt.Sscanf(parts[1], "%d", &n); err != nil || n < 1 || n > len(sessions) {
			return m.addAssistantMsg(fmt.Sprintf("Invalid number. Choose 1-%d.", len(sessions)))
		}
		return func() tea.Msg {
			s, err := history.LoadSession(sessions[n-1].ID)
			if err != nil {
				return ChatResponseMsg{Err: fmt.Errorf("load session: %w", err)}
			}
			return LoadSessionMsg{Session: s}
		}

	default:
		return m.addAssistantMsg("Unknown command: " + parts[0] + "\nAvailable: /sessions, /load <n>, /session new")
	}
}

func (m Model) addAssistantMsg(content string) tea.Cmd {
	return func() tea.Msg {
		return ChatResponseMsg{Response: content}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ServerStartedMsg:
		m.serverAddr = msg.Address
		m.client = api.New(msg.Address)
		return m, m.checkHealth()

	case ServerErrMsg:
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Server error: " + msg.err.Error()})

	case HealthCheckMsg:
		m.healthChecked = true
		if msg.Err != nil {
			m.healthErr = msg.Err
			m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Server error: " + msg.Err.Error()})
		} else {
			m.healthStatus = msg.Status
			welcome := fmt.Sprintf("Server v%s connected. Type /sessions for history.", msg.Status.Version)
			m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: welcome})
		}

	case ChatResponseMsg:
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
		m.viewPort.GotoBottom()

	case LoadSessionMsg:
		m.sessionId = msg.Session.ID
		m.messages = make([]ChatMessage, len(msg.Session.Messages))
		for i, msg := range msg.Session.Messages {
			m.messages[i] = ChatMessage{Role: msg.Role, Content: msg.Content}
		}
		m.viewPort.GotoBottom()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewPort.SetWidth(msg.Width)
		m.viewPort.SetHeight(msg.Height - 3)

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.inputText.Value() == "" {
				server.KillServer()
				return m, tea.Quit
			}
			m.inputText.SetValue("")

		case "enter":
			input := m.inputText.Value()
			if input != "" && !m.loading {
				if strings.HasPrefix(input, "/") {
					m.inputText.SetValue("")
					parts := strings.Fields(input)
					if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
						m.sessionId = ""
						m.messages = nil
						return m, m.addAssistantMsg("Started a new session.")
					}
					return m, m.handleCommand(input)
				}
				m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
				m.inputText.SetValue("")
				m.loading = true
				return m, m.sendChat(input)
			}
		}
	}

	var chatBubbles []string
	for _, msg := range m.messages {
		chatBubbles = append(chatBubbles, RenderChatBubble(msg, m))
	}
	m.viewPort.SetContent(strings.Join(chatBubbles, "\n\n"))

	var cmd tea.Cmd
	m.inputText, cmd = m.inputText.Update(msg)

	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)

	return m, tea.Batch(cmd, vpCmd)
}

func (m Model) View() tea.View {
	return ChatView(m)
}
