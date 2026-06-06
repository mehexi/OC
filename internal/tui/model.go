package tui

import (
	"oc/internal/api"
	"oc/internal/history"
	"oc/internal/tui/commands"

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
	modePerm
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
	Role      string
	Content   string
	Reasoning string
}

type Model struct {
	viewPort           viewport.Model
	inputText          textinput.Model
	messages           []ChatMessage
	sessionId          string
	loading            bool
	streaming          bool
	pendingPermission  *api.PermissionReqInfo
	permissionMsgIndex int
	pendingControl     *api.ControlRequest
	currentQuestionIdx int
	questionAnswers    []string
	awaitingResponse   bool
	width              int
	multiAgent         bool

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
	mode          VimMode
	visualAnchor  int
	visualCursor  int
	awaitingGG    bool
	qusItems      []qusItem
	qusCursor     int
	qusHeight     int
	termHeight    int
	sessions      []history.SessionSummary
	sessionPage   int
	sessionCursor int
	cmdCursor     int
	cmdPage       int
}

type (
	ServerStartedMsg      = commands.ServerStartedMsg
	ServerErrMsg          = commands.ServerErrMsg
	HealthCheckMsg        = commands.HealthCheckMsg
	ChatResponseMsg       = commands.ChatResponseMsg
	ChatStreamMsg         = commands.ChatStreamMsg
	ControlRequestMsg     = commands.ControlRequestMsg
	PermissionRequestMsg  = commands.PermissionRequestMsg
	LoadSessionMsg        = commands.LoadSessionMsg
	ProvidersInfoMsg      = commands.ProvidersInfoMsg
	PathMsg               = commands.PathMsg
	SessionUsageMsg       = commands.SessionUsageMsg
	ShowSessionListMsg    = commands.ShowSessionListMsg
)

func IntialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything ..."
	ti.SetWidth(50)
	ti.Focus()
	s := ti.Styles()

	ti.SetStyles(s)

	vp := viewport.New(viewport.WithWidth(80), viewport.WithHeight(24))

	return Model{
		viewPort:           vp,
		inputText:          ti,
		messages:           []ChatMessage{},
		sessionId:          "",
		loading:            false,
		width:              80,
		termHeight:         24,
		mode:               modeInsert,
		permissionMsgIndex: -1,
		multiAgent:         false,
	}
}
