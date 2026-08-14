package chartmux

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var horizontalFractions = [...]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}

func (chart *Chart) terminalProportions(width, height int) (string, error) {
	values := chart.series[0].values
	requiredHeight := len(values) + 1
	if height < requiredHeight {
		return "", fmt.Errorf("terminal is too short for %d categories; increase the chart height", len(values))
	}
	total := 0.0
	for _, value := range values {
		if !isMissing(value) {
			total += value
		}
	}

	rows := make([]string, 0, min(height, len(values)+1))
	rows = append(rows, chart.terminalSegmentBand(values, total, width))
	labelWidth := terminalLabelWidth(chart.labels, 8, min(20, width/3))
	for index, value := range values {
		if len(rows) >= height {
			break
		}
		style := lipgloss.NewStyle().Foreground(terminalColor(defaultColors[index%len(defaultColors)]))
		valueText := "–"
		percentText := "  0.0%"
		if !isMissing(value) {
			valueText = formatValue(value)
			if total > 0 {
				percentText = fmt.Sprintf("%5.1f%%", value/total*100)
			}
		}
		line := style.Render(string(terminalMarker(index))+"█") + " " + padTerminal(terminalSafeText(chart.labels[index]), labelWidth) + "  " + valueText + "  " + percentText
		rows = append(rows, ansi.Truncate(line, width, "…"))
	}
	return strings.Join(rows, "\n"), nil
}

func (chart *Chart) terminalSegmentBand(values []float64, total float64, width int) string {
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	activeSegments := 0
	for _, value := range values {
		if !isMissing(value) && value > 0 {
			activeSegments++
		}
	}
	showSeparators := activeSegments > 1 && activeSegments*2-1 <= width
	separatorCount := 0
	if showSeparators {
		separatorCount = activeSegments - 1
	}
	dataWidth := max(1, width-separatorCount)
	var band strings.Builder
	used := 0
	cumulative := 0.0
	drawnSegments := 0
	for index, value := range values {
		if isMissing(value) || value <= 0 {
			continue
		}
		if showSeparators && drawnSegments > 0 {
			band.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#52525B")).Render("│"))
		}
		cumulative += value
		end := int(math.Round(cumulative / total * float64(dataWidth)))
		cells := max(0, end-used)
		if cells > 0 {
			style := lipgloss.NewStyle().Foreground(terminalColor(defaultColors[index%len(defaultColors)]))
			band.WriteString(style.Render(strings.Repeat("█", cells)))
			used += cells
		}
		drawnSegments++
	}
	if used < dataWidth {
		band.WriteString(strings.Repeat(" ", dataWidth-used))
	}
	return band.String()
}

func (chart *Chart) terminalHeatmap(width, height int) (string, error) {
	showAxes := displayValue(chart.spec.Axes, true)
	requiredHeight := len(chart.labels)
	if showAxes {
		requiredHeight++
	}
	if height < requiredHeight {
		return "", fmt.Errorf("terminal is too short for %d heatmap rows; increase the chart height", len(chart.labels))
	}
	labelWidth := 0
	if showAxes {
		labelWidth = terminalLabelWidth(chart.labels, 4, min(14, width/4))
	}
	prefixWidth := labelWidth
	if prefixWidth > 0 {
		prefixWidth++
	}
	gapWidth := max(0, len(chart.series)-1)
	if width-prefixWidth-gapWidth < len(chart.series) {
		return "", fmt.Errorf("terminal is too narrow for %d heatmap series; increase the width", len(chart.series))
	}
	cellWidth := max(4, (width-prefixWidth-gapWidth)/max(1, len(chart.series)))

	maximum := 0.0
	for _, series := range chart.series {
		for _, value := range series.values {
			if !isMissing(value) {
				maximum = math.Max(maximum, value)
			}
		}
	}
	if maximum == 0 {
		maximum = 1
	}

	rows := make([]string, 0, min(height, len(chart.labels)+1))
	if showAxes {
		header := strings.Repeat(" ", prefixWidth)
		for index, series := range chart.series {
			if index > 0 {
				header += " "
			}
			style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
			header += style.Render(padTerminal(terminalSafeText(series.spec.Label), cellWidth))
		}
		rows = append(rows, ansi.Truncate(header, width, "…"))
	}
	for pointIndex, label := range chart.labels {
		if len(rows) >= height {
			break
		}
		line := ""
		if showAxes {
			line = padTerminal(terminalSafeText(label), labelWidth) + " "
		}
		for seriesIndex, series := range chart.series {
			if seriesIndex > 0 {
				line += " "
			}
			line += terminalHeatCell(series.values[pointIndex], maximum, cellWidth, series.spec.Color)
		}
		rows = append(rows, ansi.Truncate(line, width, "…"))
	}
	return strings.Join(rows, "\n"), nil
}

func terminalHeatCell(value, maximum float64, width int, color string) string {
	if isMissing(value) {
		return padTerminal("·", width)
	}
	density := []rune{'░', '▒', '▓', '█'}
	index := min(len(density)-1, max(0, int(math.Round(value/maximum*float64(len(density)-1)))))
	valueText := formatValue(value)
	fillWidth := max(1, width-ansi.StringWidth(valueText)-1)
	fill := lipgloss.NewStyle().Foreground(terminalColor(color)).Render(strings.Repeat(string(density[index]), fillWidth))
	return padTerminal(fill+" "+valueText, width)
}

func (chart *Chart) terminalRadar(width, height int) (string, error) {
	showAxes := displayValue(chart.spec.Axes, true)
	requiredHeight := len(chart.labels) * len(chart.series)
	if height < requiredHeight {
		return "", fmt.Errorf("terminal is too short for %d radar rows; increase the chart height", requiredHeight)
	}
	labelWidth := 0
	if showAxes {
		labelWidth = terminalLabelWidth(chart.labels, 8, min(18, width/4))
	}
	seriesLabels := make([]string, len(chart.series))
	for index, series := range chart.series {
		seriesLabels[index] = series.spec.Label
	}
	seriesWidth := terminalLabelWidth(seriesLabels, 6, min(14, width/5))
	maximum := chart.spec.Max
	if maximum <= 0 {
		_, maximum = chart.valueRange(nil)
	}
	if maximum <= 0 {
		maximum = 1
	}
	valueWidth := terminalValueWidth(chart.series)
	barWidth := max(1, width-labelWidth-seriesWidth-valueWidth-4)

	rows := make([]string, 0, min(height, len(chart.labels)*len(chart.series)))
	for pointIndex, label := range chart.labels {
		for seriesIndex, series := range chart.series {
			if len(rows) >= height {
				return strings.Join(rows, "\n"), nil
			}
			metric := ""
			if showAxes {
				if seriesIndex == 0 {
					metric = terminalSafeText(label)
				}
				metric = padTerminal(metric, labelWidth) + " "
			}
			value := series.values[pointIndex]
			valueText := "–"
			ratio := 0.0
			if !isMissing(value) {
				valueText = formatValue(value)
				ratio = value / maximum
			}
			style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
			line := metric + padTerminal(terminalSafeText(series.spec.Label), seriesWidth) + " " + renderHorizontalBar(ratio, barWidth, style) + " " + padTerminalLeft(valueText, valueWidth)
			rows = append(rows, ansi.Truncate(line, width, "…"))
		}
	}
	return strings.Join(rows, "\n"), nil
}

func (chart *Chart) terminalFunnel(width, height int) (string, error) {
	showAxes := displayValue(chart.spec.Axes, true)
	showValues := displayValue(chart.spec.Labels, false)
	if height < len(chart.labels) {
		return "", fmt.Errorf("terminal is too short for %d funnel stages; increase the chart height", len(chart.labels))
	}
	labelWidth := 0
	if showAxes {
		labelWidth = terminalLabelWidth(chart.labels, 8, min(18, width/4))
	}
	valueWidth := 0
	if showValues {
		valueWidth = terminalValueWidth(chart.series)
	}
	maximum := 0.0
	for _, value := range chart.series[0].values {
		if !isMissing(value) {
			maximum = math.Max(maximum, value)
		}
	}
	if maximum == 0 {
		maximum = 1
	}
	barWidth := max(1, width-labelWidth-valueWidth-3)
	style := lipgloss.NewStyle().Foreground(terminalColor(chart.series[0].spec.Color))

	rows := make([]string, 0, min(height, len(chart.labels)))
	for index, label := range chart.labels {
		if len(rows) >= height {
			break
		}
		value := chart.series[0].values[index]
		ratio := 0.0
		valueText := "–"
		if value != math.MaxFloat64 {
			ratio = value / maximum
			valueText = formatValue(value)
		}
		bar := renderCenteredBar(ratio, barWidth, style)
		line := ""
		if showAxes {
			line = padTerminal(terminalSafeText(label), labelWidth) + " "
		}
		line += bar
		if showValues {
			line += " " + padTerminalLeft(valueText, valueWidth)
		}
		rows = append(rows, ansi.Truncate(line, width, "…"))
	}
	return strings.Join(rows, "\n"), nil
}

func renderHorizontalBar(ratio float64, width int, style lipgloss.Style) string {
	bar, cells := horizontalBar(ratio, width)
	return style.Render(bar) + strings.Repeat(" ", max(0, width-cells))
}

func renderCenteredBar(ratio float64, width int, style lipgloss.Style) string {
	bar, cells := horizontalBar(ratio, width)
	left := max(0, (width-cells)/2)
	right := max(0, width-cells-left)
	return strings.Repeat(" ", left) + style.Render(bar) + strings.Repeat(" ", right)
}

func horizontalBar(ratio float64, width int) (string, int) {
	ratio = math.Max(0, math.Min(1, ratio))
	units := int(math.Round(ratio * float64(width*8)))
	if ratio > 0 && units == 0 {
		units = 1
	}
	full := min(width, units/8)
	remainder := units % 8
	var bar strings.Builder
	bar.WriteString(strings.Repeat("█", full))
	cells := full
	if remainder > 0 && cells < width {
		bar.WriteRune(horizontalFractions[remainder])
		cells++
	}
	return bar.String(), cells
}

func terminalLabelWidth(labels []string, minimum, maximum int) int {
	width := minimum
	for _, label := range labels {
		width = max(width, ansi.StringWidth(terminalSafeText(label)))
	}
	return min(width, max(minimum, maximum))
}

func terminalValueWidth(series []compiledSeries) int {
	width := 1
	for _, item := range series {
		for _, value := range item.values {
			if !isMissing(value) {
				width = max(width, ansi.StringWidth(formatValue(value)))
			}
		}
	}
	return width
}

func (chart *Chart) terminalDivergingBars(width, height int) (string, error) {
	rowCount := len(chart.labels) * len(chart.series)
	if rowCount > height {
		return "", fmt.Errorf("terminal is too short for %d signed bars; increase the chart height", rowCount)
	}
	labels := make([]string, 0, rowCount)
	minimum, maximum := 0.0, 0.0
	for pointIndex, label := range chart.labels {
		for _, series := range chart.series {
			labels = append(labels, terminalSafeText(label)+" · "+terminalSafeText(series.spec.Label))
			value := series.values[pointIndex]
			if !isMissing(value) {
				minimum = math.Min(minimum, value)
				maximum = math.Max(maximum, value)
			}
		}
	}
	labelWidth := terminalLabelWidth(labels, 6, min(22, width/3))
	valueWidth := terminalValueWidth(chart.series)
	plotWidth := width - labelWidth - valueWidth - 4
	if plotWidth < 5 {
		return "", fmt.Errorf("terminal is too narrow for signed bar labels; increase the width")
	}
	rangeWidth := maximum - minimum
	negativeWidth := 0
	if minimum < 0 && rangeWidth > 0 {
		negativeWidth = max(1, int(math.Round(-minimum/rangeWidth*float64(plotWidth-1))))
	}
	positiveWidth := max(0, plotWidth-1-negativeWidth)

	rows := make([]string, 0, rowCount)
	rowIndex := 0
	for pointIndex := range chart.labels {
		for _, series := range chart.series {
			value := series.values[pointIndex]
			valueText := "–"
			negativeBar := strings.Repeat(" ", negativeWidth)
			positiveBar := strings.Repeat(" ", positiveWidth)
			style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
			if !isMissing(value) {
				valueText = formatValue(value)
				if value < 0 && negativeWidth > 0 {
					bar, cells := horizontalBar(value/minimum, negativeWidth)
					negativeBar = strings.Repeat(" ", negativeWidth-cells) + style.Render(bar)
				} else if value > 0 && positiveWidth > 0 {
					positiveBar = renderHorizontalBar(value/maximum, positiveWidth, style)
				}
			}
			line := padTerminal(labels[rowIndex], labelWidth) + " " + negativeBar + "│" + positiveBar + " " + padTerminalLeft(valueText, valueWidth)
			rows = append(rows, ansi.Truncate(line, width, "…"))
			rowIndex++
		}
	}
	return strings.Join(rows, "\n"), nil
}

func padTerminal(value string, width int) string {
	value = ansi.Truncate(value, width, "…")
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

func padTerminalLeft(value string, width int) string {
	value = ansi.Truncate(value, width, "…")
	return strings.Repeat(" ", max(0, width-ansi.StringWidth(value))) + value
}
