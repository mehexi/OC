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
		if v, ok := resp.Default["opencode"]; ok {
			modelName = v
		} else if len(resp.Default) > 0 {
			for _, v := range resp.Default {
				modelName = v
				break
			}
		}

		var models []api.ModelList
		for _, provider := range resp.Providers {
			providerID, _ := provider["id"].(string)
			modelsRaw, _ := provider["models"].(map[string]any)
			for id, raw := range modelsRaw {
				m, _ := raw.(map[string]any)
				name, _ := m["name"].(string)
				ml := api.ModelList{
					ID:         id,
					ProviderID: providerID,
					Name:       name,
				}
				if costObj, ok := m["cost"].(map[string]any); ok {
					ml.CostInput, _ = costObj["input"].(float64)
					ml.CostOutput, _ = costObj["output"].(float64)
				}
				models = append(models, ml)
			}
		}

		return ProvidersInfoMsg{ModelName: modelName, Models: models}
	}
}
