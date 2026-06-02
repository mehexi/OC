package tui

import (
	"oc/internal/api"
	"oc/internal/history"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
)

type ChatMessage struct {
	Role    string
	Content string
}

type Model struct {
	viewPort  viewport.Model
	inputText textinput.Model
	messages  []ChatMessage
	sessionId string
	loading   bool
	width     int
	height    int
	serverAddr string
	serverErr  error

	client        *api.Client
	healthChecked bool
	healthStatus  *api.HealthResponse
	healthErr     error
}

type ServerStartedMsg struct {
	Address string
}

type ServerErrMsg struct{ err error }

type HealthCheckMsg struct {
	Status *api.HealthResponse
	Err    error
}

type ChatResponseMsg struct {
	Response  string
	SessionID string
	Err       error
}

type LoadSessionMsg struct {
	Session *history.Session
}
