package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/mertdeveci5/chartmux"
)

type chartImageMsg struct {
	image image.Image
	err   error
}

type watchModel struct {
	chart         *chartmux.Chart
	picture       picture.Model
	mode          string
	width         int
	height        int
	preferUnicode bool
	imageReady    bool
	err           error
}

func newWatchModel(chart *chartmux.Chart, mode string) watchModel {
	return watchModel{
		chart:   chart,
		picture: picture.NewWithConfig(picture.Config{Fit: picture.FitFill}),
		mode:    mode,
	}
}

func (model watchModel) Init() tea.Cmd {
	if model.mode == "unicode" {
		return nil
	}
	return model.picture.Init()
}

func (model watchModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd
	if model.mode != "unicode" {
		if command := model.picture.Update(message); command != nil {
			commands = append(commands, command)
		}
	}

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		plotRows := max(1, model.height-2)
		if model.mode != "unicode" {
			if command := model.picture.SetSize(model.width, plotRows); command != nil {
				commands = append(commands, command)
			}
			commands = append(commands, model.imageCommand(model.width, plotRows))
		}
	case tea.KeyPressMsg:
		switch message.String() {
		case "q", "ctrl+c", "esc":
			return model, tea.Quit
		case "g":
			if model.mode == "unicode" {
				break
			}
			model.preferUnicode = !model.preferUnicode
			if model.preferUnicode && model.picture.Mode() == picture.PictureKitty {
				commands = append(commands, model.picture.Toggle())
			} else if !model.preferUnicode && model.picture.KittySupported() == picture.KittyCapabilitySupported && model.picture.Mode() != picture.PictureKitty {
				commands = append(commands, model.picture.Toggle())
			}
		}
	case chartImageMsg:
		model.err = message.err
		if message.err == nil {
			model.imageReady = true
			if command := model.picture.SetImage(message.image); command != nil {
				commands = append(commands, command)
			}
		}
	}

	if model.mode != "unicode" && !model.preferUnicode && model.picture.KittySupported() == picture.KittyCapabilitySupported && model.picture.Mode() != picture.PictureKitty {
		if command := model.picture.Toggle(); command != nil {
			commands = append(commands, command)
		}
	}
	return model, tea.Batch(commands...)
}

func (model watchModel) View() tea.View {
	width := model.width
	height := model.height
	if width == 0 {
		width = chartmux.DefaultTerminalWidth
	}
	if height == 0 {
		height = chartmux.DefaultTerminalHeight + 7
	}
	modeLabel := "Unicode"
	content := ""
	if model.err != nil {
		content = errorStyle.Render(ansi.Hardwrap(model.err.Error(), max(1, width), false))
	} else if model.mode == "kitty" && model.picture.KittySupported() == picture.KittyCapabilityUnsupported {
		content = errorStyle.Render(ansi.Hardwrap("Kitty graphics are not supported by this terminal; rerun with --terminal-mode unicode", max(1, width), false))
		modeLabel = "Kitty unavailable"
	} else if !model.preferUnicode && model.imageReady && model.picture.Mode() == picture.PictureKitty {
		content = model.picture.View().Content
		modeLabel = "Kitty image"
	} else {
		output, err := model.chart.Terminal(chartmux.TerminalOptions{Width: width, Height: max(1, height-8)})
		if err != nil {
			content = errorStyle.Render(ansi.Hardwrap(err.Error(), max(1, width), false))
		} else {
			content = output
		}
	}
	help := " · q quit"
	if model.mode != "unicode" && model.picture.KittySupported() == picture.KittyCapabilitySupported {
		help = " · g switch presentation · q quit"
	}
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Render(modeLabel + help)
	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, footer))
	view.AltScreen = true
	view.WindowTitle = "chartmux"
	return view
}

func (model watchModel) imageCommand(columns, rows int) tea.Cmd {
	return func() tea.Msg {
		width := min(2400, max(320, columns*10))
		height := min(1600, max(240, rows*20))
		content, err := model.chart.PNG(chartmux.ImageOptions{Width: width, Height: height})
		if err != nil {
			return chartImageMsg{err: err}
		}
		decoded, _, err := image.Decode(bytes.NewReader(content))
		return chartImageMsg{image: decoded, err: err}
	}
}

func watch(chart *chartmux.Chart, mode string) error {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return fmt.Errorf("--watch requires an interactive terminal")
	}
	_, err := tea.NewProgram(newWatchModel(chart, mode)).Run()
	return err
}
