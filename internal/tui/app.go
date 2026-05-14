package tui

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/soarinferret/dns-tui/internal/config"
	"github.com/soarinferret/dns-tui/internal/provider"
)

const pageSize = 50

// App is the main TUI application.
type App struct {
	app        *tview.Application
	cfg        *config.Config
	configPath string
	layout     *tview.Flex

	profiles    *Profiles
	domains     *Domains
	records     *Records
	shortcutBar *ShortcutBar
	statusBar   *StatusBar

	panes      []tview.Primitive
	focusIndex int

	currentProvider provider.Provider
	currentDomain   provider.Domain
	allRecords      []provider.Record
}

// New creates and returns a new App.
func New(cfg *config.Config, configPath string) *App {
	a := &App{
		app:        tview.NewApplication(),
		cfg:        cfg,
		configPath: configPath,
	}

	a.profiles = NewProfiles(a)
	a.domains = NewDomains(a)
	a.records = NewRecords(a)
	a.shortcutBar = NewShortcutBar()
	a.statusBar = NewStatusBar()

	a.panes = []tview.Primitive{
		a.profiles.list,
		a.domains.list,
		a.records.table,
	}

	content := tview.NewFlex().
		AddItem(a.profiles.list, 0, 1, true).
		AddItem(a.domains.list, 0, 1, false).
		AddItem(a.records.table, 0, 3, false)

	a.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(a.shortcutBar.view, 1, 0, false).
		AddItem(a.statusBar.view, 1, 0, false)

	a.app.SetRoot(a.layout, true)
	a.app.SetInputCapture(a.globalInput)
	a.updateFocusBorders()

	return a
}

// Run starts the TUI application.
func (a *App) Run() error {
	return a.app.Run()
}

func (a *App) globalInput(event *tcell.EventKey) *tcell.EventKey {
	// Don't capture input when a modal/form is showing
	if a.app.GetFocus() != a.panes[a.focusIndex] {
		return event
	}

	switch event.Key() {
	case tcell.KeyTab:
		a.focusIndex = (a.focusIndex + 1) % len(a.panes)
		a.app.SetFocus(a.panes[a.focusIndex])
		a.updateFocusBorders()
		return nil
	case tcell.KeyBacktab:
		a.focusIndex = (a.focusIndex - 1 + len(a.panes)) % len(a.panes)
		a.app.SetFocus(a.panes[a.focusIndex])
		a.updateFocusBorders()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q':
			a.app.Stop()
			return nil
		case '?':
			a.showHelp()
			return nil
		}
	}
	return event
}

func (a *App) updateFocusBorders() {
	for i, p := range a.panes {
		if box, ok := p.(interface{ SetBorderColor(tcell.Color) *tview.Box }); ok {
			if i == a.focusIndex {
				box.SetBorderColor(tcell.ColorGreen)
			} else {
				box.SetBorderColor(tcell.ColorWhite)
			}
		}
	}
	a.shortcutBar.Update(a.focusIndex)
}

func (a *App) onProfileSelected(index int) {
	if index < 0 || index >= len(a.cfg.Profiles) {
		return
	}
	profile := a.cfg.Profiles[index]

	p, err := provider.New(profile.Provider, profile.Credentials)
	if err != nil {
		a.statusBar.SetError(a.app, err.Error())
		return
	}
	a.currentProvider = p

	a.domains.Clear()
	a.records.Clear()
	a.statusBar.SetMessage(a.app, "Loading domains...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		domains, err := p.ListDomains(ctx)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.statusBar.SetError(a.app, err.Error())
				return
			}
			sort.Slice(domains, func(i, j int) bool {
				return domains[i].Name < domains[j].Name
			})
			a.domains.SetDomains(domains)
			a.statusBar.SetMessage(a.app, "Ready")
		})
	}()
}

func (a *App) onDomainSelected(domain provider.Domain) {
	a.currentDomain = domain
	a.records.Clear()
	a.statusBar.SetMessage(a.app, "Loading records...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		records, err := a.currentProvider.ListRecords(ctx, domain.ID)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.statusBar.SetError(a.app, err.Error())
				return
			}
			sort.Slice(records, func(i, j int) bool {
				if records[i].Name == records[j].Name {
					return records[i].Type < records[j].Type
				}
				return records[i].Name < records[j].Name
			})
			a.allRecords = records
			a.records.SetRecords(records)
			a.statusBar.SetMessage(a.app, "Ready")
		})
	}()
}

func (a *App) createRecord(r provider.Record) {
	a.statusBar.SetMessage(a.app, "Creating record...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := a.currentProvider.CreateRecord(ctx, a.currentDomain.ID, r)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.statusBar.SetError(a.app, err.Error())
				return
			}
			a.statusBar.SetMessage(a.app, "Record created")
			a.onDomainSelected(a.currentDomain)
		})
	}()
}

func (a *App) updateRecord(r provider.Record) {
	a.statusBar.SetMessage(a.app, "Updating record...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := a.currentProvider.UpdateRecord(ctx, a.currentDomain.ID, r)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.statusBar.SetError(a.app, err.Error())
				return
			}
			a.statusBar.SetMessage(a.app, "Record updated")
			a.onDomainSelected(a.currentDomain)
		})
	}()
}

func (a *App) deleteRecord(recordID string) {
	a.statusBar.SetMessage(a.app, "Deleting record...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := a.currentProvider.DeleteRecord(ctx, a.currentDomain.ID, recordID)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.statusBar.SetError(a.app, err.Error())
				return
			}
			a.statusBar.SetMessage(a.app, "Record deleted")
			a.onDomainSelected(a.currentDomain)
		})
	}()
}

func (a *App) showHelp() {
	help := `[yellow]Keybindings[-]

[green]Global[-]
  Tab/Shift-Tab  Switch panes
  q              Quit
  ?              This help

[green]Navigation[-]
  j/↓  Move down
  k/↑  Move up
  Enter Select

[green]Profiles[-]
  a    Add profile
  d    Delete profile

[green]Records[-]
  a    Add record
  e    Edit record
  d    Delete record
  /    Search
  Esc  Clear search
  n    Next page
  p    Previous page`

	modal := tview.NewModal().
		SetText(help).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.SetRoot(a.layout, true)
			a.app.SetFocus(a.panes[a.focusIndex])
		})

	a.app.SetRoot(modal, true)
}

func (a *App) addProfile(p config.Profile) {
	a.cfg.Profiles = append(a.cfg.Profiles, p)
	if err := config.Save(a.configPath, a.cfg); err != nil {
		a.statusBar.SetError(a.app, err.Error())
		return
	}
	a.profiles.reload()
	a.statusBar.SetMessage(a.app, "Profile added")
}

func (a *App) deleteProfile(index int) {
	if index < 0 || index >= len(a.cfg.Profiles) {
		return
	}
	a.cfg.Profiles = append(a.cfg.Profiles[:index], a.cfg.Profiles[index+1:]...)
	if err := config.Save(a.configPath, a.cfg); err != nil {
		a.statusBar.SetError(a.app, err.Error())
		return
	}
	a.currentProvider = nil
	a.currentDomain = provider.Domain{}
	a.domains.Clear()
	a.records.Clear()
	a.profiles.reload()
	a.statusBar.SetMessage(a.app, "Profile deleted")
}

func (a *App) filterRecords(query string) {
	if query == "" {
		a.records.SetRecords(a.allRecords)
		return
	}
	query = strings.ToLower(query)
	var filtered []provider.Record
	for _, r := range a.allRecords {
		if strings.Contains(strings.ToLower(r.Name), query) ||
			strings.Contains(strings.ToLower(r.Value), query) {
			filtered = append(filtered, r)
		}
	}
	a.records.SetRecords(filtered)
}
