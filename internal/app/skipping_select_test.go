package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestSkippingSelect(t *testing.T) {
	options := []SelectionOption{
		{Value: "opt1", Title: "Opt 1", Disabled: false},
		{Value: "opt2", Title: "Opt 2", Disabled: true},
		{Value: "opt3", Title: "Opt 3", Disabled: false},
	}

	var huhOptions []huh.Option[string]
	for _, opt := range options {
		huhOptions = append(huhOptions, huh.NewOption(opt.Title, opt.Value))
	}

	var selected string
	selectField := huh.NewSelect[string]().
		Options(huhOptions...).
		Value(&selected)

	disabledValues := map[string]bool{
		"opt2": true,
	}

	wrapped := &skippingSelect{
		Select:   selectField,
		disabled: disabledValues,
		options:  options,
	}

	// Focus the select component
	wrapped.Focus()

	// Initial hovered value should be "opt1"
	val, ok := wrapped.Hovered()
	if !ok || val != "opt1" {
		t.Fatalf("expected initial hovered to be 'opt1', got '%s'", val)
	}

	// Trigger Down key.
	// Since huh.Select uses keymap for key matching:
	// keymap is set via WithKeyMap, or uses default keymap.
	// Let's check how huh.Select's default keymap handles tea.KeyDown.
	// In huh, keymap default is DefaultKeyMap().
	// To be safe, let's explicitly set the keymap:
	keymap := huh.NewDefaultKeyMap()
	wrapped.WithKeyMap(keymap)

	// Send Down key event
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	m, _ := wrapped.Update(downKey)
	wrapped = m.(*skippingSelect)

	// It should skip "opt2" (disabled) and land on "opt3"!
	val, ok = wrapped.Hovered()
	if !ok || val != "opt3" {
		t.Fatalf("expected hovered after Down to skip 'opt2' and be 'opt3', got '%s'", val)
	}

	// Trigger Down key again. It should wrap around to "opt1"!
	m, _ = wrapped.Update(downKey)
	wrapped = m.(*skippingSelect)

	val, ok = wrapped.Hovered()
	if !ok || val != "opt1" {
		t.Fatalf("expected hovered after another Down to wrap around to 'opt1', got '%s'", val)
	}

	// Send Up key event
	upKey := tea.KeyMsg{Type: tea.KeyUp}
	m, _ = wrapped.Update(upKey)
	wrapped = m.(*skippingSelect)

	// It should skip "opt2" (disabled) and land on "opt3"!
	val, ok = wrapped.Hovered()
	if !ok || val != "opt3" {
		t.Fatalf("expected hovered after Up to skip 'opt2' and be 'opt3', got '%s'", val)
	}
}

func TestSkippingSelectAllDisabled(t *testing.T) {
	// If all options are disabled, it should not infinite loop
	options := []SelectionOption{
		{Value: "opt1", Title: "Opt 1", Disabled: true},
		{Value: "opt2", Title: "Opt 2", Disabled: true},
	}

	var huhOptions []huh.Option[string]
	for _, opt := range options {
		huhOptions = append(huhOptions, huh.NewOption(opt.Title, opt.Value))
	}

	var selected string
	selectField := huh.NewSelect[string]().
		Options(huhOptions...).
		Value(&selected)

	disabledValues := map[string]bool{
		"opt1": true,
		"opt2": true,
	}

	wrapped := &skippingSelect{
		Select:   selectField,
		disabled: disabledValues,
		options:  options,
	}

	keymap := huh.NewDefaultKeyMap()
	wrapped.WithKeyMap(keymap)

	wrapped.Focus()

	// Sending Down key should not crash/loop infinitely
	downKey := tea.KeyMsg{Type: tea.KeyDown}
	m, _ := wrapped.Update(downKey)
	wrapped = m.(*skippingSelect)

	// Should still be on "opt1" or "opt2", but not infinite loop
	_, _ = wrapped.Hovered()
}

func TestSkippingSelectFilterNoFreeze(t *testing.T) {
	options := []SelectionOption{
		{Value: "opt1", Title: "Opt 1", Disabled: true},
	}

	var huhOptions []huh.Option[string]
	for _, opt := range options {
		huhOptions = append(huhOptions, huh.NewOption(opt.Title, opt.Value))
	}

	var selected string
	selectField := huh.NewSelect[string]().
		Options(huhOptions...).
		Value(&selected)

	disabledValues := map[string]bool{
		"opt1": true,
	}

	wrapped := &skippingSelect{
		Select:   selectField,
		disabled: disabledValues,
		options:  options,
	}

	// Focus the select component
	wrapped.Focus()

	// Send a character rune key (simulate typing to filter)
	charKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	// This must not freeze or panic
	m, _ := wrapped.Update(charKey)
	_ = m.(*skippingSelect)
}
