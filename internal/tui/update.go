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
	m.serverAddr = msg.Address
	m.client = api.New(msg.Address)
	return m, commands.CheckHealth(m.client)
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
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Server error: " + msg.Err.Error()})
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
		return m.refreshMessages(), tea.Batch(commands.FetchProviders(m.client), commands.FetchPath(m.client))
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
		return m, commands.StartSSEListener(m.client, program)
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

func (m Model) onPermissionRequest(msg PermissionRequestMsg) (Model, tea.Cmd) {
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
			return m, commands.SendChat(m.client, m.sessionId, strings.TrimSpace(sb.String()))
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

	// Route SSE events to sub-agent handlers or ignore
	if msg.SessionID != "" && m.sessionId != "" && msg.SessionID != m.sessionId {
		if idx, ok := m.agentSessions[msg.SessionID]; ok {
			return m.onSubAgentStream(msg, idx)
		}
		return m, nil
	}

	if msg.Done {
		if !m.streaming {
			return m, nil
		}
		m.streaming = false
		if msg.ModelName != "" {
			m.modelName = msg.ModelName
		}
		if msg.FullReasoning != "" {
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "assistant" {
					m.messages[i].Reasoning = msg.FullReasoning
					break
				}
			}
		}
		// Persist final assistant message
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == "assistant" {
				history.AppendMessage(m.sessionId, "assistant", m.messages[i].Content)
				break
			}
		}
		return m.refreshMessages(), commands.FetchSessionUsage(m.client, m.sessionId)
	}

	m.loading = false
	firstStream := !m.streaming
	m.streaming = true

	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Role != "assistant" {
		m.messages = append(m.messages, ChatMessage{Role: "assistant"})
	}

	last := &m.messages[len(m.messages)-1]

	if msg.Text != "" {
		last.Content += msg.Text
	}
	if msg.Reasoning != "" {
		last.Reasoning += msg.Reasoning
	}

	m = m.refreshMessages()
	m.viewPort.GotoBottom()
	if firstStream {
		return m, nil
	}
	return m, nil
}

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
	return m, commands.FetchSessionUsage(m.client, m.sessionId)
}

func (m Model) onSubAgentStream(msg ChatStreamMsg, agentIdx int) (Model, tea.Cmd) {
	agent := &m.subAgents[agentIdx]

	if msg.Err != nil {
		agent.Status = "done"
		return m.checkDebateProgress()
	}

	if msg.Done {
		agent.Status = "done"
		return m.checkDebateProgress()
	}

	if len(agent.Messages) == 0 || agent.Messages[len(agent.Messages)-1].Role != "assistant" {
		agent.Messages = append(agent.Messages, ChatMessage{Role: "assistant"})
	}

	last := &agent.Messages[len(agent.Messages)-1]
	if msg.Text != "" {
		last.Content += msg.Text
	}
	return m, nil
}

func (m Model) onMultiAgentPlan(msg MultiAgentPlanMsg) (Model, tea.Cmd) {
	m.loading = false
	m.debatePhase = "spawning"

	task := msg.Reason
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			task = m.messages[i].Content
			break
		}
	}
	m.debateTask = task

	m.messages = append(m.messages, ChatMessage{
		Role:    "assistant",
		Content: fmt.Sprintf("[Multi-Agent Debate] %d agents analyzing your task", msg.Agents),
	})
	m = m.refreshMessages()

	var cmds []tea.Cmd
	for i, personality := range msg.Personalities {
		title := fmt.Sprintf("[%s] %s", personality, truncate(task, 80))
		agent := SubAgent{
			ID:          fmt.Sprintf("agent-%d", i),
			Personality: personality,
			Status:      "spawning",
		}
		m.subAgents = append(m.subAgents, agent)
		cmds = append(cmds, commands.CreateSubSession(m.client, title, personality))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) onSubAgentSpawned(msg SubAgentSpawnedMsg) (Model, tea.Cmd) {
	if msg.Err != nil {
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "Error spawning agent: " + msg.Err.Error()})
		return m.refreshMessages(), nil
	}

	for i := range m.subAgents {
		if m.subAgents[i].SessionID == "" {
			m.subAgents[i].SessionID = msg.SessionID
			m.subAgents[i].Status = "thinking"
			m.agentSessions[msg.SessionID] = i
			break
		}
	}
	m = m.refreshMessages()

	for _, a := range m.subAgents {
		if a.SessionID == "" {
			return m, nil
		}
	}

	m.messages = append(m.messages, ChatMessage{
		Role:    "assistant",
		Content: fmt.Sprintf("[Multi-Agent Debate] All %d agents ready, starting round 1", len(m.subAgents)),
	})
	m = m.refreshMessages()
	var cmd tea.Cmd
	m, cmd = m.startDebateRound(1)
	return m, cmd
}

func (m Model) startDebateRound(round int) (Model, tea.Cmd) {
	m.debateRound = round
	m.debatePhase = "debate"

	for i := range m.subAgents {
		m.subAgents[i].Status = "thinking"
	}

	var cmds []tea.Cmd
	for _, agent := range m.subAgents {
		prompt := buildDebatePrompt(agent, round, m.debateTask, m.subAgents)
		cmds = append(cmds, commands.SendToSession(m.client, agent.SessionID, prompt))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) checkDebateProgress() (Model, tea.Cmd) {
	for _, a := range m.subAgents {
		if a.Status != "done" {
			return m, nil
		}
	}

	for i := range m.subAgents {
		m.subAgents[i].Status = "thinking"
	}

	var cmd tea.Cmd
	switch m.debateRound {
	case 1:
		m.messages = append(m.messages, ChatMessage{
			Role:    "assistant",
			Content: "[Multi-Agent Debate] Round 1 complete, starting round 2",
		})
		m = m.refreshMessages()
		m, cmd = m.startDebateRound(2)
	case 2:
		m.messages = append(m.messages, ChatMessage{
			Role:    "assistant",
			Content: "[Multi-Agent Debate] Round 2 complete, starting round 3",
		})
		m = m.refreshMessages()
		m, cmd = m.startDebateRound(3)
	case 3:
		m.messages = append(m.messages, ChatMessage{
			Role:    "assistant",
			Content: "[Multi-Agent Debate] Debate complete, synthesizing final answer",
		})
		m = m.refreshMessages()
		cmd = m.startSynthesis()
	}

	return m, cmd
}

func (m Model) startSynthesis() tea.Cmd {
	m.debatePhase = "synthesis"

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Multi-Agent Debate on: %s\n\n", m.debateTask))
	for _, agent := range m.subAgents {
		sb.WriteString(fmt.Sprintf("=== %s ===\n", agent.Personality))
		for _, msg := range agent.Messages {
			sb.WriteString(msg.Content)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	synthesisPrompt := fmt.Sprintf(`The following is a multi-agent debate about the task: "%s"

The agents produced these analyses:

%s

Based on this debate, provide a comprehensive final answer that incorporates the best points from each perspective.`,
		m.debateTask, strings.TrimSpace(sb.String()))

	return commands.SendChat(m.client, m.sessionId, synthesisPrompt)
}

func buildDebatePrompt(agent SubAgent, round int, task string, allAgents []SubAgent) string {
	var sb strings.Builder

	descriptions := map[string]string{
		"Architect":       "You think in systems, patterns, and long-term design. Focus on architecture, modularity, and scalability.",
		"Skeptic":         "You challenge assumptions and find flaws and edge cases. Identify what could go wrong.",
		"Pragmatist":      "You advocate for the fastest working solution with no over-engineering. Focus on simplicity and practicality.",
		"Security":        "You find attack surfaces and vulnerabilities. Focus on security implications.",
		"Devil's Advocate": "You argue the opposite approach to uncover blind spots in the proposed solutions.",
		"Researcher":      "You analyze tradeoffs, prior art, and known pitfalls. Provide evidence-based insights.",
		"Performance":     "You focus on bottlenecks, scalability, and efficiency. Identify performance implications.",
	}

	desc := descriptions[agent.Personality]
	if desc == "" {
		desc = "Provide your unique perspective on the task."
	}

	sb.WriteString(fmt.Sprintf("You are an AI with the personality of %s.\n%s\n\n", agent.Personality, desc))

	switch round {
	case 1:
		sb.WriteString(fmt.Sprintf("Analyze this task from your perspective:\n\n%s\n\n", task))
		sb.WriteString("Provide specific technical points and numbered arguments. Be concise but thorough.")
	case 2:
		sb.WriteString(fmt.Sprintf("Original task:\n%s\n\n", task))
		sb.WriteString("Here are all the responses from the first round. Review each one and:\n")
		sb.WriteString("1. Score each agent's key points from 1-5\n")
		sb.WriteString("2. Identify which points you agree/disagree with\n")
		sb.WriteString("3. Provide your refined perspective\n\n")
		for _, other := range allAgents {
			if other.ID == agent.ID {
				continue
			}
			sb.WriteString(fmt.Sprintf("=== %s ===\n", other.Personality))
			for _, m := range other.Messages {
				sb.WriteString(m.Content)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	case 3:
		sb.WriteString(fmt.Sprintf("Original task:\n%s\n\n", task))
		sb.WriteString("After reviewing all perspectives, provide your FINAL refined analysis.\n")
		sb.WriteString("Focus on the strongest points that emerged from the debate.\n")
		sb.WriteString("Format your response as numbered key points that should be included in the final solution.")
	}

	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func (m Model) onLoadSession(msg LoadSessionMsg) (Model, tea.Cmd) {
	m.sessionId = msg.Session.ID
	m.messages = make([]ChatMessage, len(msg.Session.Messages))
	for i, msg := range msg.Session.Messages {
		m.messages[i] = ChatMessage{Role: msg.Role, Content: msg.Content}
	}
	m = m.refreshMessages()
	m.viewPort.GotoBottom()
	return m, commands.FetchSessionUsage(m.client, m.sessionId)
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
		return m.onPermissionRequest(msg)
	case ChatResponseMsg:
		return m.onChatResponse(msg)
	case MultiAgentPlanMsg:
		return m.onMultiAgentPlan(msg)
	case SubAgentSpawnedMsg:
		return m.onSubAgentSpawned(msg)
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
