package tui

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var spinnerIdx int

func nextSpinner() string {
	spinnerIdx = (spinnerIdx + 3) % len(spinnerFrames)
	return spinnerFrames[spinnerIdx]
}
