package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"oc/internal/api"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func StartSSEListener(client *api.Client, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		resp, err := client.SubscribeGlobalEvents(ctx)
		if err != nil {
			return ControlRequestMsg{Err: err}
		}
		defer resp.Body.Close()

		partTypes := make(map[string]string)
		bufferedDeltas := make(map[string][]string)

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

			switch msg.Payload.Type {
			case "question.asked":
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
			case "permission.asked":
				var pp api.PermissionReqInfo
				if err := json.Unmarshal(msg.Payload.Properties, &pp); err != nil {
					continue
				}
				if program != nil {
					program.Send(PermissionRequestMsg{Request: &pp})
				}

			case "message.part.delta":
				var props struct {
					SessionID string `json:"sessionID"`
					PartID    string `json:"partID"`
					Field     string `json:"field"`
					Delta     string `json:"delta"`
				}
				if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
					continue
				}
				if props.Delta == "" {
					continue
				}

				if t, ok := partTypes[props.PartID]; ok {
					msg := ChatStreamMsg{SessionID: props.SessionID}
					if t == "reasoning" {
						msg.Reasoning = props.Delta
					} else {
						msg.Text = props.Delta
					}
					if program != nil {
						program.Send(msg)
					}
				} else {
					bufferedDeltas[props.PartID] = append(bufferedDeltas[props.PartID], props.Delta)
				}

			case "message.part.updated":
				var props struct {
					SessionID string `json:"sessionID"`
					Part      struct {
						ID   string `json:"id"`
						Type string `json:"type"`
					} `json:"part"`
				}
				if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
					continue
				}

				partTypes[props.Part.ID] = props.Part.Type

				if deltas, ok := bufferedDeltas[props.Part.ID]; ok {
					delete(bufferedDeltas, props.Part.ID)
					for _, delta := range deltas {
						msg := ChatStreamMsg{SessionID: props.SessionID}
						if props.Part.Type == "reasoning" {
							msg.Reasoning = delta
						} else {
							msg.Text = delta
						}
						if program != nil {
							program.Send(msg)
						}
					}
				}

			case "message.updated":
				var props struct {
					SessionID     string   `json:"sessionID"`
					MultiAgent    *bool    `json:"multi_agent"`
					Agents        int      `json:"agents"`
					Personalities []string `json:"personalities"`
					Complexity    string   `json:"complexity"`
					Reason        string   `json:"reason"`
					Info          struct {
						Role    string `json:"role"`
						Finish  string `json:"finish,omitempty"`
						ModelID string `json:"modelID,omitempty"`
					} `json:"info"`
				}
				if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
					continue
				}

				if props.MultiAgent != nil && program != nil {
					program.Send(MultiAgentPlanMsg{
						SessionID:     props.SessionID,
						Agents:        props.Agents,
						Personalities: props.Personalities,
						Complexity:    props.Complexity,
						Reason:        props.Reason,
					})
				}

				if props.Info.Role == "assistant" && props.Info.Finish != "" && program != nil {
					program.Send(ChatStreamMsg{
						SessionID: props.SessionID,
						Done:      true,
						ModelName: props.Info.ModelID,
					})
				}
			}
		}
	}
}
