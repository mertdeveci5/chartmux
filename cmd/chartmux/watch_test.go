package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mertdeveci5/chartmux"
)

func TestWatchModelUsesUnicodeWithoutImageProtocol(t *testing.T) {
	spec, err := chartmux.Demo("line")
	if err != nil {
		t.Fatal(err)
	}
	chart, err := chartmux.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	model := newWatchModel(chart, "unicode")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := updated.(watchModel).View()
	if view.Content == "" {
		t.Fatal("watch view is empty")
	}
}
