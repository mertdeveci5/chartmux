package chartmux

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var horizontalFractions = [...]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}

func (chart *Chart) terminalProportionLegendRows(values []float64, total float64, width int) []string {
	rows := make([]string, 0, len(values))
	labelWidth := terminalLabelWidth(chart.labels, 8, min(20, width/3))
	for index, value := range values {
		style := lipgloss.NewStyle().Foreground(terminalColor(defaultColors[index%len(defaultColors)]))
		valueText := "–"
		percentText := "  0.0%"
		if !isMissing(value) {
			valueText = formatValue(value)
			if total > 0 {
				percentText = fmt.Sprintf("%5.1f%%", value/total*100)
			}
		}
		line := style.Render(string(terminalBarPattern(index))) + " " + padTerminal(terminalSafeText(chart.labels[index]), labelWidth) + "  " + valueText + "  " + percentText
		rows = append(rows, ansi.Truncate(line, width, "…"))
	}
	return rows
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
			band.WriteString(style.Render(strings.Repeat(string(terminalBarPattern(index)), cells)))
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
	return chart.terminalHeatmapWithState(width, height, terminalRenderState{})
}

func (chart *Chart) terminalHeatmapWithState(width, height int, state terminalRenderState) (string, error) {
	showAxes := displayValue(chart.spec.Axes, true)
	baseHeight := len(chart.labels)
	if showAxes {
		baseHeight++
	}
	if height < baseHeight {
		return "", fmt.Errorf("terminal is too short for %d heatmap rows; increase the chart height", len(chart.labels))
	}
	labelWidth := 0
	if showAxes {
		labelWidth = terminalLabelWidth(chart.labels, 4, min(14, width/4))
	}
	prefixWidth := 0
	if showAxes {
		prefixWidth = labelWidth + 3
	}
	gapWidth := max(0, len(chart.series)-1) * 2
	if width-prefixWidth-gapWidth < len(chart.series) {
		return "", fmt.Errorf("terminal is too narrow for %d heatmap series; increase the width", len(chart.series))
	}
	cellWidth := max(7, min(14, (width-prefixWidth-gapWidth)/max(1, len(chart.series))))

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

	rows := make([]string, 0, min(height, baseHeight+2))
	if showAxes {
		header := strings.Repeat(" ", labelWidth) + " │ "
		for index, series := range chart.series {
			if index > 0 {
				header += "  "
			}
			style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
			header += style.Render(padTerminal(terminalSafeText(series.spec.Label), cellWidth))
		}
		rows = append(rows, ansi.Truncate(header, width, "…"))
		if height >= baseHeight+1 {
			separator := strings.Repeat("─", labelWidth+1) + "┼" + strings.Repeat("─", max(1, min(width-labelWidth-2, len(chart.series)*(cellWidth+2))))
			rows = append(rows, lipgloss.NewStyle().Foreground(terminalColor(terminalAxisColor)).Faint(true).Render(ansi.Truncate(separator, width, "")))
		}
	}
	for pointIndex, label := range chart.labels {
		if len(rows) >= height {
			break
		}
		line := ""
		if showAxes {
			labelText := padTerminalLeft(terminalSafeText(label), labelWidth)
			if state.inspect && pointIndex == state.focusIndex {
				labelText = lipgloss.NewStyle().Bold(true).Render(labelText)
			}
			line = labelText + " │ "
		}
		for seriesIndex, series := range chart.series {
			if seriesIndex > 0 {
				line += "  "
			}
			focused := state.inspect && pointIndex == state.focusIndex && seriesIndex == state.focusSeries
			dimmed := state.inspect && !focused
			line += terminalHeatCell(series.values[pointIndex], maximum, cellWidth, series.spec.Color, focused, dimmed)
		}
		rows = append(rows, ansi.Truncate(line, width, "…"))
	}
	if len(rows) < height {
		key := "Low  ░▒▓█  High"
		if showAxes {
			key = strings.Repeat(" ", labelWidth) + " │ " + key
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(terminalColor(terminalAxisColor)).Faint(true).Render(key))
	}
	return strings.Join(rows, "\n"), nil
}

func terminalHeatCell(value, maximum float64, width int, color string, focused, dimmed bool) string {
	if isMissing(value) {
		style := lipgloss.NewStyle().Bold(focused).Faint(dimmed)
		return padTerminal(style.Render("· –"), width)
	}
	density := []rune{'░', '▒', '▓', '█'}
	index := min(len(density)-1, max(0, int(math.Round(value/maximum*float64(len(density)-1)))))
	valueText := formatValue(value)
	fillWidth := min(4, max(1, width-ansi.StringWidth(valueText)-1))
	style := lipgloss.NewStyle().Foreground(terminalColor(color)).Bold(focused).Faint(dimmed)
	cell := style.Render(strings.Repeat(string(density[index]), fillWidth) + " " + valueText)
	return padTerminal(cell, width)
}

func (chart *Chart) terminalFunnel(width, height int) (string, error) {
	return chart.terminalFunnelWithState(width, height, terminalRenderState{})
}

func (chart *Chart) terminalFunnelWithState(width, height int, state terminalRenderState) (string, error) {
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
	showConversions := height >= len(chart.labels)*2-1
	rowCapacity := len(chart.labels)
	if showConversions {
		rowCapacity = len(chart.labels)*2 - 1
	}
	rows := make([]string, 0, min(height, rowCapacity))
	for index, label := range chart.labels {
		if len(rows) >= height {
			break
		}
		value := chart.series[0].values[index]
		ratio := 0.0
		valueText := "–"
		if !isMissing(value) {
			ratio = value / maximum
			valueText = formatValue(value)
		}
		style := lipgloss.NewStyle().Foreground(terminalColor(chart.series[0].spec.Color))
		if state.inspect && index != state.focusIndex {
			style = style.Faint(true)
		}
		if state.inspect && index == state.focusIndex {
			style = style.Bold(true)
		}
		bar := renderCenteredBar(ratio, barWidth, style)
		line := ""
		if showAxes {
			labelText := padTerminal(terminalSafeText(label), labelWidth)
			if state.inspect && index == state.focusIndex {
				labelText = lipgloss.NewStyle().Bold(true).Render(labelText)
			}
			line = labelText + " "
		}
		line += bar
		if showValues {
			line += " " + padTerminalLeft(valueText, valueWidth)
		}
		rows = append(rows, ansi.Truncate(line, width, "…"))
		if showConversions && index+1 < len(chart.labels) {
			conversion := "↓   –"
			next := chart.series[0].values[index+1]
			if !isMissing(value) && !isMissing(next) && value > 0 {
				conversion = fmt.Sprintf("↓ %5.1f%%", next/value*100)
			}
			conversionLine := ""
			if showAxes {
				conversionLine = strings.Repeat(" ", labelWidth) + " "
			}
			conversionWidth := ansi.StringWidth(conversion)
			left := max(0, (barWidth-conversionWidth)/2)
			conversionLine += strings.Repeat(" ", left) + conversion + strings.Repeat(" ", max(0, barWidth-conversionWidth-left))
			if showValues {
				conversionLine += strings.Repeat(" ", valueWidth+1)
			}
			rows = append(rows, lipgloss.NewStyle().Foreground(terminalColor(terminalAxisColor)).Faint(true).Render(ansi.Truncate(conversionLine, width, "")))
		}
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

func (chart *Chart) terminalDivergingBars(width, height int, state terminalRenderState) (string, error) {
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

	showScale := displayValue(chart.spec.Axes, true) && rowCount+1 <= height
	rows := make([]string, 0, rowCount+1)
	if showScale {
		scale := make([]rune, plotWidth)
		for index := range scale {
			scale[index] = terminalTrackGlyph(index)
		}
		minimumText := formatValue(minimum)
		maximumText := formatValue(maximum)
		copy(scale, []rune(minimumText))
		zeroIndex := min(plotWidth-1, negativeWidth)
		copy(scale[zeroIndex:], []rune("0"))
		maximumStart := max(zeroIndex+1, plotWidth-len([]rune(maximumText)))
		copy(scale[maximumStart:], []rune(maximumText))
		prefix := strings.Repeat(" ", labelWidth+1)
		rows = append(rows, prefix+lipgloss.NewStyle().Foreground(terminalColor(terminalTextColor)).Faint(true).Render(string(scale)))
	}
	rowIndex := 0
	for pointIndex := range chart.labels {
		for seriesIndex, series := range chart.series {
			value := series.values[pointIndex]
			valueText := "–"
			negativeBar := terminalTrackRun(0, negativeWidth)
			positiveBar := terminalTrackRun(negativeWidth+1, positiveWidth)
			style := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
			if state.inspect && (pointIndex != state.focusIndex || seriesIndex != state.focusSeries) {
				style = style.Faint(true)
			}
			if state.inspect && pointIndex == state.focusIndex && seriesIndex == state.focusSeries {
				style = style.Bold(true)
			}
			if !isMissing(value) {
				valueText = formatValue(value)
				if value < 0 && negativeWidth > 0 {
					bar, cells := horizontalBar(value/minimum, negativeWidth)
					negativeBar = terminalTrackRun(0, negativeWidth-cells) + style.Render(bar)
				} else if value > 0 && positiveWidth > 0 {
					bar, cells := horizontalBar(value/maximum, positiveWidth)
					positiveBar = style.Render(bar) + terminalTrackRun(negativeWidth+1+cells, positiveWidth-cells)
				}
			}
			axisStyle := lipgloss.NewStyle().Foreground(terminalColor(terminalAxisColor)).Faint(true)
			line := padTerminal(labels[rowIndex], labelWidth) + " " + negativeBar + axisStyle.Render("│") + positiveBar + " " + padTerminalLeft(valueText, valueWidth)
			rows = append(rows, ansi.Truncate(line, width, "…"))
			rowIndex++
		}
	}
	return strings.Join(rows, "\n"), nil
}

func terminalTrackRun(start, width int) string {
	var run strings.Builder
	for index := 0; index < width; index++ {
		run.WriteRune(terminalTrackGlyph(start + index))
	}
	return lipgloss.NewStyle().Foreground(terminalColor(terminalGridColor)).Faint(true).Render(run.String())
}

func padTerminal(value string, width int) string {
	value = ansi.Truncate(value, width, "…")
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

func padTerminalLeft(value string, width int) string {
	value = ansi.Truncate(value, width, "…")
	return strings.Repeat(" ", max(0, width-ansi.StringWidth(value))) + value
}
