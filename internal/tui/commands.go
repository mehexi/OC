package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"oc/internal/api"
	"oc/internal/history"
	"oc/internal/server"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) sendChat(text string) tea.Cmd {
	return func() tea.Msg {
		sessionID := m.sessionId
		if sessionID == "" {
			id, err := m.client.CreateSession(text)
			if err != nil {
				return ChatStreamMsg{Err: err}
			}
			sessionID = id
		}

		resp, err := m.client.SendMessageRaw(sessionID, text)
		if err != nil {
			return ChatStreamMsg{Err: err, SessionID: sessionID}
		}

		go streamSSE(resp, sessionID)
		return ChatStreamMsg{SessionID: sessionID}
	}
}

type streamPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type streamInfo struct {
	ModelID string `json:"modelID"`
}

type streamResponse struct {
	Info  streamInfo   `json:"info"`
	Parts []streamPart `json:"parts"`
}

func streamSSE(resp *http.Response, sessionID string) {
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	firstLine, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		if program != nil {
			program.Send(ChatStreamMsg{Err: err, SessionID: sessionID, Done: true})
		}
		return
	}

	trimmed := strings.TrimSpace(firstLine)

	var part streamPart
	if json.Unmarshal([]byte(trimmed), &part) == nil && part.Type != "" {
		streamNDJSON(trimmed, reader, sessionID)
	} else {
		rest, err := io.ReadAll(reader)
		if err != nil {
			if program != nil {
				program.Send(ChatStreamMsg{Err: err, SessionID: sessionID, Done: true})
			}
			return
		}
		streamSingleJSON(trimmed+string(rest), sessionID)
	}
}

func streamNDJSON(firstLine string, reader *bufio.Reader, sessionID string) {
	var fullText string
	var modelName string

	processLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}

		var part streamPart
		if err := json.Unmarshal([]byte(line), &part); err != nil || part.Type == "" {
			var resp streamResponse
			if err := json.Unmarshal([]byte(line), &resp); err == nil && resp.Info.ModelID != "" {
				modelName = resp.Info.ModelID
				for _, p := range resp.Parts {
					sendPart(p, &fullText, modelName, sessionID)
				}
			}
			return
		}

		sendPart(part, &fullText, modelName, sessionID)
	}

	processLine(firstLine)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		processLine(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		if program != nil {
			program.Send(ChatStreamMsg{Err: err, SessionID: sessionID, Done: true})
		}
		return
	}

	if program != nil {
		program.Send(ChatStreamMsg{
			Text:      "",
			FullText:  fullText,
			SessionID: sessionID,
			Done:      true,
			ModelName: modelName,
		})
	}
}

func streamSingleJSON(body string, sessionID string) {
	var msg streamResponse
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		if program != nil {
			program.Send(ChatStreamMsg{Err: err, SessionID: sessionID, Done: true})
		}
		return
	}

	var fullText string
	modelName := msg.Info.ModelID

	for _, p := range msg.Parts {
		sendPart(p, &fullText, modelName, sessionID)
	}

	if program != nil {
		program.Send(ChatStreamMsg{
			Text:      "",
			FullText:  fullText,
			SessionID: sessionID,
			Done:      true,
			ModelName: modelName,
		})
	}
}

func sendPart(p streamPart, fullText *string, modelName string, sessionID string) {
	if p.Type == "text" {
		*fullText += p.Text
		if program != nil {
			program.Send(ChatStreamMsg{
				Text:      p.Text,
				FullText:  *fullText,
				SessionID: sessionID,
				Done:      false,
			})
		}
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
		if t, ok := s["tokens"]; ok {
			if tokensMap, ok := t.(map[string]interface{}); ok {
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
			if limitMap, ok := l.(map[string]interface{}); ok {
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

func (m Model) addAssistantMsg(content string) tea.Cmd {
	return func() tea.Msg {
		return ChatResponseMsg{Response: content}
	}
}

func (m Model) showSessionListCmd() tea.Cmd {
	return func() tea.Msg {
		return ShowSessionListMsg{}
	}
}

func (m Model) handleCommand(input string) tea.Cmd {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/help":
		return m.addAssistantMsg(
			"Commands:\n" +
				"  /help          Show this help\n" +
				"  /sessions      List and load past sessions\n" +
				"  /session new   Start a fresh session\n" +
				"  /clear         Clear chat messages\n" +
				"  /retry         Re-send last user message\n" +
				"  /load <n>      Load session by number\n" +
				"  /tokens        Show token usage\n" +
				"  /exit          Quit the app",
		)

	case "/sessions":
		return m.showSessionListCmd()

	case "/tokens":
		usage := fmt.Sprintf("Model: %s  |  Tokens: %d / %d  |  Remaining: %d",
			m.modelName, m.tokensUsed, m.contextLimit, m.contextLimit-m.tokensUsed)
		return m.addAssistantMsg(usage)

	case "/exit":
		server.KillServer()
		return tea.Quit

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
		return m.addAssistantMsg("Unknown: " + parts[0] + "\nTry /help for available commands.")
	}
}

func (m Model) startSSEListener() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		resp, err := m.client.SubscribeGlobalEvents(ctx)
		if err != nil {
			return ControlRequestMsg{Err: err}
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return nil
			}
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var msg api.SSEMessage
			if err := json.Unmarshal([]byte(data), &msg); err != nil {
				continue
			}
			if msg.Payload.Type != "question.asked" {
				continue
			}

			var qp api.QuestionProperties
			if err := json.Unmarshal(msg.Payload.Properties, &qp); err != nil {
				continue
			}
			if len(qp.Questions) == 0 {
				continue
			}

			cr := &api.ControlRequest{
				ID:   qp.ID,
				Type: "question.asked",
				Data: api.ControlRequestData{
					Questions: qp.Questions,
				},
			}
			if program != nil {
				program.Send(ControlRequestMsg{Request: cr})
			}
		}
	}
}

func (m Model) sendControlResponse() tea.Cmd {
	return func() tea.Msg {
		var answers [][]string
		for i := range m.pendingControl.Data.Questions {
			a := ""
			if i < len(m.questionAnswers) {
				a = m.questionAnswers[i]
			}
			answers = append(answers, []string{a})
		}
		err := m.client.ReplyToQuestion(m.pendingControl.ID, answers)
		if err != nil {
			return ControlRequestMsg{Err: err}
		}
		return ControlRequestMsg{}
	}
}
