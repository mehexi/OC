package commands

import (
	"oc/internal/api"

	tea "charm.land/bubbletea/v2"
)

func CheckHealth(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		status, err := client.Health()
		return HealthCheckMsg{Status: status, Err: err}
	}
}

func FetchPath(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		p, err := client.GetPath()
		if err != nil {
			return PathMsg{Err: err}
		}
		return PathMsg{Path: p}
	}
}

func FetchProviders(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetProviders()
		if err != nil {
			return ProvidersInfoMsg{Err: err}
		}
		modelName := ""
		if len(resp.Default) > 0 {
			for _, v := range resp.Default {
				modelName = v
				break
			}
		}
		return ProvidersInfoMsg{ModelName: modelName}
	}
}
