package ui

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
)

type Model struct {
	isSplashScreen bool
	viewPort       viewport.Model
	inputText      textinput.Model
	message        []string
	sessionId      string
	loading        bool
	width          int
	height         int
}

type ServerStartedMsg struct{}

type ServerErrMsg struct{ err error }
