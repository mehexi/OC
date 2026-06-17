package commands

import (
	"oc/internal/api"
	"oc/internal/history"
)

type ServerStartedMsg struct {
	Address string
}

type ServerErrMsg struct{ Err error }

type HealthCheckMsg struct {
	Status *api.HealthResponse
	Err    error
}

type ChatResponseMsg struct {
	Response  string
	SessionID string
	ModelName string
	Err       error
}

type ChatStreamMsg struct {
	Text          string
	Reasoning     string
	SessionID     string
	FullText      string
	FullReasoning string
	Done          bool
	ModelName     string
	Err           error
}

type ControlRequestMsg struct {
	Request *api.ControlRequest
	Err     error
}

type PermissionRequestMsg struct {
	Request *api.PermissionReqInfo
	Reply   string
	Err     error
}

type LoadSessionMsg struct {
	Session *history.Session
}

type ProvidersInfoMsg struct {
	ModelName string
	Models    []api.ModelList
	Err       error
}

type PathMsg struct {
	Path string
	Err  error
}

type SessionUsageMsg struct {
	ModelName    string
	TokensUsed   int
	ContextLimit int
	Err          error
}

type ShowSessionListMsg struct{}

type SubagentRole string

const (
	RoleJudge          SubagentRole = "judge"
	RoleSystem         SubagentRole = "system"
	RoleSkeptic        SubagentRole = "skeptic"
	RoleArchitect      SubagentRole = "architect"
	RolePragmatist     SubagentRole = "pragmatist"
	RoleSecurity       SubagentRole = "security"
	RoleDevilsAdvocate SubagentRole = "devil's_advocate"
	RoleResearcher     SubagentRole = "researcher"
	RolePerformance    SubagentRole = "performance"
)

type MultiAgentPlanMsg struct {
	SessionID     string
	Role          SubagentRole
	Content       string
	Done          bool
	Task          string
	MultiAgent    bool
	Agents        int
	Personalities []string
	Complexity    string
	Reason        string
}

type verdict struct {
	MultiAgent    bool     `json:"multi_agent"`
	Agents        int      `json:"agents"`
	Personalities []string `json:"personalities"`
	Complexity    string   `json:"complexity"`
	Reason        string   `json:"reason"`
}

type Subagent struct {
	SessionID string
	Role      SubagentRole
}

type SubagentList []Subagent

var Subagents SubagentList

func findSubagent(sessionID string) (Subagent, bool) {
	for _, a := range Subagents {
		if a.SessionID == sessionID {
			return a, true
		}
	}
	return Subagent{}, false
}
