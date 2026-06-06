package commands

import (
	"oc/internal/api"

	tea "charm.land/bubbletea/v2"
)

func SendChat(client *api.Client, sessionId string, text string) tea.Cmd {
	return func() tea.Msg {
		sessionID := sessionId
		if sessionID == "" {
			id, err := client.CreateSession(text)
			if err != nil {
				return ChatStreamMsg{Err: err}
			}
			sessionID = id
		}

		resp, err := client.SendMessageRaw(sessionID, text)
		if err != nil {
			return ChatStreamMsg{Err: err, SessionID: sessionID}
		}
		resp.Body.Close()

		return ChatStreamMsg{SessionID: sessionID}
	}
}

func AddAssistantMsg(content string) tea.Cmd {
	return func() tea.Msg {
		return ChatResponseMsg{Response: content}
	}
}

func CreateSubSession(client *api.Client, title, personality string) tea.Cmd {
	return func() tea.Msg {
		id, err := client.CreateSession(title)
		if err != nil {
			return SubAgentSpawnedMsg{Err: err, Personality: personality}
		}
		return SubAgentSpawnedMsg{SessionID: id, Personality: personality}
	}
}

func SendToSession(client *api.Client, sessionID, text string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.SendMessageRaw(sessionID, text)
		if err != nil {
			return ChatStreamMsg{Err: err, SessionID: sessionID}
		}
		resp.Body.Close()
		return nil
	}
}
