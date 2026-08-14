package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mertdeveci5/chartmux"
)

func testWatchModel(t *testing.T) watchModel {
	t.Helper()
	spec, err := chartmux.Demo("grouped-bar")
	if err != nil {
		t.Fatal(err)
	}
	chart, err := chartmux.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	return newWatchModel(chart)
}

func testWatchModelFor(t *testing.T, name string) watchModel {
	t.Helper()
	spec, err := chartmux.Demo(name)
	if err != nil {
		t.Fatal(err)
	}
	chart, err := chartmux.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	return newWatchModel(chart)
}

func TestWatchModelUsesNativeUnicodeView(t *testing.T) {
	model := testWatchModel(t)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := updated.(watchModel).View()
	if !view.AltScreen || view.WindowTitle != "chartmux" {
		t.Fatalf("watch view did not configure the terminal: %+v", view)
	}
	if !strings.Contains(view.Content, "█") || !strings.Contains(view.Content, "UNICODE") {
		t.Fatalf("watch view is not a native block chart:\n%s", view.Content)
	}
	if strings.Contains(strings.ToLower(view.Content), "kitty") {
		t.Fatalf("watch view still advertises image rendering:\n%s", view.Content)
	}
	if lines := terminalLineCount(view.Content); lines > 24 {
		t.Fatalf("watch view uses %d lines in a 24-row terminal:\n%s", lines, view.Content)
	}
}

func TestWatchModelHelpAndClipboardActions(t *testing.T) {
	model := testWatchModel(t)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = updated.(watchModel)

	updated, _ = model.Update(tea.KeyPressMsg(tea.Key{Code: '?', Text: "?"}))
	model = updated.(watchModel)
	if !model.showHelp || !strings.Contains(model.View().Content, "ctrl+z suspend") {
		t.Fatalf("expanded help was not shown:\n%s", model.View().Content)
	}

	updated, command := model.Update(tea.KeyPressMsg(tea.Key{Code: 'c', Text: "c"}))
	model = updated.(watchModel)
	if command == nil || model.notice != "copied chart" {
		t.Fatalf("copy action did not produce feedback or a command: %+v", model)
	}

	updated, _ = model.Update(clearWatchNoticeMsg(model.noticeID - 1))
	if updated.(watchModel).notice == "" {
		t.Fatal("a stale notice timer cleared current feedback")
	}
	updated, _ = model.Update(clearWatchNoticeMsg(model.noticeID))
	if updated.(watchModel).notice != "" {
		t.Fatal("the current notice timer did not clear feedback")
	}
}

func TestWatchModelSuspendsThroughBubbleTea(t *testing.T) {
	model := testWatchModel(t)
	_, command := model.Update(tea.KeyPressMsg(tea.Key{Code: 'z', Mod: tea.ModCtrl}))
	if command == nil {
		t.Fatal("ctrl+z did not return a Bubble Tea command")
	}
	if _, ok := command().(tea.SuspendMsg); !ok {
		t.Fatalf("ctrl+z command returned %T, want tea.SuspendMsg", command())
	}
}

func TestWatchModelReportsShortTerminal(t *testing.T) {
	model := testWatchModel(t)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 5})
	view := updated.(watchModel).View().Content
	if !strings.Contains(view, "terminal too short") {
		t.Fatalf("short terminal error was not rendered:\n%s", view)
	}
}

func TestAnnotatedWatchViewFitsTerminal(t *testing.T) {
	model := testWatchModelFor(t, "annotated-bar")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 84, Height: 26})
	view := updated.(watchModel).View().Content
	if strings.Contains(view, "terminal is too short") {
		t.Fatalf("annotated chart did not fit a standard terminal:\n%s", view)
	}
	if lines := terminalLineCount(view); lines > 26 {
		t.Fatalf("annotated watch view uses %d rows, want at most 26:\n%s", lines, view)
	}
}
