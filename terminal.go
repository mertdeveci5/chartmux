package chartmux

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/barchart"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	"github.com/NimbleMarkets/ntcharts/v2/linechart"
	"github.com/charmbracelet/x/ansi"
)

const (
	DefaultTerminalWidth  = 80
	DefaultTerminalHeight = 14
	MinTerminalWidth      = 30
	MinTerminalHeight     = 8
)

type TerminalOptions struct {
	Width  int
	Height int
}

type TerminalSizeError struct {
	Width  int
	Height int
}

func (err *TerminalSizeError) Error() string {
	return fmt.Sprintf("terminal too small (%dx%d; need at least %dx%d)", err.Width, err.Height, MinTerminalWidth, MinTerminalHeight)
}

func (chart *Chart) Terminal(options TerminalOptions) (string, error) {
	if options.Width == 0 {
		options.Width = DefaultTerminalWidth
	}
	if options.Height == 0 {
		options.Height = DefaultTerminalHeight
	}
	if options.Width < MinTerminalWidth || options.Height < MinTerminalHeight {
		return "", &TerminalSizeError{Width: options.Width, Height: options.Height}
	}
	if displayValue(chart.spec.Labels, false) && (chart.spec.Type == Bar || chart.spec.Type == Histogram || chart.spec.Type == Line || chart.spec.Type == Area || chart.spec.Type == Scatter || chart.spec.Type == Combo) {
		return "", fmt.Errorf("terminal output does not support value labels; use --export png, svg, or html")
	}
	plotHeight := options.Height
	plot, err := chart.terminalPlot(options.Width, plotHeight)
	if err != nil {
		return "", err
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(chart.series[0].spec.Color)).Render(chart.spec.Title)
	parts := []string{ansi.Truncate(title, options.Width, "…")}
	if chart.spec.Description != "" {
		parts = append(parts, ansi.Truncate(lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Render(chart.spec.Description), options.Width, "…"))
	}
	parts = append(parts, "", plot)
	if displayValue(chart.spec.Legend, len(chart.series) > 1) {
		parts = append(parts, "", chart.terminalLegend(options.Width))
	}
	if chart.spec.Footer != "" {
		parts = append(parts, "", ansi.Truncate(lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Render(chart.spec.Footer), options.Width, "…"))
	}
	return strings.Join(parts, "\n"), nil
}

func (chart *Chart) terminalPlot(width, height int) (string, error) {
	switch chart.spec.Type {
	case Bar, Histogram:
		return chart.terminalBars(width, height)
	case Line, Area, Scatter:
		return chart.terminalCartesian(width, height, nil, false), nil
	case Combo:
		return chart.terminalCombo(width, height), nil
	case Pie, Donut, Heatmap, Radar, Funnel:
		return chart.terminalTable(width), nil
	default:
		return "", fmt.Errorf("terminal output does not support %q", chart.spec.Type)
	}
}

func (chart *Chart) terminalCartesian(width, height int, include map[int]bool, zeroBaseline bool) string {
	minimum, maximum := chart.valueRange(include)
	if chart.spec.Type == Area || zeroBaseline {
		minimum = math.Min(0, minimum)
		maximum = math.Max(0, maximum)
	}
	if minimum == maximum {
		minimum--
		maximum++
	}
	padding := (maximum - minimum) * 0.08
	minimum -= padding
	maximum += padding
	maxX := float64(max(1, len(chart.labels)-1))
	xStep := max(1, (width-8)/min(6, max(1, len(chart.labels))))
	yStep := 2
	if !displayValue(chart.spec.Axes, true) {
		xStep = 0
		yStep = 0
	}
	axisStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))
	model := linechart.New(
		width,
		height,
		0,
		maxX,
		minimum,
		maximum,
		linechart.WithXYSteps(xStep, yStep),
		linechart.WithStyles(axisStyle, labelStyle, lipgloss.NewStyle()),
		linechart.WithXLabelFormatter(func(_ int, value float64) string {
			index := min(len(chart.labels)-1, max(0, int(math.Round(value))))
			return chart.labels[index]
		}),
		linechart.WithYLabelFormatter(func(_ int, value float64) string {
			return formatValue(value)
		}),
	)
	if displayValue(chart.spec.Axes, true) {
		model.DrawXYAxisAndLabel()
	}
	for seriesIndex, series := range chart.series {
		if include != nil && !include[seriesIndex] {
			continue
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(series.spec.Color))
		lineStyle := runes.ThinLineStyle
		if chart.spec.Curve == Smooth {
			lineStyle = runes.ArcLineStyle
		}
		for pointIndex := 1; pointIndex < len(series.values); pointIndex++ {
			left := series.values[pointIndex-1]
			right := series.values[pointIndex]
			if left == math.MaxFloat64 || right == math.MaxFloat64 {
				continue
			}
			model.DrawLineWithStyle(
				canvas.Float64Point{X: float64(pointIndex - 1), Y: left},
				canvas.Float64Point{X: float64(pointIndex), Y: right},
				lineStyle,
				style,
			)
		}
		if chart.spec.Type == Scatter {
			for pointIndex, value := range series.values {
				if value != math.MaxFloat64 {
					model.DrawRuneWithStyle(canvas.Float64Point{X: float64(pointIndex), Y: value}, '●', style)
				}
			}
		}
	}
	return model.Canvas.View()
}

func (chart *Chart) terminalCombo(width, height int) string {
	include := make(map[int]bool, len(chart.series))
	for index := range chart.series {
		include[index] = true
	}
	minimum, maximum := chart.valueRange(include)
	minimum = math.Min(0, minimum)
	maximum = math.Max(0, maximum)
	if minimum == maximum {
		maximum++
	}
	padding := (maximum - minimum) * 0.08
	minimum -= padding
	maximum += padding
	maxX := float64(max(1, len(chart.labels)-1))
	xStep := max(1, (width-8)/min(6, max(1, len(chart.labels))))
	yStep := 2
	if !displayValue(chart.spec.Axes, true) {
		xStep = 0
		yStep = 0
	}
	axisStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))
	model := linechart.New(
		width,
		height,
		0,
		maxX,
		minimum,
		maximum,
		linechart.WithXYSteps(xStep, yStep),
		linechart.WithStyles(axisStyle, labelStyle, lipgloss.NewStyle()),
		linechart.WithXLabelFormatter(func(_ int, value float64) string {
			index := min(len(chart.labels)-1, max(0, int(math.Round(value))))
			return chart.labels[index]
		}),
		linechart.WithYLabelFormatter(func(_ int, value float64) string {
			return formatValue(value)
		}),
	)
	if displayValue(chart.spec.Axes, true) {
		model.DrawXYAxisAndLabel()
	}
	for _, series := range chart.series {
		if series.spec.Mark != MarkBar {
			continue
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(series.spec.Color))
		for pointIndex, value := range series.values {
			if value == math.MaxFloat64 {
				continue
			}
			model.DrawRuneLineWithStyle(
				canvas.Float64Point{X: float64(pointIndex), Y: 0},
				canvas.Float64Point{X: float64(pointIndex), Y: value},
				runes.FullBlock,
				style,
			)
		}
	}
	for _, series := range chart.series {
		if series.spec.Mark != MarkLine {
			continue
		}
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(series.spec.Color))
		for pointIndex := 1; pointIndex < len(series.values); pointIndex++ {
			left := series.values[pointIndex-1]
			right := series.values[pointIndex]
			if left == math.MaxFloat64 || right == math.MaxFloat64 {
				continue
			}
			model.DrawLineWithStyle(
				canvas.Float64Point{X: float64(pointIndex - 1), Y: left},
				canvas.Float64Point{X: float64(pointIndex), Y: right},
				runes.ArcLineStyle,
				style,
			)
		}
	}
	return model.Canvas.View()
}

func (chart *Chart) terminalBars(width, height int) (string, error) {
	for _, series := range chart.series {
		for _, value := range series.values {
			if value != math.MaxFloat64 && value < 0 {
				return "", fmt.Errorf("signed bars need terminal graphics; use --watch or export SVG, PNG, or HTML")
			}
		}
	}
	model := barchart.New(width, height)
	model.SetHorizontal(chart.spec.Orientation == Horizontal)
	if chart.spec.Orientation == Horizontal {
		model.SetBarGap(0)
	}
	stacked := chart.spec.Layout == Stacked || chart.spec.Layout == Normalized
	data := make([]barchart.BarData, 0, len(chart.labels)*len(chart.series))
	for pointIndex, label := range chart.labels {
		if stacked {
			values := make([]barchart.BarValue, 0, len(chart.series))
			for _, series := range chart.series {
				value := series.values[pointIndex]
				if value == math.MaxFloat64 {
					value = 0
				}
				values = append(values, barchart.BarValue{Name: series.spec.Label, Value: value, Style: lipgloss.NewStyle().Foreground(lipgloss.Color(series.spec.Color))})
			}
			data = append(data, barchart.BarData{Label: label, Values: values})
			continue
		}
		for seriesIndex, series := range chart.series {
			value := series.values[pointIndex]
			if value == math.MaxFloat64 {
				value = 0
			}
			barLabel := ""
			if seriesIndex == 0 {
				barLabel = label
			}
			data = append(data, barchart.BarData{Label: barLabel, Values: []barchart.BarValue{{Name: series.spec.Label, Value: value, Style: lipgloss.NewStyle().Foreground(lipgloss.Color(series.spec.Color))}}})
		}
	}
	model.PushAll(data)
	model.SetShowAxis(displayValue(chart.spec.Axes, true))
	if model.BarWidth() < 1 {
		return "", fmt.Errorf("terminal is too small for %d bars; increase the chart height or reduce the series", len(data))
	}
	model.Draw()
	return model.View(), nil
}

func (chart *Chart) terminalTable(width int) string {
	var rows []string
	for pointIndex, label := range chart.labels {
		values := make([]string, 0, len(chart.series))
		for _, series := range chart.series {
			value := series.values[pointIndex]
			if value == math.MaxFloat64 {
				values = append(values, "–")
			} else {
				values = append(values, formatValue(value))
			}
		}
		line := fmt.Sprintf("%-16s %s", ansi.Truncate(label, 16, "…"), strings.Join(values, "  "))
		rows = append(rows, ansi.Truncate(line, width, "…"))
	}
	return strings.Join(rows, "\n")
}

func (chart *Chart) terminalLegend(width int) string {
	items := make([]string, len(chart.series))
	for index, series := range chart.series {
		marker := lipgloss.NewStyle().Foreground(lipgloss.Color(series.spec.Color)).Render("━━")
		items[index] = marker + " " + series.spec.Label
	}
	return ansi.Truncate(strings.Join(items, "   "), width, "…")
}

func (chart *Chart) valueRange(include map[int]bool) (float64, float64) {
	minimum := math.Inf(1)
	maximum := math.Inf(-1)
	for index, series := range chart.series {
		if include != nil && !include[index] {
			continue
		}
		for _, value := range series.values {
			if value == math.MaxFloat64 {
				continue
			}
			minimum = math.Min(minimum, value)
			maximum = math.Max(maximum, value)
		}
	}
	if math.IsInf(minimum, 1) {
		return 0, 1
	}
	return minimum, maximum
}

func formatValue(value float64) string {
	abs := math.Abs(value)
	switch {
	case abs >= 1_000_000:
		return fmt.Sprintf("%.1fm", value/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%.1fk", value/1_000)
	case value == math.Trunc(value):
		return fmt.Sprintf("%.0f", value)
	default:
		return fmt.Sprintf("%.1f", value)
	}
}
