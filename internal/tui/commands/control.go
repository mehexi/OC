package commands

import (
	"oc/internal/api"

	tea "charm.land/bubbletea/v2"
)

func SendControlResponse(client *api.Client, pendingControl *api.ControlRequest, questionAnswers []string) tea.Cmd {
	return func() tea.Msg {
		var answers [][]string
		for i := range pendingControl.Data.Questions {
			a := ""
			if i < len(questionAnswers) {
				a = questionAnswers[i]
			}
			answers = append(answers, []string{a})
		}
		err := client.ReplyToQuestion(pendingControl.ID, answers)
		if err != nil {
			return ControlRequestMsg{Err: err}
		}
		return ControlRequestMsg{}
	}
}
