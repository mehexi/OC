package tui

import (
	"fmt"
	"oc/internal/history"
	"strings"

	tea "charm.land/bubbletea/v2"
)

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
		resp, modelName, err := m.client.SendMessage(sessionID, text)
		if err != nil {
			return ChatResponseMsg{Err: err, SessionID: sessionID}
		}
		return ChatResponseMsg{Response: resp, SessionID: sessionID, ModelName: modelName}
	}
}

func (m Model) checkHealth() tea.Cmd {
	return func() tea.Msg {
		status, err := m.client.Health()
		return HealthCheckMsg{Status: status, Err: err}
	}
}

func (m Model) fetchPath() tea.Cmd {
	return func() tea.Msg {
		p, err := m.client.GetPath()
		if err != nil {
			return PathMsg{Err: err}
		}
		return PathMsg{Path: p}
	}
}

func (m Model) fetchProviders() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.GetProviders()
		if err != nil {
			return ProvidersInfoMsg{Err: err}
		}
		modelName := ""
		if len(resp.Default) > 0 {
			for _, v := range resp.Default {
				modelName = v
				break
			}
		}
		return ProvidersInfoMsg{ModelName: modelName}
	}
}

func (m Model) fetchSessionUsage() tea.Cmd {
	return func() tea.Msg {
		if m.sessionId == "" {
			return SessionUsageMsg{}
		}
		s, err := m.client.GetSession(m.sessionId)
		if err != nil {
			return SessionUsageMsg{Err: err}
		}
		tokens := 0
		limit := 0
		model := ""
		if v, ok := s["model"]; ok {
			model, _ = v.(string)
		}
		if u, ok := s["usage"]; ok {
			if usage, ok := u.(map[string]interface{}); ok {
				for _, key := range []string{"tokens_used", "total_tokens", "input_tokens"} {
					if t, ok := usage[key]; ok {
						if f, ok := t.(float64); ok {
							tokens = int(f)
							break
						}
					}
				}
				for _, key := range []string{"context_limit", "max_context", "context_window", "max_tokens"} {
					if l, ok := usage[key]; ok {
						if f, ok := l.(float64); ok {
							limit = int(f)
							break
						}
					}
				}
			}
		}
		return SessionUsageMsg{TokensUsed: tokens, ContextLimit: limit, ModelName: model}
	}
}

func (m Model) addAssistantMsg(content string) tea.Cmd {
	return func() tea.Msg {
		return ChatResponseMsg{Response: content}
	}
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
