package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/mertdeveci5/chartmux"
)

const watchNoticeDuration = 2 * time.Second

type clearWatchNoticeMsg uint64

type watchModel struct {
	chart       *chartmux.Chart
	width       int
	height      int
	showHelp    bool
	inspect     bool
	focusIndex  int
	focusSeries int
	notice      string
	noticeID    uint64
	err         error
}

func newWatchModel(chart *chartmux.Chart) watchModel {
	return watchModel{chart: chart}
}

func (watchModel) Init() tea.Cmd {
	return nil
}

func (model watchModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		model.width = message.Width
		model.height = message.Height
		model.err = nil
	case tea.KeyPressMsg:
		switch message.String() {
		case "q", "ctrl+c":
			return model, tea.Quit
		case "esc":
			if model.inspect {
				model.inspect = false
				return model, nil
			}
			return model, tea.Quit
		case "ctrl+z":
			return model, tea.Suspend
		case "left":
			model.inspect = true
			model.focusIndex = wrapWatchSelection(model.focusIndex-1, model.chart.PointCount())
		case "right":
			model.inspect = true
			model.focusIndex = wrapWatchSelection(model.focusIndex+1, model.chart.PointCount())
		case "up":
			model.inspect = true
			model.focusSeries = wrapWatchSelection(model.focusSeries-1, model.chart.SeriesCount())
		case "down":
			model.inspect = true
			model.focusSeries = wrapWatchSelection(model.focusSeries+1, model.chart.SeriesCount())
		case "?":
			model.showHelp = !model.showHelp
		case "c":
			content, err := model.renderChart()
			if err != nil {
				model.err = err
				return model, nil
			}
			model.noticeID++
			model.notice = "copied chart"
			return model, tea.Batch(
				tea.SetClipboard(ansi.Strip(content)),
				clearWatchNoticeAfter(model.noticeID),
			)
		}
	case clearWatchNoticeMsg:
		if uint64(message) == model.noticeID {
			model.notice = ""
		}
	case tea.ResumeMsg:
		return model, tea.RequestWindowSize
	}
	return model, nil
}

func (model watchModel) View() tea.View {
	width, _ := model.dimensions()
	content, err := model.renderChart()
	if model.err != nil {
		err = model.err
	}
	if err != nil {
		content = errorStyle.Render(ansi.Hardwrap(err.Error(), max(1, width), false))
	}

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, content, model.footer(width)))
	view.AltScreen = true
	view.WindowTitle = "chartmux"
	return view
}

func (model watchModel) renderChart() (string, error) {
	width, height := model.dimensions()
	availableHeight := height - 1
	if model.showHelp {
		availableHeight--
	}
	if availableHeight < chartmux.MinTerminalHeight {
		return "", fmt.Errorf("terminal too short (%d rows; need at least %d)", height, chartmux.MinTerminalHeight+1)
	}

	plotHeight := availableHeight
	for plotHeight >= chartmux.MinTerminalHeight {
		content, err := model.chart.Terminal(chartmux.TerminalOptions{
			Width:       width,
			Height:      plotHeight,
			Inspect:     model.inspect,
			FocusIndex:  model.focusIndex,
			FocusSeries: model.focusSeries,
		})
		if err != nil {
			return "", err
		}
		overflow := terminalLineCount(content) - availableHeight
		if overflow <= 0 {
			return content, nil
		}
		plotHeight -= max(1, overflow)
	}
	return "", fmt.Errorf("terminal is too short to display this chart")
}

func (model watchModel) footer(width int) string {
	_, height := model.dimensions()
	left := fmt.Sprintf("UNICODE · %d×%d", width, height)
	right := "←→ inspect · ? help · c copy · q quit"
	if model.notice != "" {
		right = model.notice
	}
	left = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")).Render(left)
	right = lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Render(right)
	gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
	footer := right
	if gap >= 1 {
		footer = left + strings.Repeat(" ", gap) + right
	}
	footer = ansi.Truncate(footer, max(1, width), "…")
	if !model.showHelp {
		return footer
	}
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Render("←→ point · ↑↓ series · esc close · c copy · ctrl+z suspend · ? close help · q quit")
	return lipgloss.JoinVertical(lipgloss.Left, ansi.Truncate(help, max(1, width), "…"), footer)
}

func wrapWatchSelection(value, count int) int {
	if count <= 0 {
		return 0
	}
	value %= count
	if value < 0 {
		value += count
	}
	return value
}

func (model watchModel) dimensions() (int, int) {
	width := model.width
	height := model.height
	if width == 0 {
		width = chartmux.DefaultTerminalWidth
	}
	if height == 0 {
		height = chartmux.DefaultTerminalHeight + 8
	}
	return width, height
}

func clearWatchNoticeAfter(id uint64) tea.Cmd {
	return tea.Tick(watchNoticeDuration, func(time.Time) tea.Msg {
		return clearWatchNoticeMsg(id)
	})
}

func terminalLineCount(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func watch(chart *chartmux.Chart) error {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return fmt.Errorf("--watch requires an interactive terminal")
	}
	_, err := tea.NewProgram(newWatchModel(chart), tea.WithFPS(30)).Run()
	return err
}
