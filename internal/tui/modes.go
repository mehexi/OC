package tui

import (
	"fmt"
	"oc/internal/server"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

func (m Model) onNormalKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		m.mode = modeInsert
		m.inputText.Focus()
		s := m.inputText.Styles()
		s.Cursor.Color = cyanColor
		s.Cursor.Blink = false
		m.inputText.SetStyles(s)
		return m, nil

	case "j":
		m.viewPort.ScrollDown(3)
		return m, nil

	case "k":
		m.viewPort.ScrollUp(3)
		return m, nil

	case "g":
		if m.awaitingGG {
			m.awaitingGG = false
			m.viewPort.GotoTop()
		} else {
			m.awaitingGG = true
		}
		return m, nil

	case "G":
		m.awaitingGG = false
		m.viewPort.GotoBottom()
		return m, nil

	case "V":
		m.awaitingGG = false
		if len(m.messages) == 0 {
			return m, nil
		}
		m.mode = modeVisual
		last := len(m.messages) - 1
		m.visualAnchor = last
		m.visualCursor = last
		return m, nil

	case "/":
		m.awaitingGG = false
		m.inputText.SetValue("/")
		m.inputText.SetCursor(len("/"))
		m.mode = modeInsert
		m.inputText.Focus()
		return m, nil

	case "enter":
		m.awaitingGG = false
		input := m.inputText.Value()
		if input == "" {
			return m, nil
		}
		if m.awaitingResponse {
			return m.handleQuestionAnswer(input)
		}
		if !m.loading && !m.streaming {
			if strings.HasPrefix(input, "/") {
				m.inputText.SetValue("")
				parts := strings.Fields(input)
				if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
					m.sessionId = ""
					m.messages = nil
					m = m.refreshMessages()
					return m, m.addAssistantMsg("Started a new session.")
				}
				return m, m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			m = m.refreshMessages()
			m.inputText.SetValue("")
			m.loading = true
			return m, m.sendChat(input)
		}
		return m, nil

	case "ctrl+c":
		if m.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.inputText.SetValue("")
		return m, nil

	default:
		m.awaitingGG = false
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		return m, vpCmd
	}
}

func (m Model) onInsertKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.inputText.Blur()
		return m, nil

	case "enter":
		input := m.inputText.Value()
		if input == "" {
			return m, nil
		}
		if m.awaitingResponse {
			return m.handleQuestionAnswer(input)
		}
		if !m.loading && !m.streaming {
			if strings.HasPrefix(input, "/") {
				m.inputText.SetValue("")
				parts := strings.Fields(input)
				if len(parts) == 2 && parts[0] == "/session" && parts[1] == "new" {
					m.sessionId = ""
					m.messages = nil
					m = m.refreshMessages()
					return m, m.addAssistantMsg("Started a new session.")
				}
				return m, m.handleCommand(input)
			}
			m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
			m = m.refreshMessages()
			m.inputText.SetValue("")
			m.loading = true
			return m, m.sendChat(input)
		}
		return m, nil

	case "ctrl+c":
		if m.inputText.Value() == "" {
			server.KillServer()
			return m, tea.Quit
		}
		m.inputText.SetValue("")
		return m, nil

	default:
		var cmd tea.Cmd
		m.inputText, cmd = m.inputText.Update(msg)
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		return m, tea.Batch(cmd, vpCmd)
	}
}

func (m Model) handleQuestionAnswer(input string) (Model, tea.Cmd) {
	m.questionAnswers = append(m.questionAnswers, input)
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: input})
	m = m.refreshMessages()
	m.inputText.SetValue("")

	if m.currentQuestionIdx+1 < len(m.pendingControl.Data.Questions) {
		m.currentQuestionIdx++
		content := formatQuestion(m.pendingControl.Data.Questions[m.currentQuestionIdx])
		m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: content})
		m.mode = modeInsert
		m.inputText.Focus()
		m.loading = false
		return m.refreshMessages(), nil
	}

	m.awaitingResponse = false
	m.loading = true
	return m, m.sendControlResponse()
}

func (m Model) onVisualKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j":
		if m.visualCursor < len(m.messages)-1 {
			m.visualCursor++
			m.viewPort.ScrollDown(3)
		}
		return m, nil

	case "k":
		if m.visualCursor > 0 {
			m.visualCursor--
			m.viewPort.ScrollUp(3)
		}
		return m, nil

	case "y":
		lo, hi := m.visualAnchor, m.visualCursor
		if lo > hi {
			lo, hi = hi, lo
		}
		var b strings.Builder
		for i := lo; i <= hi && i < len(m.messages); i++ {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(m.messages[i].Content)
		}
		text := b.String()
		if text != "" {
			clipboard.WriteAll(text)
		}
		m.mode = modeNormal
		m.inputText.SetValue(fmt.Sprintf("Yanked %d message(s)", hi-lo+1))
		m.inputText.SetCursor(len(m.inputText.Value()))
		return m, nil

	case "esc":
		m.mode = modeNormal
		return m, nil

	default:
		var vpCmd tea.Cmd
		m.viewPort, vpCmd = m.viewPort.Update(msg)
		return m, vpCmd
	}
}
