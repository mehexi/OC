package commands

import (
	"encoding/json"
	"oc/internal/api"
	"os"

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

		modelName := ""
		if raw, err := os.ReadFile("config.json"); err == nil {
			var cfg struct {
				Model string `json:"model"`
			}
			if json.Unmarshal(raw, &cfg) == nil && cfg.Model != "" {
				for _, m := range models {
					if m.ID == cfg.Model {
						modelName = cfg.Model
						break
					}
				}
			}
		}
		if modelName == "" {
			for _, v := range resp.Default {
				for _, m := range models {
					if m.ID == v {
						modelName = v
						break
					}
				}
				if modelName != "" {
					break
				}
			}
			if modelName == "" && len(resp.Default) > 0 {
				for _, v := range resp.Default {
					modelName = v
					break
				}
			}
		}

		return ProvidersInfoMsg{ModelName: modelName, Models: models}
	}
}
