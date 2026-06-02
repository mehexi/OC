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
		resp, err := m.client.SendMessage(sessionID, text)
		if err != nil {
			return ChatResponseMsg{Err: err, SessionID: sessionID}
		}
		return ChatResponseMsg{Response: resp, SessionID: sessionID}
	}
}

func (m Model) checkHealth() tea.Cmd {
	return func() tea.Msg {
		status, err := m.client.Health()
		return HealthCheckMsg{Status: status, Err: err}
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
