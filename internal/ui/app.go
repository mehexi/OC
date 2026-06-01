package ui

import (
	"os/exec"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func IntialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything ..."
	ti.SetWidth(50)
	ti.Focus()

	return Model{
		isSplashScreen: true,
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
		cmd := exec.Command("opencode", "server")
		err := cmd.Start()
		if err != nil {
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
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			m.isSplashScreen = false
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.inputText, cmd = m.inputText.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	if m.isSplashScreen {
		return splashView(m)
	}
	return ChatView(m)
}
