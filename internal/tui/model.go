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
	modeModel
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
	Role      MessageRole
	Content   string
	Reasoning string
}

type MessageRole string

const (
	RoleUser           MessageRole = "user"
	RoleAssistant      MessageRole = "assistant"
	RoleJudge          MessageRole = "judge"
	RolePermission     MessageRole = "permission"
	RoleSystem         MessageRole = "system"
	RoleSkeptic        MessageRole = "skeptic"
	RoleArchitect      MessageRole = "architect"
	RolePragmatist     MessageRole = "pragmatist"
	RoleSecurity       MessageRole = "security"
	RoleDevilsAdvocate MessageRole = "devil's_advocate"
	RoleResearcher     MessageRole = "researcher"
	RolePerformance    MessageRole = "performance"
)

type LayoutState struct {
	viewPort   viewport.Model
	inputText  textinput.Model
	width      int
	termHeight int
}

type ServerState struct {
	serverAddr      string
	serverErr       error
	client          *api.Client
	healthChecked   bool
	healthStatus    *api.HealthResponse
	healthErr       error
	modelName       string
	modelID         string
	modelProviderID string
	tokensUsed      int
	contextLimit    int
	currentPath     string
}

type FlowState struct {
	pendingPermission  *api.PermissionReqInfo
	permissionMsgIndex int
	pendingControl     *api.ControlRequest
	currentQuestionIdx int
	questionAnswers    []string
	awaitingResponse   bool
}

type ChatState struct {
	messages      []ChatMessage
	sessionId     string
	loading       bool
	streaming     bool
	multiAgent    *bool
	agents        int
	complexity    string
	reason        string
	personalities []string
}

type ModesState struct {
	mode          VimMode
	visualAnchor  int
	visualCursor  int
	awaitingGG    bool
	qusItems      []qusItem
	qusCursor     int
	qusHeight     int
	sessions      []history.SessionSummary
	sessionPage   int
	sessionCursor int
	cmdCursor     int
	cmdPage       int
	models        []api.ModelList
	modelCursor   int
	modelPage     int
}

type Model struct {
	Layout LayoutState
	Server ServerState
	Flow   FlowState
	Chat   ChatState
	Modes  ModesState
}

type Subagents []struct {
	SessionID string
	Role      string
}

func (m Model) MultiAgent() bool { return m.Chat.multiAgent != nil && *m.Chat.multiAgent }

type (
	ServerStartedMsg     = commands.ServerStartedMsg
	ServerErrMsg         = commands.ServerErrMsg
	HealthCheckMsg       = commands.HealthCheckMsg
	ChatResponseMsg      = commands.ChatResponseMsg
	ChatStreamMsg        = commands.ChatStreamMsg
	ControlRequestMsg    = commands.ControlRequestMsg
	PermissionRequestMsg = commands.PermissionRequestMsg
	LoadSessionMsg       = commands.LoadSessionMsg
	ProvidersInfoMsg     = commands.ProvidersInfoMsg
	PathMsg              = commands.PathMsg
	SessionUsageMsg      = commands.SessionUsageMsg
	ShowSessionListMsg   = commands.ShowSessionListMsg
	MultiAgentPlanMsg    = commands.MultiAgentPlanMsg
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
		Layout: LayoutState{
			viewPort:   vp,
			inputText:  ti,
			width:      80,
			termHeight: 24,
		},
		Chat: ChatState{
			messages:   []ChatMessage{},
			sessionId:  "",
			loading:    false,
			multiAgent: new(bool),
		},
		Modes: ModesState{
			mode: modeInsert,
		},
		Flow: FlowState{
			permissionMsgIndex: -1,
		},
	}
}
