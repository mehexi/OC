package tui

import (
	"oc/internal/api"
	"oc/internal/history"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
)

type VimMode int

const (
	modeNormal VimMode = iota
	modeInsert
	modeVisual
)

type ChatMessage struct {
	Role    string
	Content string
}

type Model struct {
	viewPort   viewport.Model
	inputText  textinput.Model
	messages   []ChatMessage
	sessionId  string
	loading    bool
	width      int
	serverAddr string
	serverErr  error

	client        *api.Client
	healthChecked bool
	healthStatus  *api.HealthResponse
	healthErr     error

	modelName     string
	tokensUsed    int
	contextLimit  int
	currentPath   string

	mode         VimMode
	visualAnchor int
	visualCursor int
	awaitingGG   bool
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
	ModelName string
	Err       error
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

func IntialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything ..."
	ti.SetWidth(50)
	ti.Focus()

	vp := viewport.New()

	return Model{
		viewPort:  vp,
		inputText: ti,
		messages:  []ChatMessage{},
		sessionId: "",
		loading:   false,
		width:     0,
		mode:      modeNormal,
	}
}
