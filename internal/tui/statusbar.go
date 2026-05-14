package tui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// StatusBar displays messages and errors at the bottom of the screen.
type StatusBar struct {
	view     *tview.TextView
	clearTag int
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetText(" Ready")
	tv.SetBackgroundColor(tcell.ColorDarkSlateGray)
	return &StatusBar{view: tv}
}

// SetMessage displays an informational message.
func (s *StatusBar) SetMessage(app *tview.Application, msg string) {
	s.clearTag++
	s.view.SetText(" " + msg)
	s.view.SetTextColor(tcell.ColorWhite)
}

// SetError displays an error message that auto-clears after 5 seconds.
func (s *StatusBar) SetError(app *tview.Application, msg string) {
	s.clearTag++
	tag := s.clearTag
	s.view.SetText(" Error: " + msg)
	s.view.SetTextColor(tcell.ColorRed)

	go func() {
		time.Sleep(5 * time.Second)
		app.QueueUpdateDraw(func() {
			if s.clearTag == tag {
				s.SetMessage(app, "Ready")
			}
		})
	}()
}
