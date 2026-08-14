package chartmux

import (
	"fmt"
	"math"
	"strings"
)

func (chart *Chart) terminalProportionsWithState(width, height int, state terminalRenderState) (string, error) {
	values := chart.series[0].values
	if height < len(values)+1 {
		return "", fmt.Errorf("terminal is too short for %d categories; increase the chart height", len(values))
	}
	total := 0.0
	for _, value := range values {
		if !isMissing(value) && value > 0 {
			total += value
		}
	}
	shapeHeight := height - len(values) - 1
	minimumHeight := 6
	innerRadius := 0.0
	if chart.spec.Type == Donut {
		minimumHeight = 4
		innerRadius = 0.48
	}
	if shapeHeight < minimumHeight || width < 24 {
		band := chart.terminalSegmentBand(values, total, width)
		if total <= 0 {
			band = strings.Repeat(" ", max(0, (width-len("No data"))/2)) + "No data"
		}
		rows := []string{band}
		if chart.spec.Type == Donut && total > 0 {
			rows = append(rows, padTerminal("Total "+formatValue(total), width))
		}
		rows = append(rows, chart.terminalProportionLegendRows(values, total, width)...)
		return strings.Join(rows, "\n"), nil
	}
	shapeHeight = min(9, shapeHeight)
	shapeWidth := min(36, width-2)
	frame := newTerminalFrame(shapeWidth, shapeHeight)
	centerX := float64(shapeWidth-1) / 2
	centerY := float64(shapeHeight-1) / 2
	radiusX := max(1.0, centerX)
	radiusY := max(1.0, centerY)
	cumulative := make([]float64, len(values))
	running := 0.0
	for index, value := range values {
		if !isMissing(value) && value > 0 {
			running += value
		}
		cumulative[index] = running
	}
	for y := 0; y < shapeHeight; y++ {
		for x := 0; x < shapeWidth; x++ {
			dx := (float64(x) - centerX) / radiusX
			dy := (float64(y) - centerY) / radiusY
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance > 1.04 || distance < innerRadius {
				continue
			}
			if total <= 0 {
				if distance >= 0.86 || (chart.spec.Type == Donut && distance <= innerRadius+0.14) {
					frame.paint(x, y, '·', terminalPaintStyle{
						color: terminalGridColor, priority: terminalLayerGrid, faint: true,
					})
				}
				continue
			}
			angle := math.Atan2(dx, -dy)
			if angle < 0 {
				angle += 2 * math.Pi
			}
			target := angle / (2 * math.Pi) * total
			seriesIndex := 0
			for seriesIndex < len(cumulative)-1 && target > cumulative[seriesIndex] {
				seriesIndex++
			}
			style := terminalPaintStyle{
				color: defaultColors[seriesIndex%len(defaultColors)], priority: terminalLayerMark,
			}
			if state.inspect && seriesIndex != state.focusIndex {
				style.faint = true
			}
			if state.inspect && seriesIndex == state.focusIndex {
				style.bold = true
			}
			frame.paint(x, y, terminalBarGlyph(seriesIndex, x, y), style)
		}
	}
	if total <= 0 {
		frame.paintAlignedText(0, shapeHeight/2, shapeWidth, "No data", 0, terminalPaintStyle{
			priority: terminalLayerLabel, bold: true,
		})
	} else if chart.spec.Type == Donut {
		centerText := "Total –"
		if total > 0 {
			centerText = "Total " + formatValue(total)
		}
		frame.paintAlignedText(0, shapeHeight/2, shapeWidth, centerText, 0, terminalPaintStyle{
			priority: terminalLayerLabel, bold: true,
		})
	}
	chart.paintTerminalProportionAnnotation(frame, values, total, centerX, centerY, radiusX, radiusY)

	rows := strings.Split(frame.render(), "\n")
	leftPad := strings.Repeat(" ", max(0, (width-shapeWidth)/2))
	for index := range rows {
		rows[index] = leftPad + rows[index]
	}
	rows = append(rows, "")
	rows = append(rows, chart.terminalProportionLegendRows(values, total, width)...)
	return strings.Join(rows, "\n"), nil
}

func (chart *Chart) paintTerminalProportionAnnotation(frame *terminalFrame, values []float64, total, centerX, centerY, radiusX, radiusY float64) {
	if total <= 0 {
		return
	}
	for _, annotation := range chart.spec.Annotations {
		if annotation.DataIndex == nil || *annotation.DataIndex < 0 || *annotation.DataIndex >= len(values) {
			continue
		}
		index := *annotation.DataIndex
		start := 0.0
		for valueIndex := 0; valueIndex < index; valueIndex++ {
			if !isMissing(values[valueIndex]) && values[valueIndex] > 0 {
				start += values[valueIndex]
			}
		}
		value := values[index]
		if isMissing(value) || value <= 0 {
			continue
		}
		angle := (start + value/2) / total * 2 * math.Pi
		x := int(math.Round(centerX + math.Sin(angle)*radiusX*0.9))
		y := int(math.Round(centerY - math.Cos(angle)*radiusY*0.9))
		color := annotation.Color
		if color == "" {
			color = defaultColors[index%len(defaultColors)]
		}
		frame.paint(x, y, '✦', terminalPaintStyle{
			color: color, priority: terminalLayerAnnotation, bold: true,
		})
	}
}
