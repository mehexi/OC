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
