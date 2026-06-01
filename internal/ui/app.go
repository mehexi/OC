package ui

import (
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func IntialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything ..."
	styles := ti.Styles()
	styles.Focused.Placeholder = lipgloss.NewStyle().Background(inputBgColor)
	styles.Focused.Text = lipgloss.NewStyle().Background(inputBgColor)
	ti.SetStyles(styles)
	ti.SetWidth(50)
	ti.Focus()

	vp := viewport.New()

	return Model{
		isSplashScreen: true,
		viewPort:       vp,
		inputText:      ti,
		message:        []string{},
		sessionId:      "",
		loading:        false,
		width:          0,
		height:         0,
	}
}

func initServer() tea.Cmd {
	return func() tea.Msg {
		check := exec.Command("pgrep", "-f", "opencode server")
		if err := check.Run(); err == nil {
			return ServerStartedMsg{}
		}

		cmd := exec.Command("opencode", "server")
		if err := cmd.Start(); err != nil {
			return ServerErrMsg{err}
		}

		return ServerStartedMsg{}
	}
}

func (m Model) Init() tea.Cmd {
	return initServer()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewPort.SetWidth(msg.Width)
		m.viewPort.SetHeight(msg.Height - 3)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.isSplashScreen == true || m.inputText.Value() == "" {
				return m, tea.Quit
			} else {
				m.inputText.SetValue("")
			}
		case "enter":
			input := m.inputText.Value()
			if input != "" {
				m.message = append(m.message, input)
				m.inputText.SetValue("")
				m.isSplashScreen = false
				m.viewPort.GotoBottom()
			}
		case "":

		}
	}

	var chatBubbles []string
	for _, text := range m.message {
		chatBubbles = append(chatBubbles, RenderChatBubble(text, m))
	}
	m.viewPort.SetContent(strings.Join(chatBubbles, "\n\n"))

	var cmd tea.Cmd
	m.inputText, cmd = m.inputText.Update(msg)

	var vpCmd tea.Cmd
	m.viewPort, vpCmd = m.viewPort.Update(msg)

	return m, tea.Batch(cmd, vpCmd)
}

func (m Model) View() tea.View {
	if m.isSplashScreen {
		return splashView(m)
	}
	return ChatView(m)
}
