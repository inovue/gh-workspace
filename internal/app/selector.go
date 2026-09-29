package app

import (
	"errors"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// Selector draws the interactive prompts. The default implementation renders
// huh forms on stderr so stdout stays reserved for results.
type Selector interface {
	Select(title string, options []SelectionOption) (string, error)
	Input(title string, value *string, validate func(string) error) error
	Confirm(title string, defaultVal bool) (bool, error)
}

type SelectionOption struct {
	Value       string
	Title       string
	Description string
	Disabled    bool
}

type huhSelector struct {
	output io.Writer
	input  io.Reader
}

func (s huhSelector) run(form *huh.Form) error {
	form = form.WithOutput(s.output)
	if s.input != nil {
		form = form.WithInput(s.input)
	}
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return errCancelled
		}
		return err
	}
	return nil
}

func (s huhSelector) Confirm(title string, defaultVal bool) (bool, error) {
	confirmed := defaultVal
	err := s.run(huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(title).Value(&confirmed))))
	return confirmed, err
}

func (s huhSelector) Input(title string, value *string, validate func(string) error) error {
	field := huh.NewInput().Title(title).Value(value)
	if validate != nil {
		field = field.Validate(validate)
	}
	return s.run(huh.NewForm(huh.NewGroup(field)))
}

func (s huhSelector) Select(title string, options []SelectionOption) (string, error) {
	huhOptions := make([]huh.Option[string], 0, len(options))
	disabled := make(map[string]bool)
	var selected string
	for _, option := range options {
		label := option.Title
		if option.Description != "" {
			label += "  " + option.Description
		}
		huhOptions = append(huhOptions, huh.NewOption(label, option.Value))
		if option.Disabled {
			disabled[option.Value] = true
		} else if selected == "" {
			selected = option.Value
		}
	}

	field := huh.NewSelect[string]().
		Title(title).
		Options(huhOptions...).
		Filtering(true).
		Value(&selected).
		Validate(func(value string) error {
			if disabled[value] {
				return fmt.Errorf("already cloned")
			}
			return nil
		})
	if len(options) > 12 {
		field.Height(14)
	}

	form := huh.NewForm(huh.NewGroup(&skippingSelect{Select: field, disabled: disabled, options: options})).
		WithProgramOptions(
			tea.WithMouseCellMotion(),
			tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
				if mouse, ok := msg.(tea.MouseMsg); ok {
					switch mouse.Button {
					case tea.MouseButtonWheelUp:
						return tea.KeyMsg{Type: tea.KeyUp}
					case tea.MouseButtonWheelDown:
						return tea.KeyMsg{Type: tea.KeyDown}
					}
				}
				return msg
			}),
		)
	err := s.run(form)
	return selected, err
}

// skippingSelect moves the cursor past disabled options, which huh does not
// support natively.
type skippingSelect struct {
	*huh.Select[string]
	disabled map[string]bool
	options  []SelectionOption
}

func (s *skippingSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := s.Select.Update(msg)
	s.Select = model.(*huh.Select[string])

	if !s.isNavigation(msg) || !s.hasEnabled() {
		return s, cmd
	}
	for i := 0; i < len(s.options); i++ {
		hovered, ok := s.Hovered()
		if !ok || !s.disabled[hovered] {
			break
		}
		model, _ = s.Select.Update(msg)
		s.Select = model.(*huh.Select[string])
	}
	return s, cmd
}

func (s *skippingSelect) isNavigation(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	switch key.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyCtrlN, tea.KeyCtrlP:
		return true
	case tea.KeyRunes:
		switch string(key.Runes) {
		case "j", "k", "h", "l":
			return !s.GetFiltering()
		}
	}
	return false
}

func (s *skippingSelect) hasEnabled() bool {
	for _, option := range s.options {
		if !s.disabled[option.Value] {
			return true
		}
	}
	return false
}

func (s *skippingSelect) WithTheme(theme *huh.Theme) huh.Field {
	s.Select = s.Select.WithTheme(theme).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithKeyMap(keymap *huh.KeyMap) huh.Field {
	s.Select = s.Select.WithKeyMap(keymap).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithWidth(width int) huh.Field {
	s.Select = s.Select.WithWidth(width).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithHeight(height int) huh.Field {
	s.Select = s.Select.WithHeight(height).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithPosition(position huh.FieldPosition) huh.Field {
	s.Select = s.Select.WithPosition(position).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithAccessible(accessible bool) huh.Field {
	s.Select = s.Select.WithAccessible(accessible).(*huh.Select[string])
	return s
}
