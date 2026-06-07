package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"oc/internal/api"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type multiAgentProvider interface {
	MultiAgent() bool
}

func StartSSEListener(client *api.Client, program *tea.Program, provider multiAgentProvider) tea.Cmd {
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

			if msg.Payload.Type == "session.status" {
				var props struct {
					SessionID string `json:"sessionID"`
					Status    struct {
						Type   string `json:"type"`
						Action *struct {
							Title string `json:"title"`
							Link  string `json:"link"`
						} `json:"action"`
					} `json:"status"`
				}
				if err := json.Unmarshal(msg.Payload.Properties, &props); err == nil {
					if props.Status.Type == "retry" && props.Status.Action != nil {
						program.Send(ChatStreamMsg{
							Done: true,
							Err:  fmt.Errorf("%s — %s", props.Status.Action.Title, props.Status.Action.Link),
						})
					}
				}
				continue
			}
			if provider.MultiAgent() {
				handleMultiAgentPlan(msg, program)
			} else {
				handleSSEEvent(msg, program, partTypes, bufferedDeltas)
			}

		}
	}
}

func handleMultiAgentPlan(msg api.SSEMessage, program *tea.Program) {
	if msg.Payload.Type != "message.part.updated" {
		return
	}

	var props struct {
		Part struct {
			Type string `json:"type"`
			Text string `json:"text"`
			Time struct {
				End int64 `json:"end"`
			} `json:"time"`
		} `json:"part"`
	}
	if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
		return
	}

	if props.Part.Type != "text" || props.Part.Time.End == 0 {
		return
	}

	var verdict struct {
		MultiAgent    bool     `json:"multi_agent"`
		Agents        int      `json:"agents"`
		Personalities []string `json:"personalities"`
		Complexity    string   `json:"complexity"`
		Reason        string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(props.Part.Text), &verdict); err != nil {
		program.Send(MultiAgentPlanMsg{
			Reason: props.Part.Text,
		})
		return
	}

	program.Send(MultiAgentPlanMsg{
		Reason: verdict.Reason,
	})
}

func handleSSEEvent(msg api.SSEMessage, program *tea.Program, partTypes map[string]string, bufferedDeltas map[string][]string) {
	switch msg.Payload.Type {
	case "question.asked":
		var qp api.QuestionProperties
		if err := json.Unmarshal(msg.Payload.Properties, &qp); err != nil {
			return
		}
		if len(qp.Questions) == 0 {
			return
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
			return
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
			return
		}
		if props.Delta == "" {
			return
		}

		if t, ok := partTypes[props.PartID]; ok {
			m := ChatStreamMsg{SessionID: props.SessionID}
			if t == "reasoning" {
				m.Reasoning = props.Delta
			} else {
				m.Text = props.Delta
			}
			if program != nil {
				program.Send(m)
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
			return
		}

		partTypes[props.Part.ID] = props.Part.Type

		if deltas, ok := bufferedDeltas[props.Part.ID]; ok {
			delete(bufferedDeltas, props.Part.ID)
			for _, delta := range deltas {
				m := ChatStreamMsg{SessionID: props.SessionID}
				if props.Part.Type == "reasoning" {
					m.Reasoning = delta
				} else {
					m.Text = delta
				}
				if program != nil {
					program.Send(m)
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
			return
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
