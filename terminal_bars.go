package chartmux

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var terminalBarPatterns = [...]rune{'█', '▓', '▒', '▦', '▩', '▧'}

type terminalBarSegment struct {
	series int
	value  float64
}

type terminalBarRow struct {
	point    int
	series   int
	label    string
	segments []terminalBarSegment
}

func terminalBarPattern(index int) rune {
	return terminalBarPatterns[index%len(terminalBarPatterns)]
}

func (chart *Chart) terminalBarsWithState(width, height int, state terminalRenderState) (string, error) {
	for _, series := range chart.series {
		for _, value := range series.values {
			if !isMissing(value) && value < 0 {
				return chart.terminalDivergingBars(width, height, state)
			}
		}
	}
	if chart.spec.Orientation == Horizontal {
		return chart.terminalHorizontalBarsWithState(width, height, state)
	}
	return chart.terminalVerticalBarsWithState(width, height, state)
}

func (chart *Chart) terminalVerticalBarsWithState(width, height int, state terminalRenderState) (string, error) {
	maximum := chart.terminalBarMaximum()
	percent := chart.spec.Layout == Normalized
	scene, err := newTerminalCartesianScene(width, height, 0, maximum, displayValue(chart.spec.Axes, true), percent)
	if err != nil {
		return "", err
	}
	scene.paintCategoryLabels(chart.labels, true)
	if state.inspect && len(chart.labels) > 0 {
		band := scene.band(state.focusIndex, len(chart.labels))
		x := band.left + band.width()/2
		scene.frame.paintVertical(x, scene.plot.top, scene.plot.bottom-1, '┊', terminalPaintStyle{
			color: terminalAxisColor, priority: terminalLayerCrosshair, faint: true,
		})
	}

	stacked := chart.spec.Layout == Stacked || chart.spec.Layout == Normalized
	zeroY := scene.y(0)
	for pointIndex := range chart.labels {
		band := scene.band(pointIndex, len(chart.labels))
		if stacked {
			chart.paintTerminalStackedBar(scene, band, pointIndex, zeroY, state)
			continue
		}
		if err := chart.paintTerminalGroupedBar(scene, band, pointIndex, zeroY, state); err != nil {
			return "", err
		}
	}
	return scene.frame.render(), nil
}

func (chart *Chart) paintTerminalStackedBar(scene *terminalCartesianScene, band terminalRect, pointIndex, zeroY int, state terminalRenderState) {
	innerWidth := max(1, band.width()-2)
	barWidth := min(7, max(2, int(math.Round(float64(innerWidth)*0.72))))
	barWidth = min(barWidth, innerWidth)
	x0 := band.left + max(0, (band.width()-barWidth)/2)
	cumulative := 0.0
	for seriesIndex, series := range chart.series {
		value := series.values[pointIndex]
		if isMissing(value) || value <= 0 {
			continue
		}
		cumulative += value
		capY := scene.y(cumulative)
		chart.paintTerminalBarRect(scene, x0, x0+barWidth-1, zeroY, capY, seriesIndex, state, pointIndex)
		chart.paintTerminalBarAnnotation(scene, pointIndex, seriesIndex, x0+barWidth/2, capY)
		zeroY = capY
	}
}

func (chart *Chart) paintTerminalGroupedBar(scene *terminalCartesianScene, band terminalRect, pointIndex, zeroY int, state terminalRenderState) error {
	seriesCount := len(chart.series)
	innerWidth := max(1, band.width()-2)
	gap := 1
	if innerWidth < seriesCount*2-1 {
		gap = 0
	}
	barWidth := (innerWidth - gap*max(0, seriesCount-1)) / max(1, seriesCount)
	barWidth = min(3, barWidth)
	if barWidth < 1 {
		return fmt.Errorf("terminal is too narrow for %d grouped series; increase the width", seriesCount)
	}
	groupWidth := barWidth*seriesCount + gap*max(0, seriesCount-1)
	startX := band.left + max(0, (band.width()-groupWidth)/2)
	for seriesIndex, series := range chart.series {
		value := series.values[pointIndex]
		if isMissing(value) || value <= 0 {
			continue
		}
		x0 := startX + seriesIndex*(barWidth+gap)
		capY := scene.y(value)
		chart.paintTerminalBarRect(scene, x0, x0+barWidth-1, zeroY, capY, seriesIndex, state, pointIndex)
		chart.paintTerminalBarAnnotation(scene, pointIndex, seriesIndex, x0+barWidth/2, capY)
	}
	return nil
}

func (chart *Chart) paintTerminalBarRect(scene *terminalCartesianScene, x0, x1, zeroY, valueY, seriesIndex int, state terminalRenderState, pointIndex int) {
	style := chart.terminalSeriesStyle(seriesIndex, terminalLayerMark, state)
	if state.inspect && pointIndex != state.focusIndex {
		style.faint = true
	}
	capStyle := style
	capStyle.priority = terminalLayerLine
	capStyle.bold = state.inspect && pointIndex == state.focusIndex && seriesIndex == state.focusSeries
	if valueY <= zeroY {
		for y := valueY + 1; y < zeroY; y++ {
			for x := x0; x <= x1; x++ {
				scene.frame.paint(x, y, terminalBarGlyph(seriesIndex, x, y), style)
			}
		}
		for x := x0; x <= x1; x++ {
			scene.frame.paint(x, valueY, '▄', capStyle)
		}
		return
	}
	for y := zeroY + 1; y < valueY; y++ {
		for x := x0; x <= x1; x++ {
			scene.frame.paint(x, y, terminalBarGlyph(seriesIndex, x, y), style)
		}
	}
	for x := x0; x <= x1; x++ {
		scene.frame.paint(x, valueY, '▀', capStyle)
	}
}

func (chart *Chart) paintTerminalBarAnnotation(scene *terminalCartesianScene, pointIndex, seriesIndex, x, capY int) {
	for _, annotation := range chart.spec.Annotations {
		if annotation.DataIndex == nil || *annotation.DataIndex != pointIndex {
			continue
		}
		annotationSeries := chart.annotationSeriesIndex(annotation)
		if annotationSeries != seriesIndex {
			continue
		}
		color := annotation.Color
		if color == "" {
			color = chart.series[seriesIndex].spec.Color
		}
		y := max(scene.plot.top, capY-1)
		scene.frame.paint(x, y, '✦', terminalPaintStyle{
			color: color, priority: terminalLayerAnnotation, bold: true,
		})
	}
}

func (chart *Chart) terminalHorizontalBarsWithState(width, height int, state terminalRenderState) (string, error) {
	showAxes := displayValue(chart.spec.Axes, true)
	stacked := chart.spec.Layout == Stacked || chart.spec.Layout == Normalized
	barRows := make([]terminalBarRow, 0, len(chart.labels)*len(chart.series))
	for pointIndex, label := range chart.labels {
		label = terminalSafeText(label)
		if stacked {
			row := terminalBarRow{point: pointIndex, series: -1, label: label}
			for seriesIndex, series := range chart.series {
				row.segments = append(row.segments, terminalBarSegment{series: seriesIndex, value: series.values[pointIndex]})
			}
			barRows = append(barRows, row)
			continue
		}
		for seriesIndex, series := range chart.series {
			barRows = append(barRows, terminalBarRow{
				point: pointIndex, series: seriesIndex,
				label:    label + " · " + terminalSafeText(series.spec.Label),
				segments: []terminalBarSegment{{series: seriesIndex, value: series.values[pointIndex]}},
			})
		}
	}
	showScale := showAxes && len(barRows)+1 <= height
	if len(barRows) > height {
		return "", fmt.Errorf("terminal is too short for %d horizontal bars; increase the chart height", len(barRows))
	}
	labelWidth := 0
	plotWidth := width
	if showAxes {
		labels := make([]string, len(barRows))
		for index := range barRows {
			labels[index] = barRows[index].label
		}
		labelWidth = terminalLabelWidth(labels, 6, min(28, width/3))
		plotWidth -= labelWidth + 3
	}
	if plotWidth < 8 {
		return "", fmt.Errorf("terminal is too narrow for horizontal bars; increase the width")
	}
	maximum := chart.terminalBarMaximum()
	rows := make([]string, 0, len(barRows)+1)
	if showScale {
		maximumText := chart.terminalBarAxisValue(maximum)
		middleText := chart.terminalBarAxisValue(maximum / 2)
		track := make([]rune, plotWidth)
		for index := range track {
			track[index] = terminalTrackGlyph(index)
		}
		copy(track, []rune("0"))
		middleStart := max(1, plotWidth/2-len([]rune(middleText))/2)
		copy(track[middleStart:], []rune(middleText))
		maximumStart := max(1, plotWidth-len([]rune(maximumText)))
		copy(track[maximumStart:], []rune(maximumText))
		rows = append(rows, strings.Repeat(" ", labelWidth+3)+lipgloss.NewStyle().Foreground(terminalColor(terminalTextColor)).Faint(true).Render(string(track)))
	}
	for _, row := range barRows {
		prefix := ""
		if showAxes {
			prefix = padTerminal(row.label, labelWidth) + " │ "
		}
		bar := chart.renderTerminalHorizontalSegments(row, maximum, plotWidth, state)
		rows = append(rows, ansi.Truncate(prefix+bar, width, "…"))
	}
	return strings.Join(rows, "\n"), nil
}

func (chart *Chart) renderTerminalHorizontalSegments(row terminalBarRow, maximum float64, width int, state terminalRenderState) string {
	parts := make([]string, 0, len(row.segments)+1)
	previousCells := 0
	cumulative := 0.0
	for _, segment := range row.segments {
		if isMissing(segment.value) || segment.value <= 0 {
			continue
		}
		cumulative += segment.value
		endCells := min(width, max(previousCells, int(math.Round(cumulative/maximum*float64(width)))))
		if endCells <= previousCells {
			continue
		}
		style := lipgloss.NewStyle().Foreground(terminalColor(chart.series[segment.series].spec.Color))
		if state.inspect && (row.point != state.focusIndex || (row.series >= 0 && segment.series != state.focusSeries)) {
			style = style.Faint(true)
		}
		if state.inspect && row.point == state.focusIndex && (row.series < 0 || segment.series == state.focusSeries) {
			style = style.Bold(true)
		}
		var run strings.Builder
		for x := previousCells; x < endCells; x++ {
			run.WriteRune(terminalBarGlyph(segment.series, x, row.point))
		}
		parts = append(parts, style.Render(run.String()))
		previousCells = endCells
	}
	if previousCells < width {
		var run strings.Builder
		for x := previousCells; x < width; x++ {
			run.WriteRune(terminalTrackGlyph(x))
		}
		track := lipgloss.NewStyle().Foreground(terminalColor(terminalGridColor)).Faint(true).Render(run.String())
		parts = append(parts, track)
	}
	return strings.Join(parts, "")
}

func terminalTrackGlyph(index int) rune {
	if index%3 == 0 {
		return '┄'
	}
	return ' '
}

func (chart *Chart) terminalBarMaximum() float64 {
	if chart.spec.Layout == Normalized {
		return 100
	}
	stacked := chart.spec.Layout == Stacked
	maximum := 0.0
	for pointIndex := range chart.labels {
		total := 0.0
		for _, series := range chart.series {
			value := series.values[pointIndex]
			if isMissing(value) {
				continue
			}
			if stacked {
				total += value
			} else {
				maximum = math.Max(maximum, value)
			}
		}
		if stacked {
			maximum = math.Max(maximum, total)
		}
	}
	if maximum <= 0 {
		return 1
	}
	return maximum
}

func (chart *Chart) terminalBarAxisValue(value float64) string {
	if chart.spec.Layout == Normalized {
		return fmt.Sprintf("%.0f%%", value)
	}
	return formatValue(value)
}
