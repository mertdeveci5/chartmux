package chartmux

import (
	"fmt"
	"math"

	"github.com/charmbracelet/x/ansi"
)

type terminalRadarPoint struct {
	terminalPoint
	valid bool
}

func (chart *Chart) terminalRadarWithState(width, height int, state terminalRenderState) (string, error) {
	if height < 10 {
		return "", fmt.Errorf("terminal is too short for 10 radar rows; increase the chart height")
	}
	frame := newTerminalFrame(width, height)
	showAxes := displayValue(chart.spec.Axes, true)
	center := terminalPoint{x: width / 2, y: height / 2}
	yRadius := max(2, (height-4)/2)
	longestLabel := 0
	for _, label := range chart.labels {
		longestLabel = max(longestLabel, ansi.StringWidth(terminalSafeText(label)))
	}
	labelReserve := min(14, max(5, longestLabel/2+2))
	xRadius := min(max(4, yRadius*3), max(4, width/2-labelReserve))
	if xRadius < 4 || yRadius < 2 {
		return "", fmt.Errorf("terminal is too narrow for radial radar geometry; increase the chart width")
	}

	gridStyle := terminalPaintStyle{color: terminalGridColor, priority: terminalLayerGrid, faint: true}
	axisStyle := terminalPaintStyle{color: terminalAxisColor, priority: terminalLayerGrid, faint: true}
	if showAxes {
		for _, fraction := range []float64{1} {
			chart.paintTerminalRadarRing(frame, center, xRadius, yRadius, fraction, gridStyle)
		}
		for metricIndex := range chart.labels {
			angle := terminalRadarAngle(metricIndex, len(chart.labels))
			endpoint := terminalPoint{
				x: center.x + int(math.Round(math.Cos(angle)*float64(xRadius))),
				y: center.y + int(math.Round(math.Sin(angle)*float64(yRadius))),
			}
			frame.paintLine(center, endpoint, '·', axisStyle)
		}
	}

	maximum := chart.spec.Max
	if maximum <= 0 {
		_, maximum = chart.valueRange(nil)
	}
	if maximum <= 0 {
		maximum = 1
	}
	seriesPoints := make([][]terminalRadarPoint, len(chart.series))
	for seriesIndex, series := range chart.series {
		points := make([]terminalRadarPoint, len(chart.labels))
		for metricIndex, value := range series.values {
			if isMissing(value) {
				points[metricIndex] = terminalRadarPoint{terminalPoint: center}
				continue
			}
			ratio := math.Max(0, math.Min(1, value/maximum))
			angle := terminalRadarAngle(metricIndex, len(chart.labels))
			points[metricIndex] = terminalRadarPoint{
				terminalPoint: terminalPoint{
					x: center.x + int(math.Round(math.Cos(angle)*float64(xRadius)*ratio)),
					y: center.y + int(math.Round(math.Sin(angle)*float64(yRadius)*ratio)),
				},
				valid: true,
			}
		}
		seriesPoints[seriesIndex] = points
	}

	for seriesIndex, points := range seriesPoints {
		lineStyle := chart.terminalSeriesStyle(seriesIndex, terminalLayerLine, state)
		lineChar := '•'
		if seriesIndex%2 == 1 {
			lineChar = '·'
		}
		for pointIndex, point := range points {
			next := points[(pointIndex+1)%len(points)]
			if point.valid && next.valid {
				frame.paintLine(point.terminalPoint, next.terminalPoint, lineChar, lineStyle)
			}
			if !point.valid {
				continue
			}
			pointStyle := chart.terminalSeriesStyle(seriesIndex, terminalLayerPoint, state)
			if state.inspect && pointIndex == state.focusIndex && seriesIndex == state.focusSeries {
				pointStyle.bold = true
				pointStyle.reverse = true
			}
			frame.paint(point.x, point.y, terminalMarker(seriesIndex), pointStyle)
		}
	}

	if state.inspect && state.focusIndex < len(chart.labels) {
		angle := terminalRadarAngle(state.focusIndex, len(chart.labels))
		endpoint := terminalPoint{
			x: center.x + int(math.Round(math.Cos(angle)*float64(xRadius))),
			y: center.y + int(math.Round(math.Sin(angle)*float64(yRadius))),
		}
		frame.paintLine(center, endpoint, '┊', terminalPaintStyle{
			color: terminalAxisColor, priority: terminalLayerCrosshair, faint: true,
		})
	}
	chart.paintTerminalRadarAnnotations(frame, seriesPoints)
	if showAxes {
		chart.paintTerminalRadarLabels(frame, center, xRadius, yRadius)
	}
	return frame.render(), nil
}

func terminalRadarAngle(index, count int) float64 {
	if count <= 0 {
		return -math.Pi / 2
	}
	return -math.Pi/2 + 2*math.Pi*float64(index)/float64(count)
}

func (chart *Chart) paintTerminalRadarRing(frame *terminalFrame, center terminalPoint, xRadius, yRadius int, fraction float64, style terminalPaintStyle) {
	points := make([]terminalPoint, len(chart.labels))
	for index := range points {
		angle := terminalRadarAngle(index, len(points))
		points[index] = terminalPoint{
			x: center.x + int(math.Round(math.Cos(angle)*float64(xRadius)*fraction)),
			y: center.y + int(math.Round(math.Sin(angle)*float64(yRadius)*fraction)),
		}
	}
	for index, point := range points {
		frame.paintLine(point, points[(index+1)%len(points)], '·', style)
	}
}

func (chart *Chart) paintTerminalRadarLabels(frame *terminalFrame, center terminalPoint, xRadius, yRadius int) {
	style := terminalPaintStyle{color: terminalTextColor, priority: terminalLayerLabel, faint: true}
	for index, raw := range chart.labels {
		angle := terminalRadarAngle(index, len(chart.labels))
		label := ansi.Truncate(terminalSafeText(raw), min(14, frame.width/3), "…")
		labelWidth := ansi.StringWidth(label)
		x := center.x + int(math.Round(math.Cos(angle)*float64(xRadius+2)))
		y := center.y + int(math.Round(math.Sin(angle)*float64(yRadius+1)))
		switch cosine := math.Cos(angle); {
		case cosine < -0.25:
			x -= labelWidth
		case cosine <= 0.25:
			x -= labelWidth / 2
		}
		x = max(0, min(frame.width-labelWidth, x))
		y = max(0, min(frame.height-1, y))
		frame.paintText(x, y, label, style)
	}
}

func (chart *Chart) paintTerminalRadarAnnotations(frame *terminalFrame, seriesPoints [][]terminalRadarPoint) {
	for _, annotation := range chart.spec.Annotations {
		if annotation.DataIndex == nil || *annotation.DataIndex < 0 || *annotation.DataIndex >= len(chart.labels) {
			continue
		}
		seriesIndex := chart.annotationSeriesIndex(annotation)
		if seriesIndex < 0 || seriesIndex >= len(seriesPoints) {
			continue
		}
		point := seriesPoints[seriesIndex][*annotation.DataIndex]
		if !point.valid {
			continue
		}
		color := annotation.Color
		if color == "" {
			color = chart.series[seriesIndex].spec.Color
		}
		frame.paint(point.x, point.y, '✦', terminalPaintStyle{
			color: color, priority: terminalLayerAnnotation, bold: true,
		})
	}
}
