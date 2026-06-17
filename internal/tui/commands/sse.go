package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"oc/internal/api"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type multiAgentProvider interface {
	MultiAgent() bool
}

func StartSSEListener(client *api.Client, program *tea.Program, provider multiAgentProvider) tea.Cmd {
	return func() tea.Msg {
		for {
			func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				resp, err := client.SubscribeGlobalEvents(ctx)
				if err != nil {
					program.Send(ControlRequestMsg{Err: err})
					return
				}
				defer resp.Body.Close()

				partTypes := make(map[string]string)
				bufferedDeltas := make(map[string][]string)

				reader := bufio.NewReader(resp.Body)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						time.Sleep(1 * time.Second)
						return
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
						handleMultiAgentPlan(client, msg, program)
					} else {
						handleSSEEvent(msg, program, partTypes, bufferedDeltas)
					}

				}
			}()
		}
	}
}

type multiAgentProperties struct {
	SessionID string `json:"sessionID"`
	Part      struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Time struct {
			End int64 `json:"end"`
		} `json:"time"`
	} `json:"part"`
}

func handleMultiAgentPlan(client *api.Client, msg api.SSEMessage, program *tea.Program) bool {

	if msg.Payload.Type != "message.part.updated" {
		return false
	}

	var props multiAgentProperties
	if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
		return false
	}

	if props.Part.Type != "text" || props.Part.Time.End == 0 {
		return false
	}

	if agent, ok := findSubagent(props.SessionID); ok {
		program.Send(MultiAgentPlanMsg{
			SessionID: props.SessionID,
			Role:      agent.Role,
			Content:   props.Part.Text,
			Done:      true,
		})
		return true
	}

	var v verdict

	if err := json.Unmarshal([]byte(props.Part.Text), &v); err == nil {
		handleVerdict(client, v, props, program)
	} else {

		program.Send(MultiAgentPlanMsg{
			SessionID: props.SessionID,
			Role:      RoleJudge,
			Content:   props.Part.Text,
		})

	}

	program.Send(MultiAgentPlanMsg{
		SessionID: props.SessionID,
		Role:      RoleJudge,
		Content:   v.Reason,
	})
	return true
}

func handleVerdict(c *api.Client, v verdict, props multiAgentProperties, program *tea.Program) {

	if v.MultiAgent {

		var createdID []string

		for _, p := range v.Personalities {
			data, err := handleSubAgentCreate(c, p)
			if err != nil {
				program.Send(MultiAgentPlanMsg{
					SessionID: props.SessionID,
					Role:      RoleJudge,
					Content:   "failed to create subagents",
					Done:      false,
				})
				continue
			}

			if err := handleSubagentTask(c, data.SessionID); err != nil {
				program.Send(MultiAgentPlanMsg{
					SessionID: props.SessionID,
					Role:      RoleJudge,
					Content:   fmt.Sprintf("failed to send task to subagent: %v", err),
					Done:      false,
				})
			}

			createdID = append(createdID, data.SessionID)
			Subagents = append(Subagents, data)
		}

		program.Send(MultiAgentPlanMsg{
			SessionID: props.SessionID,
			Role:      RoleJudge,
			Content:   fmt.Sprintf("created agents: %s", strings.Join(createdID, ", ")),
			Done:      true,
		})
	}
}

var personalityRoleMap = map[string]SubagentRole{
	"judge":            RoleJudge,
	"system":           RoleSystem,
	"skeptic":          RoleSkeptic,
	"architect":        RoleArchitect,
	"pragmatist":       RolePragmatist,
	"security":         RoleSecurity,
	"devil's_advocate": RoleDevilsAdvocate,
	"researcher":       RoleResearcher,
	"performance":      RolePerformance,
}

func handleSubAgentCreate(client *api.Client, personality string) (Subagent, error) {

	sessionID, err := client.CreateSession(personality)

	if err != nil {
		return Subagent{}, err
	}

	role, ok := personalityRoleMap[personality]

	if !ok {
		role = RoleJudge
	}

	subAgent := Subagent{
		SessionID: sessionID,
		Role:      role,
	}

	return subAgent, nil
}

func handleSubagentTask(c *api.Client, s string) error {
	_, err := c.SendMessageRaw(s, "hii")

	if err != nil {
		return err
	}

	return nil
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
		cr := &api.ControlRequest{ID: qp.ID, Type: "question.asked",
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

	case "session.error":
		var props struct {
			SessionID string `json:"sessionID"`
			Error     struct {
				Name string `json:"name"`
				Data struct {
					Message    string `json:"message"`
					StatusCode int    `json:"statusCode"`
				} `json:"data"`
			} `json:"error"`
		}
		if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
			return
		}
		if props.Error.Data.StatusCode >= 400 && program != nil {
			program.Send(ChatStreamMsg{
				SessionID: props.SessionID,
				Done:      true,
				Err:       fmt.Errorf("%s", props.Error.Data.Message),
			})
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
				Error   *struct {
					Name string `json:"name"`
					Data struct {
						Message    string `json:"message"`
						StatusCode int    `json:"statusCode"`
					} `json:"data"`
				} `json:"error"`
			} `json:"info"`
		}
		if err := json.Unmarshal(msg.Payload.Properties, &props); err != nil {
			return
		}

		if props.Info.Role == "assistant" && program != nil {
			if props.Info.Finish != "" {
				program.Send(ChatStreamMsg{
					SessionID: props.SessionID,
					Done:      true,
					ModelName: props.Info.ModelID,
				})
			}
		}
	}
}
