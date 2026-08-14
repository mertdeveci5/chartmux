package chartmux

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
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

	// Inspect adds a contextual value panel and highlights a selected mark.
	Inspect bool
	// FocusIndex and FocusSeries are zero-based and clamp to available data.
	FocusIndex  int
	FocusSeries int
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
	state := chart.terminalState(options)
	plot, err := chart.terminalPlot(options.Width, plotHeight, state)
	if err != nil {
		return "", err
	}

	var parts []string
	titleStyle := lipgloss.NewStyle().Bold(true)
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
	if inspection := chart.terminalInspection(state, options.Width); inspection != "" {
		parts = append(parts, "", inspection)
	}
	if chart.spec.Type != Heatmap && displayValue(chart.spec.Legend, len(chart.series) > 1) {
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

func (chart *Chart) terminalPlot(width, height int, state terminalRenderState) (string, error) {
	switch chart.spec.Type {
	case Bar, Histogram:
		return chart.terminalBarsWithState(width, height, state)
	case Line, Area, Scatter:
		return chart.terminalCartesianFrame(width, height, nil, false, state)
	case Combo:
		return chart.terminalComboFrame(width, height, state)
	case Pie, Donut:
		return chart.terminalProportionsWithState(width, height, state)
	case Heatmap:
		return chart.terminalHeatmapWithState(width, height, state)
	case Radar:
		return chart.terminalRadarWithState(width, height, state)
	case Funnel:
		return chart.terminalFunnelWithState(width, height, state)
	default:
		return "", fmt.Errorf("terminal output does not support %q", chart.spec.Type)
	}
}

func (chart *Chart) terminalLegend(width int) string {
	items := make([]string, len(chart.series))
	for index, series := range chart.series {
		markerText := string(terminalMarker(index)) + "··"
		if chart.spec.Type == Bar || chart.spec.Type == Histogram || (chart.spec.Type == Combo && series.spec.Mark == MarkBar) {
			markerText = string(terminalBarPattern(index))
		} else if chart.spec.Type == Area {
			markerText = string(terminalAreaPattern(index)) + "·"
		}
		marker := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color)).Render(markerText)
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
