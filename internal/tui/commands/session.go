package commands

import (
	"oc/internal/api"

	tea "charm.land/bubbletea/v2"
)

func FetchSessionUsage(client *api.Client, sessionId string) tea.Cmd {
	return func() tea.Msg {
		if sessionId == "" {
			return SessionUsageMsg{}
		}
		s, err := client.GetSession(sessionId)
		if err != nil {
			return SessionUsageMsg{Err: err}
		}
		tokens := 0
		limit := 0
		if t, ok := s["tokens"]; ok {
			if tokensMap, ok := t.(map[string]any); ok {
				for _, key := range []string{"input", "output"} {
					if v, ok := tokensMap[key]; ok {
						if f, ok := v.(float64); ok {
							tokens += int(f)
						}
					}
				}
			}
		}
		if l, ok := s["limit"]; ok {
			if limitMap, ok := l.(map[string]any); ok {
				if ctx, ok := limitMap["context"]; ok {
					if f, ok := ctx.(float64); ok {
						limit = int(f)
					}
				}
			}
		}
		return SessionUsageMsg{TokensUsed: tokens, ContextLimit: limit}
	}
}

func ShowSessionListCmd() tea.Cmd {
	return func() tea.Msg {
		return ShowSessionListMsg{}
	}
}
