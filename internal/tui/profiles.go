package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Profiles manages the profiles sidebar pane.
type Profiles struct {
	list *tview.List
	app  *App
}

// NewProfiles creates the profiles pane populated from config.
func NewProfiles(a *App) *Profiles {
	p := &Profiles{
		list: tview.NewList().
			SetHighlightFullLine(true).
			ShowSecondaryText(false),
		app: a,
	}

	p.list.SetBorder(true).SetTitle(" Profiles ")
	p.list.SetInputCapture(p.handleInput)
	p.populate()

	return p
}

func (p *Profiles) populate() {
	for i, profile := range p.app.cfg.Profiles {
		idx := i
		p.list.AddItem(
			fmt.Sprintf(" %s", profile.Name),
			"",
			0,
			func() { p.app.onProfileSelected(idx) },
		)
	}
}

func (p *Profiles) reload() {
	p.list.Clear()
	p.populate()
}

func (p *Profiles) handleInput(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() != tcell.KeyRune {
		return event
	}
	switch event.Rune() {
	case 'a':
		showProfileForm(p.app)
		return nil
	case 'd':
		idx := p.list.GetCurrentItem()
		if idx >= 0 && idx < len(p.app.cfg.Profiles) {
			showDeleteProfileConfirm(p.app, idx)
		}
		return nil
	}
	return event
}
