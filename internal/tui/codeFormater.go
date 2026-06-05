package tui

import (
	"encoding/json"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
)

var glamourStyle = func() []byte {
	s := map[string]any{
		"code_block": map[string]any{
			"color":  "244",
			"margin": 2,
			"chroma": map[string]any{
				"background": map[string]string{
					"background_color": "#082c4e",
				},
			},
		},
	}
	b, _ := json.Marshal(s)
	return b
}()

func RenderMarkdown(content string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(out)
}

func RenderCodeBlock(code, lang string) string {
	block := "```" + lang + "\n" + code + "\n```"
	return RenderMarkdown(block, 80)
}

func FormatCodeBlock(code, lang string) string {
	codeStyle := lipgloss.NewStyle().
		Background(chatBgColor).
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(cyanColor).
		Width(80).
		MaxWidth(80)

	header := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(chatBgColor).
		Padding(0, 2).
		Render(" " + lang + " ")

	body := RenderCodeBlock(code, lang)

	return lipgloss.JoinVertical(lipgloss.Top,
		header,
		codeStyle.Render(body),
	)
}
