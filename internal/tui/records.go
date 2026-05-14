package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/soarinferret/dns-tui/internal/provider"
)

// Records manages the main records table pane.
type Records struct {
	table   *tview.Table
	app     *App
	records []provider.Record
	page    int
	pages   int
}

// NewRecords creates the records pane.
func NewRecords(a *App) *Records {
	r := &Records{
		table: tview.NewTable().
			SetSelectable(true, false).
			SetFixed(1, 0),
		app: a,
	}
	r.table.SetBorder(true).SetTitle(" Records ")
	r.table.SetInputCapture(r.handleInput)
	r.writeHeader()
	return r
}

func (r *Records) writeHeader() {
	headers := []string{"Type", "Name", "Value", "TTL", "Priority"}
	for i, h := range headers {
		r.table.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1))
	}
	// Give Value column more space
	r.table.GetCell(0, 2).SetExpansion(3)
}

// Clear removes all records from the table.
func (r *Records) Clear() {
	r.records = nil
	r.page = 0
	r.pages = 0
	r.table.Clear()
	r.writeHeader()
	r.table.SetTitle(" Records ")
}

// SetRecords sets the records and displays the first page.
func (r *Records) SetRecords(records []provider.Record) {
	r.records = records
	r.page = 0
	r.pages = (len(records) + pageSize - 1) / pageSize
	if r.pages == 0 {
		r.pages = 1
	}
	r.render()
}

func (r *Records) render() {
	r.table.Clear()
	r.writeHeader()

	start := r.page * pageSize
	end := start + pageSize
	if end > len(r.records) {
		end = len(r.records)
	}

	for i, rec := range r.records[start:end] {
		row := i + 1
		r.table.SetCell(row, 0, tview.NewTableCell(rec.Type).SetExpansion(1))
		r.table.SetCell(row, 1, tview.NewTableCell(rec.Name).SetExpansion(1))
		r.table.SetCell(row, 2, tview.NewTableCell(rec.Value).SetExpansion(3))
		r.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%d", rec.TTL)).SetExpansion(1))
		pri := ""
		if rec.Type == "MX" || rec.Type == "SRV" {
			pri = fmt.Sprintf("%d", rec.Priority)
		}
		r.table.SetCell(row, 4, tview.NewTableCell(pri).SetExpansion(1))
	}

	r.table.SetTitle(fmt.Sprintf(" Records [%d/%d] ", r.page+1, r.pages))
	r.table.ScrollToBeginning()
	r.table.Select(1, 0)
}

func (r *Records) selectedRecord() (provider.Record, bool) {
	row, _ := r.table.GetSelection()
	idx := r.page*pageSize + row - 1
	if idx < 0 || idx >= len(r.records) {
		return provider.Record{}, false
	}
	return r.records[idx], true
}

func (r *Records) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'a':
			if r.app.currentProvider != nil && r.app.currentDomain.ID != "" {
				showRecordForm(r.app, provider.Record{TTL: 300}, false)
			}
			return nil
		case 'e':
			if rec, ok := r.selectedRecord(); ok {
				showRecordForm(r.app, rec, true)
			}
			return nil
		case 'd':
			if rec, ok := r.selectedRecord(); ok {
				showDeleteConfirm(r.app, rec)
			}
			return nil
		case '/':
			r.showSearch()
			return nil
		case 'n':
			if r.page < r.pages-1 {
				r.page++
				r.render()
			}
			return nil
		case 'p':
			if r.page > 0 {
				r.page--
				r.render()
			}
			return nil
		}
	case tcell.KeyEscape:
		r.app.filterRecords("")
		return nil
	}
	return event
}

func (r *Records) showSearch() {
	input := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(30)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			r.app.filterRecords(input.GetText())
		}
		r.app.app.SetRoot(r.app.layout, true)
		r.app.app.SetFocus(r.table)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(r.app.layout, 0, 1, false).
		AddItem(input, 1, 0, true)

	r.app.app.SetRoot(flex, true)
	r.app.app.SetFocus(input)
}
