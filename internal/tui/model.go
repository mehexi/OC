package tui

import (
	"oc/internal/api"
	"oc/internal/history"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

var program *tea.Program

func SetProgram(p *tea.Program) {
	program = p
}

type VimMode int

const (
	modeInsert VimMode = iota
	modeNormal
	modeVisual
	modeQus
	modeSession
	modeCmd
)

type cmdItem struct {
	Name        string
	Category    string
	Description string
}

type qusItem struct {
	label, desc string
}

type ChatMessage struct {
	Role    string
	Content string
}

type Model struct {
	viewPort           viewport.Model
	inputText          textinput.Model
	messages           []ChatMessage
	sessionId          string
	loading            bool
	streaming          bool
	pendingControl     *api.ControlRequest
	currentQuestionIdx int
	questionAnswers    []string
	awaitingResponse   bool
	width              int

	// TIPS:: serevr and stuff
	serverAddr    string
	serverErr     error
	client        *api.Client
	healthChecked bool
	healthStatus  *api.HealthResponse
	healthErr     error
	modelName     string
	tokensUsed    int
	contextLimit  int
	currentPath   string

	// TIP: modes and stuff
	mode         VimMode
	visualAnchor int
	visualCursor int
	awaitingGG   bool
	qusItems     []qusItem
	qusCursor    int
	qusHeight    int
	termHeight   int
	sessions      []history.SessionSummary
	sessionPage   int
	sessionCursor int
	cmdCursor     int
	cmdPage       int
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

type ChatStreamMsg struct {
	Text      string
	SessionID string
	FullText  string
	Done      bool
	ModelName string
	Err       error
}

type ControlRequestMsg struct {
	Request *api.ControlRequest
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

func IntialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything ..."
	ti.SetWidth(50)
	ti.Focus()
	s := ti.Styles()

	ti.SetStyles(s)

	vp := viewport.New()

	return Model{
		viewPort:  vp,
		inputText: ti,
		messages:  []ChatMessage{},
		sessionId: "",
		loading:   false,
		width:     0,
		mode:      modeInsert,
	}
}
