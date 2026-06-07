package tui

import (
	"fmt"
	"oc/internal/history"
	"oc/internal/server"
	"oc/internal/sysprompt"
	"oc/internal/tui/commands"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleCommand(input string) (Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return m, nil
	}

	switch parts[0] {
	case "/help":
		return m, commands.AddAssistantMsg(
			"Commands:\n" +
				"  /help          Show this help\n" +
				"  /sessions      List and load past sessions\n" +
				"  /session new   Start a fresh session\n" +
				"  /clear         Clear chat messages\n" +
				"  /multiagent    Toggle multi-agent mode — spawns sub-agents to work on tasks in parallel\n" +
				"  /retry         Re-send last user message\n" +
				"  /load <n>      Load session by number\n" +
				"  /tokens        Show token usage\n" +
				"  /exit          Quit the app",
		)

	case "/sessions":
		return m, commands.ShowSessionListCmd()

	case "/tokens":
		usage := fmt.Sprintf("Model: %s  |  Tokens: %d / %d  |  Remaining: %d",
			m.modelName, m.tokensUsed, m.contextLimit, m.contextLimit-m.tokensUsed)
		return m, commands.AddAssistantMsg(usage)

	case "/exit":
		server.KillServer()
		return m, tea.Quit

	case "/load":
		if len(parts) < 2 {
			return m, commands.AddAssistantMsg("Usage: /load <number>")
		}
		sessions, err := history.ListSessions()
		if err != nil {
			return m, commands.AddAssistantMsg("Error: " + err.Error())
		}
		var n int
		if _, err := fmt.Sscanf(parts[1], "%d", &n); err != nil || n < 1 || n > len(sessions) {
			return m, commands.AddAssistantMsg(fmt.Sprintf("Invalid number. Choose 1-%d.", len(sessions)))
		}
		return m, func() tea.Msg {
			s, err := history.LoadSession(sessions[n-1].ID)
			if err != nil {
				return ChatResponseMsg{Err: fmt.Errorf("load session: %w", err)}
			}
			return LoadSessionMsg{Session: s}
		}
	case "/multiagent":
		if m.multiAgent != nil {
			*m.multiAgent = !*m.multiAgent
		}
		return m, commands.SendChat(m.client, m.sessionId, sysprompt.JudgeSysPrompt())
	default:
		return m, commands.AddAssistantMsg("Unknown: " + parts[0] + "\nTry /help for available commands.")
	}
}
