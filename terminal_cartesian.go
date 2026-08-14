package chartmux

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type terminalRenderState struct {
	inspect     bool
	focusIndex  int
	focusSeries int
}

func (chart *Chart) terminalState(options TerminalOptions) terminalRenderState {
	state := terminalRenderState{inspect: options.Inspect}
	if !state.inspect {
		return state
	}
	state.focusIndex = min(max(0, options.FocusIndex), max(0, len(chart.labels)-1))
	state.focusSeries = min(max(0, options.FocusSeries), max(0, len(chart.series)-1))
	return state
}

func (chart *Chart) terminalCartesianFrame(width, height int, include map[int]bool, zeroBaseline bool, state terminalRenderState) (string, error) {
	displayed := chart.terminalCartesianValues()
	minimum, maximum := valueRangeForSeries(displayed, include)
	anchorAtZero := false
	if chart.spec.Type == Area || zeroBaseline {
		anchorAtZero = minimum >= 0
		minimum = math.Min(0, minimum)
		maximum = math.Max(0, maximum)
	}
	if minimum == maximum {
		minimum--
		maximum++
	}
	padding := (maximum - minimum) * 0.04
	if !anchorAtZero {
		minimum -= padding
	}
	maximum += padding
	percent := chart.spec.Layout == Normalized
	scene, err := newTerminalCartesianScene(width, height, minimum, maximum, displayValue(chart.spec.Axes, true), percent)
	if err != nil {
		return "", err
	}

	if chart.spec.Type == Scatter {
		minimumX, maximumX := chart.scatterXRange()
		scene.paintNumericLabels(minimumX, maximumX)
		chart.paintTerminalInspectionCrosshair(scene, state, minimumX, maximumX)
		chart.paintTerminalScatter(scene, displayed, include, state, minimumX, maximumX)
	} else {
		scene.paintCategoryLabels(chart.labels, false)
		chart.paintTerminalInspectionCrosshair(scene, state, 0, float64(max(1, len(chart.labels)-1)))
		if chart.spec.Type == Area {
			chart.paintTerminalAreas(scene, displayed, include, state)
		}
		chart.paintTerminalLines(scene, displayed, include, state)
	}
	chart.paintTerminalCartesianAnnotations(scene, displayed)
	return scene.frame.render(), nil
}

func (chart *Chart) scatterXRange() (float64, float64) {
	if len(chart.xValues) == 0 {
		return 0, 1
	}
	minimum, maximum := chart.xValues[0], chart.xValues[0]
	for _, value := range chart.xValues[1:] {
		minimum = math.Min(minimum, value)
		maximum = math.Max(maximum, value)
	}
	if minimum == maximum {
		minimum--
		maximum++
	}
	padding := (maximum - minimum) * 0.03
	return minimum - padding, maximum + padding
}

func (chart *Chart) paintTerminalInspectionCrosshair(scene *terminalCartesianScene, state terminalRenderState, minimumX, maximumX float64) {
	if !state.inspect || len(chart.labels) == 0 {
		return
	}
	x := scene.pointX(state.focusIndex, len(chart.labels))
	if chart.spec.Type == Scatter && state.focusIndex < len(chart.xValues) {
		x = scene.numericX(chart.xValues[state.focusIndex], minimumX, maximumX)
	}
	style := terminalPaintStyle{color: terminalAxisColor, priority: terminalLayerCrosshair, faint: true}
	scene.frame.paintVertical(x, scene.plot.top, scene.plot.bottom-1, '┊', style)
}

func (chart *Chart) paintTerminalScatter(scene *terminalCartesianScene, displayed [][]float64, include map[int]bool, state terminalRenderState, minimumX, maximumX float64) {
	for seriesIndex, values := range displayed {
		if include != nil && !include[seriesIndex] {
			continue
		}
		for pointIndex, value := range values {
			if isMissing(value) || pointIndex >= len(chart.xValues) {
				continue
			}
			style := chart.terminalSeriesStyle(seriesIndex, terminalLayerPoint, state)
			if state.inspect && pointIndex == state.focusIndex && seriesIndex == state.focusSeries {
				style.bold = true
				style.reverse = true
			}
			scene.frame.paint(
				scene.numericX(chart.xValues[pointIndex], minimumX, maximumX),
				scene.y(value),
				terminalMarker(seriesIndex),
				style,
			)
		}
	}
}

func (chart *Chart) paintTerminalLines(scene *terminalCartesianScene, displayed [][]float64, include map[int]bool, state terminalRenderState) {
	for seriesIndex, values := range displayed {
		if include != nil && !include[seriesIndex] {
			continue
		}
		lineStyle := chart.terminalSeriesStyle(seriesIndex, terminalLayerLine, state)
		lineChar := '·'
		if chart.spec.Curve == Smooth {
			lineChar = '•'
		}
		for pointIndex := 1; pointIndex < len(values); pointIndex++ {
			left, right := values[pointIndex-1], values[pointIndex]
			if isMissing(left) || isMissing(right) {
				continue
			}
			chart.paintTerminalCurveSegment(
				scene,
				scene.pointX(pointIndex-1, len(values)), left,
				scene.pointX(pointIndex, len(values)), right,
				lineChar,
				lineStyle,
			)
		}
		for pointIndex, value := range values {
			if isMissing(value) {
				continue
			}
			style := chart.terminalSeriesStyle(seriesIndex, terminalLayerPoint, state)
			if state.inspect && pointIndex == state.focusIndex && seriesIndex == state.focusSeries {
				style.bold = true
				style.reverse = true
			}
			scene.frame.paint(scene.pointX(pointIndex, len(values)), scene.y(value), terminalMarker(seriesIndex), style)
		}
	}
}

func (chart *Chart) paintTerminalCurveSegment(scene *terminalCartesianScene, x0 int, value0 float64, x1 int, value1 float64, char rune, style terminalPaintStyle) {
	span := max(1, x1-x0)
	previous := terminalPoint{x: x0, y: scene.y(value0)}
	for x := x0 + 1; x <= x1; x++ {
		ratio := float64(x-x0) / float64(span)
		if chart.spec.Curve == Smooth {
			ratio = ratio * ratio * (3 - 2*ratio)
		}
		value := value0 + (value1-value0)*ratio
		current := terminalPoint{x: x, y: scene.y(value)}
		scene.frame.paintLine(previous, current, char, style)
		previous = current
	}
}

func (chart *Chart) paintTerminalAreas(scene *terminalCartesianScene, displayed [][]float64, include map[int]bool, state terminalRenderState) {
	stacked := chart.spec.Layout == Stacked || chart.spec.Layout == Normalized
	for seriesIndex, upperValues := range displayed {
		if include != nil && !include[seriesIndex] {
			continue
		}
		style := chart.terminalSeriesStyle(seriesIndex, terminalLayerFill, state)
		for pointIndex := 1; pointIndex < len(upperValues); pointIndex++ {
			upperLeft, upperRight := upperValues[pointIndex-1], upperValues[pointIndex]
			if isMissing(upperLeft) || isMissing(upperRight) {
				continue
			}
			lowerLeft, lowerRight := 0.0, 0.0
			if stacked && seriesIndex > 0 {
				lowerLeft = displayed[seriesIndex-1][pointIndex-1]
				lowerRight = displayed[seriesIndex-1][pointIndex]
				if isMissing(lowerLeft) {
					lowerLeft = 0
				}
				if isMissing(lowerRight) {
					lowerRight = 0
				}
			}
			x0 := scene.pointX(pointIndex-1, len(upperValues))
			x1 := scene.pointX(pointIndex, len(upperValues))
			span := max(1, x1-x0)
			for x := x0; x <= x1; x++ {
				ratio := float64(x-x0) / float64(span)
				if chart.spec.Curve == Smooth {
					ratio = ratio * ratio * (3 - 2*ratio)
				}
				upper := upperLeft + (upperRight-upperLeft)*ratio
				lower := lowerLeft + (lowerRight-lowerLeft)*ratio
				top := min(scene.y(upper), scene.y(lower))
				bottom := max(scene.y(upper), scene.y(lower))
				for y := top + 1; y < bottom; y++ {
					if (x+y+seriesIndex)%2 != 0 {
						continue
					}
					scene.frame.paint(x, y, terminalAreaGlyph(seriesIndex, x, y), style)
				}
			}
		}
	}
}

func (chart *Chart) terminalSeriesStyle(seriesIndex, priority int, state terminalRenderState) terminalPaintStyle {
	style := terminalPaintStyle{
		color:    chart.series[seriesIndex].spec.Color,
		priority: priority,
	}
	if state.inspect && seriesIndex != state.focusSeries {
		style.faint = true
	}
	return style
}

func (chart *Chart) paintTerminalCartesianAnnotations(scene *terminalCartesianScene, displayed [][]float64) {
	for _, annotation := range chart.spec.Annotations {
		if annotation.DataIndex == nil || *annotation.DataIndex < 0 || *annotation.DataIndex >= len(chart.labels) {
			continue
		}
		seriesIndex := chart.annotationSeriesIndex(annotation)
		if seriesIndex < 0 || seriesIndex >= len(displayed) {
			continue
		}
		value := displayed[seriesIndex][*annotation.DataIndex]
		if isMissing(value) {
			continue
		}
		color := annotation.Color
		if color == "" {
			color = chart.series[seriesIndex].spec.Color
		}
		x := scene.pointX(*annotation.DataIndex, len(chart.labels))
		if chart.spec.Type == Scatter && *annotation.DataIndex < len(chart.xValues) {
			minimumX, maximumX := chart.scatterXRange()
			x = scene.numericX(chart.xValues[*annotation.DataIndex], minimumX, maximumX)
		}
		scene.frame.paint(x, scene.y(value), '✦', terminalPaintStyle{
			color: color, priority: terminalLayerAnnotation, bold: true,
		})
	}
}

func (chart *Chart) annotationSeriesIndex(annotation Annotation) int {
	if annotation.Series == "" {
		return 0
	}
	for index, series := range chart.series {
		if series.spec.DataKey == annotation.Series {
			return index
		}
	}
	return -1
}

func (chart *Chart) terminalComboFrame(width, height int, state terminalRenderState) (string, error) {
	minimum, maximum := chart.valueRange(nil)
	minimum = math.Min(0, minimum)
	maximum = math.Max(0, maximum)
	if minimum == maximum {
		maximum++
	}
	padding := (maximum - minimum) * 0.04
	if minimum < 0 {
		minimum -= padding
	}
	maximum += padding
	scene, err := newTerminalCartesianScene(width, height, minimum, maximum, displayValue(chart.spec.Axes, true), false)
	if err != nil {
		return "", err
	}
	scene.paintCategoryLabels(chart.labels, true)
	chart.paintTerminalInspectionCrosshair(scene, state, 0, float64(max(1, len(chart.labels)-1)))

	barSeries := make([]int, 0, len(chart.series))
	lineInclude := make(map[int]bool, len(chart.series))
	for index, series := range chart.series {
		if series.spec.Mark == MarkBar {
			barSeries = append(barSeries, index)
		} else if series.spec.Mark == MarkLine {
			lineInclude[index] = true
		}
	}
	zeroY := scene.y(0)
	for pointIndex := range chart.labels {
		band := scene.band(pointIndex, len(chart.labels))
		available := max(1, band.width()-2)
		gap := 1
		if available < len(barSeries)*2-1 {
			gap = 0
		}
		barWidth := max(1, min(3, (available-gap*max(0, len(barSeries)-1))/max(1, len(barSeries))))
		groupWidth := barWidth*len(barSeries) + gap*max(0, len(barSeries)-1)
		startX := band.left + max(0, (band.width()-groupWidth)/2)
		for offset, seriesIndex := range barSeries {
			value := chart.series[seriesIndex].values[pointIndex]
			if isMissing(value) {
				continue
			}
			x0 := startX + offset*(barWidth+gap)
			chart.paintTerminalBarRect(scene, x0, x0+barWidth-1, zeroY, scene.y(value), seriesIndex, state, pointIndex)
		}
	}
	chart.paintTerminalLines(scene, chart.seriesValues(), lineInclude, state)
	chart.paintTerminalCartesianAnnotations(scene, chart.seriesValues())
	return scene.frame.render(), nil
}

func (chart *Chart) terminalInspection(state terminalRenderState, width int) string {
	if !state.inspect || len(chart.labels) == 0 || len(chart.series) == 0 {
		return ""
	}
	index := min(max(0, state.focusIndex), len(chart.labels)-1)
	seriesIndex := min(max(0, state.focusSeries), len(chart.series)-1)
	borderStyle := lipgloss.NewStyle().Foreground(terminalColor(terminalAxisColor)).Faint(true)
	headingStyle := lipgloss.NewStyle().Bold(true)
	innerWidth := max(1, width-2)
	heading := ansi.Truncate(terminalSafeText(chart.labels[index]), innerWidth-2, "…")
	topTail := max(1, innerWidth-ansi.StringWidth(heading)-2)
	rows := []string{borderStyle.Render("┌─") + headingStyle.Render(heading) + borderStyle.Render(strings.Repeat("─", topTail)+"┐")}

	items := make([]string, 0, len(chart.series))
	for currentSeries, series := range chart.series {
		if index >= len(series.values) {
			continue
		}
		value := "–"
		if !isMissing(series.values[index]) {
			value = formatValue(series.values[index])
		}
		markerStyle := lipgloss.NewStyle().Foreground(terminalColor(series.spec.Color))
		if currentSeries == seriesIndex {
			markerStyle = markerStyle.Bold(true)
		}
		items = append(items, markerStyle.Render(string(terminalMarker(currentSeries)))+" "+terminalSafeText(series.spec.Label)+" "+value)
	}
	contentRows := wrapTerminalInspectionItems(items, max(1, innerWidth-2))
	for _, row := range contentRows {
		rows = append(rows, borderStyle.Render("│ ")+padTerminal(row, max(1, innerWidth-2))+borderStyle.Render(" │"))
	}
	rows = append(rows, borderStyle.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
	return strings.Join(rows, "\n")
}

func wrapTerminalInspectionItems(items []string, width int) []string {
	if len(items) == 0 {
		return []string{"No data"}
	}
	rows := make([]string, 0, len(items))
	current := ""
	for _, item := range items {
		candidate := item
		if current != "" {
			candidate = current + "   " + item
		}
		if ansi.StringWidth(candidate) <= width {
			current = candidate
			continue
		}
		if current != "" {
			rows = append(rows, current)
		}
		current = ansi.Truncate(item, width, "…")
	}
	if current != "" {
		rows = append(rows, current)
	}
	return rows
}

// PointCount returns the number of selectable data points in the chart.
func (chart *Chart) PointCount() int {
	return len(chart.labels)
}

// SeriesCount returns the number of selectable data series in the chart.
func (chart *Chart) SeriesCount() int {
	return len(chart.series)
}
