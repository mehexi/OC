package tui

import (
	"oc/internal/server"

	tea "charm.land/bubbletea/v2"
)

func initServer() tea.Cmd {
	return func() tea.Msg {
		addr, err := server.EnsureRunning()
		if err != nil {
			return ServerErrMsg{Err: err}
		}
		return ServerStartedMsg{Address: addr}
	}
}

func (m Model) Init() tea.Cmd {
	return initServer()
}
