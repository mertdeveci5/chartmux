package chartmux

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

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

	var parts []string
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(terminalColor(chart.series[0].spec.Color))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))
	parts = append(parts, terminalStyledLines(chart.spec.Title, options.Width, titleStyle)...)
	parts = append(parts, terminalStyledLines(chart.spec.Description, options.Width, mutedStyle)...)
	for _, annotation := range chart.spec.Annotations {
		if annotationPosition(annotation) == AnnotationTop {
			parts = append(parts, chart.terminalAnnotationLines(annotation, options.Width)...)
		}
	}
	if len(parts) > 0 {
		parts = append(parts, "")
	}
	parts = append(parts, plot)
	if displayValue(chart.spec.Legend, len(chart.series) > 1) {
		parts = append(parts, "", chart.terminalLegend(options.Width))
	}
	for _, annotation := range chart.spec.Annotations {
		if annotationPosition(annotation) == AnnotationBottom {
			parts = append(parts, "")
			parts = append(parts, chart.terminalAnnotationLines(annotation, options.Width)...)
		}
	}
	if chart.spec.Footer != "" {
		parts = append(parts, "")
		parts = append(parts, terminalStyledLines(chart.spec.Footer, options.Width, mutedStyle)...)
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
	case Pie, Donut:
		return chart.terminalProportions(width, height)
	case Heatmap:
		return chart.terminalHeatmap(width, height)
	case Radar:
		return chart.terminalRadar(width, height)
	case Funnel:
		return chart.terminalFunnel(width, height)
	default:
		return "", fmt.Errorf("terminal output does not support %q", chart.spec.Type)
	}
}

func (chart *Chart) terminalCartesian(width, height int, include map[int]bool, zeroBaseline bool) string {
	displayed := chart.terminalCartesianValues()
	minimum, maximum := valueRangeForSeries(displayed, include)
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
	minX := 0.0
	maxX := float64(max(1, len(chart.labels)-1))
	if chart.spec.Type == Scatter && len(chart.xValues) > 0 {
		minX, maxX = chart.xValues[0], chart.xValues[0]
		for _, value := range chart.xValues[1:] {
			minX = math.Min(minX, value)
			maxX = math.Max(maxX, value)
		}
		if minX == maxX {
			minX--
			maxX++
		}
	}
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
		minX,
		maxX,
		minimum,
		maximum,
		linechart.WithXYSteps(xStep, yStep),
		linechart.WithStyles(axisStyle, labelStyle, lipgloss.NewStyle()),
		linechart.WithXLabelFormatter(func(_ int, value float64) string {
			if chart.spec.Type == Scatter {
				return formatValue(value)
			}
			index := min(len(chart.labels)-1, max(0, int(math.Round(value))))
			return terminalSafeText(chart.labels[index])
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
		style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
		lineStyle := runes.ThinLineStyle
		if chart.spec.Curve == Smooth {
			lineStyle = runes.ArcLineStyle
		}
		values := displayed[seriesIndex]
		if chart.spec.Type != Scatter {
			for pointIndex := 1; pointIndex < len(values); pointIndex++ {
				left := values[pointIndex-1]
				right := values[pointIndex]
				if isMissing(left) || isMissing(right) {
					continue
				}
				model.DrawLineWithStyle(
					canvas.Float64Point{X: float64(pointIndex - 1), Y: left},
					canvas.Float64Point{X: float64(pointIndex), Y: right},
					lineStyle,
					style,
				)
			}
			for pointIndex, value := range values {
				if !isMissing(value) {
					model.DrawRuneWithStyle(canvas.Float64Point{X: float64(pointIndex), Y: value}, terminalMarker(seriesIndex), style)
				}
			}
		}
		if chart.spec.Type == Scatter {
			for pointIndex, value := range values {
				if !isMissing(value) {
					model.DrawRuneWithStyle(canvas.Float64Point{X: chart.xValues[pointIndex], Y: value}, terminalMarker(seriesIndex), style)
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
			return terminalSafeText(chart.labels[index])
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
		style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
		for pointIndex, value := range series.values {
			if isMissing(value) {
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
	for seriesIndex, series := range chart.series {
		if series.spec.Mark != MarkLine {
			continue
		}
		style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
		for pointIndex := 1; pointIndex < len(series.values); pointIndex++ {
			left := series.values[pointIndex-1]
			right := series.values[pointIndex]
			if isMissing(left) || isMissing(right) {
				continue
			}
			model.DrawLineWithStyle(
				canvas.Float64Point{X: float64(pointIndex - 1), Y: left},
				canvas.Float64Point{X: float64(pointIndex), Y: right},
				runes.ArcLineStyle,
				style,
			)
		}
		for pointIndex, value := range series.values {
			if !isMissing(value) {
				model.DrawRuneWithStyle(canvas.Float64Point{X: float64(pointIndex), Y: value}, terminalMarker(seriesIndex), style)
			}
		}
	}
	return model.Canvas.View()
}

func (chart *Chart) terminalBars(width, height int) (string, error) {
	for _, series := range chart.series {
		for _, value := range series.values {
			if !isMissing(value) && value < 0 {
				return chart.terminalDivergingBars(width, height)
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
				if isMissing(value) {
					value = 0
				}
				values = append(values, barchart.BarValue{Name: series.spec.Label, Value: value, Style: lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))})
			}
			data = append(data, barchart.BarData{Label: terminalSafeText(label), Values: values})
			continue
		}
		for seriesIndex, series := range chart.series {
			value := series.values[pointIndex]
			if isMissing(value) {
				value = 0
			}
			barLabel := ""
			if seriesIndex == 0 {
				barLabel = terminalSafeText(label)
			}
			data = append(data, barchart.BarData{Label: barLabel, Values: []barchart.BarValue{{Name: series.spec.Label, Value: value, Style: lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))}}})
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

func (chart *Chart) terminalLegend(width int) string {
	items := make([]string, len(chart.series))
	for index, series := range chart.series {
		marker := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color)).Render(string(terminalMarker(index)) + "━")
		items[index] = marker + " " + terminalSafeText(series.spec.Label)
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
			if isMissing(value) {
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

func (chart *Chart) terminalCartesianValues() [][]float64 {
	values := chart.seriesValues()
	if chart.spec.Type != Area || (chart.spec.Layout != Stacked && chart.spec.Layout != Normalized) {
		return values
	}
	for pointIndex := range chart.labels {
		total := 0.0
		for seriesIndex := range values {
			if isMissing(values[seriesIndex][pointIndex]) {
				continue
			}
			total += values[seriesIndex][pointIndex]
			values[seriesIndex][pointIndex] = total
		}
	}
	return values
}

func valueRangeForSeries(values [][]float64, include map[int]bool) (float64, float64) {
	minimum := math.Inf(1)
	maximum := math.Inf(-1)
	for seriesIndex, series := range values {
		if include != nil && !include[seriesIndex] {
			continue
		}
		for _, value := range series {
			if isMissing(value) {
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

var terminalMarkers = [...]rune{'●', '◆', '■', '▲', '✦', '✚'}

func terminalMarker(index int) rune {
	return terminalMarkers[index%len(terminalMarkers)]
}

// terminalColor resolves schema palette tokens before they reach Lip Gloss.
// Lip Gloss accepts concrete terminal colors, not CSS var(--chart-N) values.
func terminalColor(value string) color.Color {
	color, err := colorHex(value)
	if err != nil {
		// Chart construction validates colors, so this is only a defensive
		// fallback for future internal call sites.
		return lipgloss.Color("#71717A")
	}
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", color.R, color.G, color.B))
}

func terminalSafeText(value string) string {
	var safe strings.Builder
	for index := 0; index < len(value); {
		if value[index] == 0x1b {
			index = skipTerminalEscape(value, index)
			continue
		}
		char, size := utf8.DecodeRuneInString(value[index:])
		if char == utf8.RuneError && size == 1 {
			index++
			continue
		}
		index += size
		if char == '\n' || char == '\r' || char == '\t' {
			safe.WriteByte(' ')
			continue
		}
		if unicode.IsControl(char) {
			continue
		}
		safe.WriteRune(char)
	}
	return safe.String()
}

func skipTerminalEscape(value string, start int) int {
	index := start + 1
	if index >= len(value) {
		return index
	}
	switch value[index] {
	case '[':
		index++
		for index < len(value) {
			char := value[index]
			index++
			if char >= 0x40 && char <= 0x7e {
				break
			}
		}
	case ']', 'P', 'X', '^', '_':
		index++
		for index < len(value) {
			if value[index] == 0x07 {
				return index + 1
			}
			if value[index] == 0x1b && index+1 < len(value) && value[index+1] == '\\' {
				return index + 2
			}
			index++
		}
	default:
		index++
	}
	return index
}

func terminalStyledLines(value string, width int, style lipgloss.Style) []string {
	value = strings.TrimSpace(terminalSafeText(value))
	if value == "" {
		return nil
	}
	wrapped := ansi.Hardwrap(ansi.Wordwrap(value, width, ""), width, false)
	lines := strings.Split(wrapped, "\n")
	for index := range lines {
		lines[index] = style.Render(lines[index])
	}
	return lines
}

func (chart *Chart) terminalAnnotationLines(annotation Annotation, width int) []string {
	color := annotation.Color
	if color == "" {
		color = "#71717A"
	}
	style := lipgloss.NewStyle().Foreground(terminalColor(color))
	return terminalStyledLines("◆ "+chart.annotationText(annotation), width, style)
}

func formatValue(value float64) string {
	abs := math.Abs(value)
	switch {
	case abs >= 1_000_000_000_000:
		return fmt.Sprintf("%.1ftn", value/1_000_000_000_000)
	case abs >= 1_000_000_000:
		return fmt.Sprintf("%.1fbn", value/1_000_000_000)
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
