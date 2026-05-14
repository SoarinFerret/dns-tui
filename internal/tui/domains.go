package tui

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/soarinferret/dns-tui/internal/provider"
)

// Domains manages the domains sidebar pane.
type Domains struct {
	list    *tview.List
	app     *App
	domains []provider.Domain
}

// NewDomains creates the domains pane.
func NewDomains(a *App) *Domains {
	d := &Domains{
		list: tview.NewList().
			SetHighlightFullLine(true).
			ShowSecondaryText(false),
		app: a,
	}
	d.list.SetBorder(true).SetTitle(" Domains ")
	return d
}

// Clear removes all items from the domain list.
func (d *Domains) Clear() {
	d.list.Clear()
	d.domains = nil
}

// SetDomains populates the domain list.
func (d *Domains) SetDomains(domains []provider.Domain) {
	d.list.Clear()
	d.domains = domains
	for _, domain := range domains {
		dm := domain
		d.list.AddItem(
			fmt.Sprintf(" %s", domain.Name),
			"",
			0,
			func() { d.app.onDomainSelected(dm) },
		)
	}
}
