package tui

import (
	"fmt"
	"strconv"

	"github.com/rivo/tview"

	"github.com/soarinferret/dns-tui/internal/config"
	"github.com/soarinferret/dns-tui/internal/provider"
)

var (
	recordTypes   = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"}
	providerTypes = []string{"cloudflare", "godaddy", "dnsmadeeasy", "fortigate"}
	providerCreds = map[string][]string{
		"cloudflare":  {"api_token"},
		"godaddy":     {"api_key", "api_secret"},
		"dnsmadeeasy": {"api_key", "api_secret"},
		"fortigate":   {"host", "api_token"},
	}
	providerOptionalCreds = map[string][]string{
		"fortigate": {"vdom", "insecure_skip_verify"},
	}
)

func showRecordForm(a *App, rec provider.Record, editing bool) {
	form := tview.NewForm()

	title := "Add Record"
	if editing {
		title = "Edit Record"
	}

	typeIdx := 0
	for i, t := range recordTypes {
		if t == rec.Type {
			typeIdx = i
			break
		}
	}
	selectedType := rec.Type
	if selectedType == "" {
		selectedType = "A"
	}

	form.AddDropDown("Type", recordTypes, typeIdx, func(option string, index int) {
		selectedType = option
	})
	form.AddInputField("Name", rec.Name, 40, nil, nil)
	form.AddInputField("Value", rec.Value, 40, nil, nil)
	form.AddInputField("TTL", fmt.Sprintf("%d", rec.TTL), 10, nil, nil)
	form.AddInputField("Priority", fmt.Sprintf("%d", rec.Priority), 10, nil, nil)

	if editing {
		// Lock the type field when editing
		form.GetFormItemByLabel("Type").(*tview.DropDown).SetCurrentOption(typeIdx)
	}

	form.AddButton("Save", func() {
		nameField := form.GetFormItemByLabel("Name").(*tview.InputField).GetText()
		valueField := form.GetFormItemByLabel("Value").(*tview.InputField).GetText()
		ttlStr := form.GetFormItemByLabel("TTL").(*tview.InputField).GetText()
		priStr := form.GetFormItemByLabel("Priority").(*tview.InputField).GetText()

		ttl, err := strconv.Atoi(ttlStr)
		if err != nil {
			a.statusBar.SetError(a.app, "Invalid TTL value")
			return
		}
		pri, _ := strconv.Atoi(priStr)

		r := provider.Record{
			ID:       rec.ID,
			Type:     selectedType,
			Name:     nameField,
			Value:    valueField,
			TTL:      ttl,
			Priority: pri,
		}

		a.app.SetRoot(a.layout, true)
		a.app.SetFocus(a.panes[a.focusIndex])

		if editing {
			a.updateRecord(r)
		} else {
			a.createRecord(r)
		}
	})

	form.AddButton("Cancel", func() {
		a.app.SetRoot(a.layout, true)
		a.app.SetFocus(a.panes[a.focusIndex])
	})

	form.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", title))

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 18, 0, true).
			AddItem(nil, 0, 1, false),
			60, 0, true).
		AddItem(nil, 0, 1, false)

	a.app.SetRoot(modal, true)
	a.app.SetFocus(form)
}

func showDeleteConfirm(a *App, rec provider.Record) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete %s record %q (%s)?", rec.Type, rec.Name, rec.Value)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.SetRoot(a.layout, true)
			a.app.SetFocus(a.panes[a.focusIndex])
			if buttonLabel == "Delete" {
				a.deleteRecord(rec.ID)
			}
		})

	a.app.SetRoot(modal, true)
}

func showProfileForm(a *App) {
	form := tview.NewForm()

	selectedProvider := providerTypes[0]
	type credField struct {
		input    *tview.InputField
		label    string
		optional bool
	}
	credFields := map[string]credField{}

	buildCredFields := func(form *tview.Form, provName string) {
		for _, cf := range credFields {
			idx := form.GetFormItemIndex(cf.label)
			if idx >= 0 {
				form.RemoveFormItem(idx)
			}
		}
		credFields = map[string]credField{}

		for _, field := range providerCreds[provName] {
			label := field
			input := tview.NewInputField().
				SetLabel(label).
				SetFieldWidth(40)
			credFields[field] = credField{input: input, label: label}
			form.AddFormItem(input)
		}
		for _, field := range providerOptionalCreds[provName] {
			label := field + " (optional)"
			input := tview.NewInputField().
				SetLabel(label).
				SetFieldWidth(40)
			credFields[field] = credField{input: input, label: label, optional: true}
			form.AddFormItem(input)
		}
	}

	form.AddInputField("Name", "", 40, nil, nil)
	form.AddDropDown("Provider", providerTypes, 0, func(option string, index int) {
		if option != selectedProvider {
			selectedProvider = option
			buildCredFields(form, option)
		}
	})
	buildCredFields(form, selectedProvider)

	form.AddButton("Save", func() {
		name := form.GetFormItemByLabel("Name").(*tview.InputField).GetText()
		if name == "" {
			a.statusBar.SetError(a.app, "Profile name is required")
			return
		}

		creds := make(map[string]string)
		for field, cf := range credFields {
			val := cf.input.GetText()
			if val == "" {
				if cf.optional {
					continue
				}
				a.statusBar.SetError(a.app, fmt.Sprintf("%s is required", field))
				return
			}
			creds[field] = val
		}

		a.app.SetRoot(a.layout, true)
		a.app.SetFocus(a.panes[a.focusIndex])

		a.addProfile(config.Profile{
			Name:        name,
			Provider:    selectedProvider,
			Credentials: creds,
		})
	})

	form.AddButton("Cancel", func() {
		a.app.SetRoot(a.layout, true)
		a.app.SetFocus(a.panes[a.focusIndex])
	})

	form.SetBorder(true).SetTitle(" Add Profile ")

	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 18, 0, true).
			AddItem(nil, 0, 1, false),
			60, 0, true).
		AddItem(nil, 0, 1, false)

	a.app.SetRoot(modal, true)
	a.app.SetFocus(form)
}

func showDeleteProfileConfirm(a *App, index int) {
	name := a.cfg.Profiles[index].Name
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete profile %q?", name)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.app.SetRoot(a.layout, true)
			a.app.SetFocus(a.panes[a.focusIndex])
			if buttonLabel == "Delete" {
				a.deleteProfile(index)
			}
		})

	a.app.SetRoot(modal, true)
}
