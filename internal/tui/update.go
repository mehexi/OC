package tui

import (
	"fmt"
	"oc/internal/api"
	"oc/internal/history"
	"oc/internal/tui/commands"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// onServerStarted initialises the API client and triggers a health check.
func (m Model) onServerStarted(msg ServerStartedMsg) (Model, tea.Cmd) {
	m.Server.serverAddr = msg.Address
	m.Server.client = api.New(msg.Address)
	return m, commands.CheckHealth(m.Server.client)
}

func (m Model) refreshMessages() Model {
	var chatBubbles []string
	for i, msg := range m.Chat.messages {
		bubble := RenderChatBubble(msg, m)
		if m.Modes.mode == modeVisual {
			lo, hi := m.Modes.visualAnchor, m.Modes.visualCursor
			if lo > hi {
				lo, hi = hi, lo
			}
			if i >= lo && i <= hi {
				bubble = lipgloss.NewStyle().Background(selectBgColor).Render(bubble)
			}
		}
		chatBubbles = append(chatBubbles, bubble)
	}
	m.Layout.viewPort.SetContent(strings.Join(chatBubbles, "\n\n"))
	return m
}

// onServerErr appends a server-error message to the chat.
func (m Model) onServerErr(msg ServerErrMsg) (Model, tea.Cmd) {
	m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Server error: " + msg.Err.Error()})
	return m.refreshMessages(), nil
}

// onHealthCheck records health status and shows a welcome or error message.
func (m Model) onHealthCheck(msg HealthCheckMsg) (Model, tea.Cmd) {
	m.Server.healthChecked = true
	if msg.Err != nil {
		m.Server.healthErr = msg.Err
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Server error: " + msg.Err.Error()})
	} else {
		m.Server.healthStatus = msg.Status
		return m.refreshMessages(), tea.Batch(commands.FetchProviders(m.Server.client), commands.FetchPath(m.Server.client))
	}
	return m.refreshMessages(), nil
}

// onProvidersInfo stores the default model name and model list.
func (m Model) onProvidersInfo(msg ProvidersInfoMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.Server.modelName = msg.ModelName
		m.Modes.models = msg.Models
		for _, model := range msg.Models {
			if model.ID == msg.ModelName {
				m.Server.modelID = model.ID
				m.Server.modelProviderID = model.ProviderID
				m.Server.modelName = model.Name
				m.Server.client.ModelID = model.ID
				m.Server.client.ModelProviderID = model.ProviderID
				break
			}
		}
	}
	return m, nil
}

// onPath stores the current working directory path and starts SSE listener.
func (m Model) onPath(msg PathMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.Server.currentPath = msg.Path
		m.Server.client.Directory = msg.Path
		return m, commands.StartSSEListener(m.Server.client, program, m)
	}
	return m, nil
}

// onSessionUsage stores token usage info from the current session.
func (m Model) onSessionUsage(msg SessionUsageMsg) (Model, tea.Cmd) {
	if msg.Err == nil {
		m.Server.tokensUsed = msg.TokensUsed
		m.Server.contextLimit = msg.ContextLimit
	}
	return m, nil
}

func (m Model) onPermissionRequest(msg PermissionRequestMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Permission error: " + msg.Err.Error()})
		return m.refreshMessages(), nil
	}
	if msg.Reply != "" {
		m.Flow.pendingPermission = nil
		var label string
		switch msg.Reply {
		case "once":
			label = "Permission granted (once)"
		case "always":
			label = "Permission granted (always)"
		case "reject":
			label = "Permission rejected"
		}
		if m.Flow.permissionMsgIndex >= 0 && m.Flow.permissionMsgIndex < len(m.Chat.messages) {
			m.Chat.messages[m.Flow.permissionMsgIndex].Content = label
		}
		m.Flow.permissionMsgIndex = -1
		return m.refreshMessages(), nil
	}
	m.Flow.pendingPermission = msg.Request
	m.Modes.mode = modePerm
	m.Layout.inputText.Blur()
	patterns := strings.Join(msg.Request.Patterns, ", ")
	m.Flow.permissionMsgIndex = len(m.Chat.messages)
	m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RolePermission, Content: "Permission: " + msg.Request.Permission + " on " + patterns + "\n  y=once  a=always  n=reject  esc=cancel"})
	return m.refreshMessages(), nil
}

// onControlRequest handles incoming questions from the question tool.
func (m Model) onControlRequest(msg ControlRequestMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.Chat.loading = false
		m.Flow.awaitingResponse = false
		m.Flow.pendingControl = nil
		m.Flow.currentQuestionIdx = 0
		m.Flow.questionAnswers = nil
		m.Layout.inputText.Placeholder = "Ask anything ..."
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Control request error: " + msg.Err.Error()})
		return m.refreshMessages(), nil
	}
	if msg.Request == nil {
		m.Chat.loading = false
		m.Layout.inputText.Placeholder = "Ask anything ..."
		if m.Flow.awaitingResponse {
			return m, nil
		}
		m.Flow.awaitingResponse = false
		if m.Flow.pendingControl != nil {
			m = m.syncLayout()
			var sb strings.Builder
			sb.WriteString("Answers:\n")
			for i, q := range m.Flow.pendingControl.Data.Questions {
				a := ""
				if i < len(m.Flow.questionAnswers) {
					a = m.Flow.questionAnswers[i]
				}
				fmt.Fprintf(&sb, "- %s: %s\n", q.Header, a)
			}
			m.Flow.pendingControl = nil
			m.Flow.currentQuestionIdx = 0
			m.Flow.questionAnswers = nil
			m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleUser, Content: strings.TrimSpace(sb.String())})
			m = m.refreshMessages()
			m.Layout.inputText.SetValue("")
			if m.Chat.streaming {
				return m, nil
			}
			m.Chat.loading = true
			return m, commands.SendChat(m.Server.client, m.Chat.sessionId, strings.TrimSpace(sb.String()))
		}
		if m.Chat.streaming {
			return m, nil
		}
		return m, nil
	}

	if m.Flow.pendingControl != nil {
		return m, nil
	}
	m.Flow.pendingControl = msg.Request
	m.Flow.currentQuestionIdx = 0
	m.Flow.questionAnswers = nil
	m.Flow.awaitingResponse = true
	m.Chat.loading = false

	m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: msg.Request.Data.Questions[0].Header})
	m = m.refreshMessages()
	return m.showQusList(), nil
}

// onStreamMsg handles SSE streaming chunks from the AI response.
func (m Model) onStreamMsg(msg ChatStreamMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.Chat.loading = false
		m.Chat.streaming = false
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Error: " + msg.Err.Error()})
		return m.refreshMessages(), nil
	}

	// Session-ID-only message (no text/reasoning/done/err) — response comes via SSE
	if msg.SessionID != "" && msg.Text == "" && msg.Reasoning == "" && !msg.Done && msg.Err == nil {
		if m.Chat.sessionId == "" {
			m.Chat.sessionId = msg.SessionID
			if len(m.Chat.messages) > 0 {
				history.AppendMessage(m.Chat.sessionId, string(RoleUser), m.Chat.messages[len(m.Chat.messages)-1].Content)
			}
		}
		return m, nil
	}

	if msg.Done {
		if !m.Chat.streaming {
			return m, nil
		}
		m.Chat.streaming = false
		if msg.FullReasoning != "" {
			for i := len(m.Chat.messages) - 1; i >= 0; i-- {
				if m.Chat.messages[i].Role == RoleAssistant {
					m.Chat.messages[i].Reasoning = msg.FullReasoning
					break
				}
			}
		}
		// Persist final assistant message
		for i := len(m.Chat.messages) - 1; i >= 0; i-- {
			if m.Chat.messages[i].Role == RoleAssistant {
				history.AppendMessage(m.Chat.sessionId, string(RoleAssistant), m.Chat.messages[i].Content)
				break
			}
		}
		return m.refreshMessages(), commands.FetchSessionUsage(m.Server.client, m.Chat.sessionId)
	}

	m.Chat.loading = false
	firstStream := !m.Chat.streaming
	m.Chat.streaming = true

	lastRole := m.Chat.messages[len(m.Chat.messages)-1].Role

	if len(m.Chat.messages) == 0 || (lastRole != RoleAssistant && lastRole != RoleJudge) {
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant})
	}
	last := &m.Chat.messages[len(m.Chat.messages)-1]

	if msg.Text != "" {
		last.Content += msg.Text
	}
	if msg.Reasoning != "" {
		last.Reasoning += msg.Reasoning
	}

	m = m.refreshMessages()
	m.Layout.viewPort.GotoBottom()
	if firstStream {
		return m, nil
	}
	return m, nil
}

func (m Model) onChatResponse(msg ChatResponseMsg) (Model, tea.Cmd) {
	m.Chat.loading = false
	if msg.Err != nil {
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: "Error: " + msg.Err.Error()})
	} else {
		if msg.SessionID != "" {
			isNew := m.Chat.sessionId == ""
			m.Chat.sessionId = msg.SessionID
			if isNew {
				history.AppendMessage(msg.SessionID, string(RoleUser), m.Chat.messages[len(m.Chat.messages)-1].Content)
			}
		}
		m.Chat.messages = append(m.Chat.messages, ChatMessage{Role: RoleAssistant, Content: msg.Response})
		if m.Chat.sessionId != "" {
			history.AppendMessage(m.Chat.sessionId, string(RoleAssistant), msg.Response)
		}
	}
	m = m.refreshMessages()
	m.Layout.viewPort.GotoBottom()
	return m, commands.FetchSessionUsage(m.Server.client, m.Chat.sessionId)
}

func (m Model) onMultiAgentPlan(msg MultiAgentPlanMsg) (Model, tea.Cmd) {
	m.Chat.loading = false
	m.Chat.agents = msg.Agents
	m.Chat.complexity = msg.Complexity
	m.Chat.reason = msg.Reason
	m.Chat.personalities = msg.Personalities
	if msg.MultiAgent {
		m.Chat.multiAgent = &msg.MultiAgent
	}

	m.Chat.messages = append(m.Chat.messages, ChatMessage{
		Role:    MessageRole(msg.Role),
		Content: msg.Content,
	})

	return m.refreshMessages(), nil
}

func (m Model) onLoadSession(msg LoadSessionMsg) (Model, tea.Cmd) {
	m.Chat.sessionId = msg.Session.ID
	m.Chat.messages = make([]ChatMessage, len(msg.Session.Messages))
	for i, msg := range msg.Session.Messages {
		m.Chat.messages[i] = ChatMessage{Role: MessageRole(msg.Role), Content: msg.Content}
	}
	m = m.refreshMessages()
	m.Layout.viewPort.GotoBottom()
	return m, commands.FetchSessionUsage(m.Server.client, m.Chat.sessionId)
}

const inputBoxHeight = 3

func (m Model) viewportHeight() int {
	headerHeight := lipgloss.Height(m.renderHeader())
	available := m.Layout.termHeight - headerHeight - inputBoxHeight
	if available < 1 {
		available = 1
	}
	switch m.Modes.mode {
	case modeQus:
		available -= m.Modes.qusHeight
	case modeSession:
		sessionLines := 2 + 5
		total := len(m.Modes.sessions)
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
	case modeModel:
		models := filteredModelList(m)
		modelLines := 2 + 5
		total := len(models)
		totalPages := (total + 5 - 1) / 5
		if totalPages > 1 {
			modelLines += 2
		}
		available -= modelLines
	}
	if available < 1 {
		available = 1
	}
	return available
}

func (m Model) syncLayout() Model {
	m.Layout.viewPort.SetWidth(m.Layout.width)
	m.Layout.viewPort.SetHeight(m.viewportHeight())
	m.Layout.inputText.SetWidth(m.Layout.width - 6)
	return m
}

// onWindowSize updates layout dimensions when the terminal is resized.
func (m Model) onWindowSize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.Layout.width = msg.Width
	m.Layout.termHeight = msg.Height
	m = m.syncLayout()
	return m, nil
}

// onKeyPress dispatches key events to the active mode handler.
func (m Model) onKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch m.Modes.mode {
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
	case modeModel:
		return m.onModelKey(msg)
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
	m.Layout.inputText, cmd = m.Layout.inputText.Update(msg)
	var vpCmd tea.Cmd
	m.Layout.viewPort, vpCmd = m.Layout.viewPort.Update(msg)
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
		return m.onPermissionRequest(msg)
	case ChatResponseMsg:
		return m.onChatResponse(msg)
	case MultiAgentPlanMsg:
		return m.onMultiAgentPlan(msg)
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
