package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Tab     key.Binding
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Refresh key.Binding
	Quit    key.Binding
}

var keys = keyMap{
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch view")),
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev granularity")),
	Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next granularity")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
