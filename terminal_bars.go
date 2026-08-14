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
	// Any negative value requires a shared zero axis. The diverging layout
	// intentionally supersedes horizontal and stacked presentation options,
	// which cannot represent mixed signs without misleading segment geometry.
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
	color, annotated := chart.terminalAnnotationColor(pointIndex, seriesIndex)
	if !annotated {
		return
	}
	y := max(scene.plot.top, capY-1)
	scene.frame.paint(x, y, '✦', terminalPaintStyle{
		color: color, priority: terminalLayerAnnotation, bold: true,
	})
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
		middleStart := max(1, plotWidth/2-len([]rune(middleText))/2)
		maximumStart := max(1, plotWidth-len([]rune(maximumText)))
		guide := terminalScaleGuide(plotWidth,
			terminalScaleLabel{start: maximumStart, text: maximumText},
			terminalScaleLabel{start: 0, text: "0"},
			terminalScaleLabel{start: middleStart, text: middleText},
		)
		rows = append(rows, strings.Repeat(" ", labelWidth+3)+lipgloss.NewStyle().Foreground(terminalColor(terminalTextColor)).Faint(true).Render(guide))
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
	cells := make([]terminalHorizontalCell, width)
	for x := range cells {
		cells[x] = terminalHorizontalCell{
			glyph: terminalTrackGlyph(x), color: terminalGridColor, faint: true,
		}
	}
	type annotationMarker struct {
		x     int
		color string
	}
	markers := make([]annotationMarker, 0, len(row.segments))
	previousCells := 0
	cumulative := 0.0
	for _, segment := range row.segments {
		if isMissing(segment.value) {
			continue
		}
		annotationColor, annotated := chart.terminalAnnotationColor(row.point, segment.series)
		if segment.value <= 0 {
			if segment.value == 0 && annotated {
				markers = append(markers, annotationMarker{x: min(previousCells, width-1), color: annotationColor})
			}
			continue
		}
		cumulative += segment.value
		endCells := min(width, max(previousCells, int(math.Round(cumulative/maximum*float64(width)))))
		if endCells <= previousCells {
			if annotated {
				markers = append(markers, annotationMarker{x: min(previousCells, width-1), color: annotationColor})
			}
			continue
		}
		paint := terminalHorizontalCell{color: chart.series[segment.series].spec.Color}
		if state.inspect && (row.point != state.focusIndex || (row.series >= 0 && segment.series != state.focusSeries)) {
			paint.faint = true
		}
		if state.inspect && row.point == state.focusIndex && (row.series < 0 || segment.series == state.focusSeries) {
			paint.bold = true
		}
		for x := previousCells; x < endCells; x++ {
			paint.glyph = terminalBarGlyph(segment.series, x, row.point)
			cells[x] = paint
		}
		if annotated {
			markers = append(markers, annotationMarker{x: endCells - 1, color: annotationColor})
		}
		previousCells = endCells
	}
	for _, marker := range markers {
		cells[marker.x] = terminalHorizontalCell{glyph: '✦', color: marker.color, bold: true}
	}
	return renderTerminalHorizontalCells(cells)
}

type terminalHorizontalCell struct {
	glyph rune
	color string
	bold  bool
	faint bool
}

func renderTerminalHorizontalCells(cells []terminalHorizontalCell) string {
	parts := make([]string, 0, len(cells))
	for start := 0; start < len(cells); {
		end := start + 1
		for end < len(cells) && cells[end].color == cells[start].color && cells[end].bold == cells[start].bold && cells[end].faint == cells[start].faint {
			end++
		}
		run := make([]rune, end-start)
		for index := start; index < end; index++ {
			run[index-start] = cells[index].glyph
		}
		style := lipgloss.NewStyle().Foreground(terminalColor(cells[start].color)).Bold(cells[start].bold).Faint(cells[start].faint)
		parts = append(parts, style.Render(string(run)))
		start = end
	}
	return strings.Join(parts, "")
}

func terminalTrackGlyph(index int) rune {
	if index%3 == 0 {
		return '┄'
	}
	return ' '
}

type terminalScaleLabel struct {
	start int
	text  string
}

func terminalScaleGuide(width int, labels ...terminalScaleLabel) string {
	track := make([]rune, max(0, width))
	occupied := make([]bool, len(track))
	for index := range track {
		track[index] = terminalTrackGlyph(index)
	}
	for _, label := range labels {
		runes := []rune(label.text)
		end := label.start + len(runes)
		if len(runes) == 0 || label.start < 0 || end > len(track) {
			continue
		}
		checkStart := max(0, label.start-1)
		checkEnd := min(len(track), end+1)
		collision := false
		for index := checkStart; index < checkEnd; index++ {
			if occupied[index] {
				collision = true
				break
			}
		}
		if collision {
			continue
		}
		copy(track[label.start:end], runes)
		for index := label.start; index < end; index++ {
			occupied[index] = true
		}
	}
	return string(track)
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
