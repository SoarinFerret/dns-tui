package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ShortcutBar displays context-sensitive keyboard shortcuts.
type ShortcutBar struct {
	view *tview.TextView
}

// NewShortcutBar creates the shortcut bar.
func NewShortcutBar() *ShortcutBar {
	tv := tview.NewTextView().
		SetDynamicColors(true)
	tv.SetBackgroundColor(tcell.ColorDarkBlue)
	return &ShortcutBar{view: tv}
}

type shortcut struct {
	key  string
	desc string
}

var (
	globalShortcuts = []shortcut{
		{"Tab", "Switch pane"},
		{"?", "Help"},
		{"q", "Quit"},
	}

	profileShortcuts = []shortcut{
		{"Enter", "Select"},
		{"a", "Add"},
		{"d", "Delete"},
	}

	domainShortcuts = []shortcut{
		{"Enter", "Select"},
	}

	recordShortcuts = []shortcut{
		{"a", "Add"},
		{"e", "Edit"},
		{"d", "Delete"},
		{"/", "Search"},
		{"n/p", "Page"},
	}
)

// Update refreshes the shortcut bar for the given focused pane index.
func (s *ShortcutBar) Update(focusIndex int) {
	var contextual []shortcut
	switch focusIndex {
	case 0:
		contextual = profileShortcuts
	case 1:
		contextual = domainShortcuts
	case 2:
		contextual = recordShortcuts
	}

	var text string
	for _, sc := range contextual {
		text += " [black:aqua] " + sc.key + " [white:darkblue] " + sc.desc
	}
	for _, sc := range globalShortcuts {
		text += " [black:aqua] " + sc.key + " [white:darkblue] " + sc.desc
	}

	s.view.SetText(text)
}
