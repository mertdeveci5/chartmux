package chartmux

import (
	"fmt"
	"math"

	"github.com/go-analyze/charts"
)

type graphicalValueAxis struct {
	minimum float64
	maximum float64
	step    float64
}

func (chart *Chart) drawGraphicalSignedBars(painter *charts.Painter, theme charts.ColorPalette, showAxes, showLabels bool) error {
	minimum, maximum := chart.graphicalBarExtent()
	axis := newGraphicalValueAxis(minimum, maximum, showLabels)
	if chart.spec.Orientation == Horizontal {
		return chart.drawGraphicalSignedHorizontalBars(painter, theme, axis, showAxes, showLabels)
	}
	return chart.drawGraphicalSignedVerticalBars(painter, theme, axis, showAxes, showLabels)
}

func (chart *Chart) graphicalBarExtent() (float64, float64) {
	minimum, maximum := 0.0, 0.0
	stacked := chart.spec.Layout == Stacked
	if !stacked {
		for _, series := range chart.series {
			for _, value := range series.values {
				if isMissing(value) {
					continue
				}
				minimum = math.Min(minimum, value)
				maximum = math.Max(maximum, value)
			}
		}
		return minimum, maximum
	}
	for valueIndex := range chart.labels {
		positive, negative := 0.0, 0.0
		for _, series := range chart.series {
			value := series.values[valueIndex]
			if isMissing(value) {
				continue
			}
			if value >= 0 {
				positive += value
			} else {
				negative += value
			}
		}
		minimum = math.Min(minimum, negative)
		maximum = math.Max(maximum, positive)
	}
	return minimum, maximum
}

func newGraphicalValueAxis(minimum, maximum float64, padForLabels bool) graphicalValueAxis {
	return newGraphicalValueAxisWithIntervals(minimum, maximum, 6, padForLabels)
}

func newGraphicalValueAxisWithIntervals(minimum, maximum float64, intervals int, padForLabels bool) graphicalValueAxis {
	if minimum == maximum {
		if minimum == 0 {
			minimum, maximum = -1, 1
		} else {
			padding := math.Max(1, math.Abs(minimum)*0.2)
			minimum -= padding
			maximum += padding
		}
	}
	rawStep := (maximum - minimum) / float64(max(1, intervals))
	magnitude := math.Pow(10, math.Floor(math.Log10(rawStep)))
	fraction := rawStep / magnitude
	niceFraction := 1.0
	switch {
	case fraction < 1.5:
		niceFraction = 1
	case fraction < 2.25:
		niceFraction = 2
	case fraction < 3.75:
		niceFraction = 2.5
	case fraction < 7.5:
		niceFraction = 5
	default:
		niceFraction = 10
	}
	step := niceFraction * magnitude
	axisMinimum := math.Floor(minimum/step) * step
	axisMaximum := math.Ceil(maximum/step) * step
	if padForLabels {
		if maximum >= axisMaximum-step*0.05 {
			axisMaximum += step
		}
		if minimum <= axisMinimum+step*0.05 {
			axisMinimum -= step
		}
	}
	return graphicalValueAxis{minimum: axisMinimum, maximum: axisMaximum, step: step}
}

func (axis graphicalValueAxis) ticks() []float64 {
	count := int(math.Round((axis.maximum-axis.minimum)/axis.step)) + 1
	count = min(20, max(2, count))
	ticks := make([]float64, count)
	for index := range ticks {
		ticks[index] = axis.minimum + float64(index)*axis.step
		if math.Abs(ticks[index]) < axis.step*1e-9 {
			ticks[index] = 0
		}
	}
	return ticks
}

func (chart *Chart) signedBarLegendHeight(painter *charts.Painter, theme charts.ColorPalette, left, right int) int {
	if !displayValue(chart.spec.Legend, len(chart.series) > 1) {
		return 0
	}
	style := charts.NewFontStyleWithSize(10.5).WithColor(theme.GetLegendTextColor())
	return chart.scatterLegendRows(painter, style, max(1, right-left)) * 20
}

func (chart *Chart) drawSignedBarLegend(painter *charts.Painter, theme charts.ColorPalette, left, right, top int) {
	if !displayValue(chart.spec.Legend, len(chart.series) > 1) {
		return
	}
	style := charts.NewFontStyleWithSize(10.5).WithColor(theme.GetLegendTextColor())
	x, y := left, top+12
	for seriesIndex, series := range chart.series {
		itemWidth := painter.MeasureText(series.spec.Label, 0, style).Width() + 34
		if x > left && x+itemWidth > right {
			x = left
			y += 20
		}
		painter.FilledRect(x, y-9, x+9, y, theme.GetSeriesColor(seriesIndex), charts.ColorTransparent, 0)
		painter.Text(series.spec.Label, x+14, y+1, 0, style)
		x += itemWidth
	}
}

func (chart *Chart) drawGraphicalSignedVerticalBars(painter *charts.Painter, theme charts.ColorPalette, axis graphicalValueAxis, showAxes, showLabels bool) error {
	axisStyle := charts.NewFontStyleWithSize(10.5).WithColor(theme.GetYAxisTextColor())
	labelStyle := charts.NewFontStyleWithSize(9.5).WithColor(theme.GetLabelTextColor())
	left, right := 18, painter.Width()-20
	top := 16
	legendHeight := chart.signedBarLegendHeight(painter, theme, left, right)
	chart.drawSignedBarLegend(painter, theme, left, right, top)
	top += legendHeight
	bottom := painter.Height() - 18
	if showAxes {
		maxTickWidth := 0
		for _, tick := range axis.ticks() {
			maxTickWidth = max(maxTickWidth, painter.MeasureText(formatValue(tick), 0, axisStyle).Width())
		}
		left += maxTickWidth + 12
		bottom -= painter.MeasureText("Ag", 0, axisStyle).Height() + 16
	}
	if right-left < 80 || bottom-top < 60 {
		return fmt.Errorf("image is too small for signed bar chart")
	}
	plotWidth, plotHeight := right-left, bottom-top
	mapY := func(value float64) int {
		return bottom - int(math.Round((value-axis.minimum)/(axis.maximum-axis.minimum)*float64(plotHeight)))
	}
	zeroY := mapY(0)
	if showAxes {
		for _, tick := range axis.ticks() {
			y := mapY(tick)
			color, width := theme.GetAxisSplitLineColor(), 0.6
			if tick == 0 {
				color, width = theme.GetXAxisStrokeColor(), 1.1
			}
			painter.LineStroke([]charts.Point{{X: left, Y: y}, {X: right, Y: y}}, color, width)
			text := formatValue(tick)
			box := painter.MeasureText(text, 0, axisStyle)
			painter.Text(text, left-box.Width()-8, y+box.Height()/2, 0, axisStyle)
		}
	}

	categoryCount := len(chart.labels)
	seriesCount := len(chart.series)
	stacked := chart.spec.Layout == Stacked
	stackedLabelThreshold := (axis.maximum - axis.minimum) * 0.01
	for categoryIndex, category := range chart.labels {
		slotLeft := left + categoryIndex*plotWidth/categoryCount
		slotRight := left + (categoryIndex+1)*plotWidth/categoryCount
		slotWidth := slotRight - slotLeft
		if showAxes {
			text := fitPainterText(painter, category, max(1, slotWidth-8), axisStyle)
			box := painter.MeasureText(text, 0, axisStyle)
			painter.Text(text, (slotLeft+slotRight-box.Width())/2, bottom+box.Height()+10, 0, axisStyle)
		}
		if stacked {
			barWidth := max(1, slotWidth*56/100)
			x := (slotLeft + slotRight - barWidth) / 2
			positiveBase, negativeBase := 0.0, 0.0
			for seriesIndex, series := range chart.series {
				value := series.values[categoryIndex]
				if isMissing(value) {
					continue
				}
				start := positiveBase
				if value < 0 {
					start = negativeBase
					negativeBase += value
				} else {
					positiveBase += value
				}
				showValue := showLabels && math.Abs(value) > stackedLabelThreshold
				chart.drawSignedVerticalBar(painter, theme.GetSeriesColor(seriesIndex), labelStyle, x, barWidth, mapY(start), mapY(start+value), zeroY, value, showValue)
			}
			continue
		}

		groupWidth := max(1, slotWidth*76/100)
		gap := 2
		barWidth := max(1, (groupWidth-gap*(seriesCount-1))/seriesCount)
		groupLeft := (slotLeft + slotRight - (barWidth*seriesCount + gap*(seriesCount-1))) / 2
		for seriesIndex, series := range chart.series {
			value := series.values[categoryIndex]
			if isMissing(value) {
				continue
			}
			x := groupLeft + seriesIndex*(barWidth+gap)
			chart.drawSignedVerticalBar(painter, theme.GetSeriesColor(seriesIndex), labelStyle, x, barWidth, zeroY, mapY(value), zeroY, value, showLabels)
		}
	}
	return nil
}

func (chart *Chart) drawSignedVerticalBar(painter *charts.Painter, color charts.Color, labelStyle charts.FontStyle, x, width, startY, valueY, zeroY int, value float64, showLabel bool) {
	if value != 0 {
		top, bottom := min(startY, valueY), max(startY, valueY)
		if top == bottom {
			if value > 0 {
				top--
			} else {
				bottom++
			}
		}
		painter.FilledRect(x, top, x+width, bottom, color, charts.ColorTransparent, 0)
	}
	if !showLabel {
		return
	}
	text := formatValue(value)
	box := painter.MeasureText(text, 0, labelStyle)
	labelY := zeroY - 4
	if value > 0 {
		labelY = min(startY, valueY) - 4
	} else if value < 0 {
		labelY = max(startY, valueY) + box.Height() + 3
	}
	painter.Text(text, x+(width-box.Width())/2, labelY, 0, labelStyle)
}

func (chart *Chart) drawGraphicalSignedHorizontalBars(painter *charts.Painter, theme charts.ColorPalette, axis graphicalValueAxis, showAxes, showLabels bool) error {
	axisStyle := charts.NewFontStyleWithSize(10.5).WithColor(theme.GetXAxisTextColor())
	labelStyle := charts.NewFontStyleWithSize(9.5).WithColor(theme.GetLabelTextColor())
	left, right := 18, painter.Width()-20
	top := 16
	legendHeight := chart.signedBarLegendHeight(painter, theme, left, right)
	chart.drawSignedBarLegend(painter, theme, left, right, top)
	top += legendHeight
	bottom := painter.Height() - 18
	rowLabelWidth := 0
	if showAxes {
		limit := max(72, painter.Width()/3)
		for _, category := range chart.labels {
			rowLabelWidth = max(rowLabelWidth, min(limit, painter.MeasureText(category, 0, axisStyle).Width()))
		}
		left += rowLabelWidth + 12
		bottom -= painter.MeasureText("Ag", 0, axisStyle).Height() + 16
	}
	if right-left < 80 || bottom-top < 60 {
		return fmt.Errorf("image is too small for signed horizontal bar chart")
	}
	plotWidth, plotHeight := right-left, bottom-top
	mapX := func(value float64) int {
		return left + int(math.Round((value-axis.minimum)/(axis.maximum-axis.minimum)*float64(plotWidth)))
	}
	zeroX := mapX(0)
	if showAxes {
		for _, tick := range axis.ticks() {
			x := mapX(tick)
			color, width := theme.GetAxisSplitLineColor(), 0.6
			if tick == 0 {
				color, width = theme.GetYAxisStrokeColor(), 1.1
			}
			painter.LineStroke([]charts.Point{{X: x, Y: top}, {X: x, Y: bottom}}, color, width)
			text := formatValue(tick)
			box := painter.MeasureText(text, 0, axisStyle)
			painter.Text(text, x-box.Width()/2, bottom+box.Height()+10, 0, axisStyle)
		}
	}

	categoryCount := len(chart.labels)
	seriesCount := len(chart.series)
	stacked := chart.spec.Layout == Stacked
	stackedLabelThreshold := (axis.maximum - axis.minimum) * 0.01
	for categoryIndex, category := range chart.labels {
		slotTop := top + categoryIndex*plotHeight/categoryCount
		slotBottom := top + (categoryIndex+1)*plotHeight/categoryCount
		slotHeight := slotBottom - slotTop
		if showAxes {
			text := fitPainterText(painter, category, rowLabelWidth, axisStyle)
			box := painter.MeasureText(text, 0, axisStyle)
			painter.Text(text, left-box.Width()-10, (slotTop+slotBottom+box.Height())/2-1, 0, axisStyle)
		}
		if stacked {
			barHeight := max(1, slotHeight*52/100)
			y := (slotTop + slotBottom - barHeight) / 2
			positiveBase, negativeBase := 0.0, 0.0
			for seriesIndex, series := range chart.series {
				value := series.values[categoryIndex]
				if isMissing(value) {
					continue
				}
				start := positiveBase
				if value < 0 {
					start = negativeBase
					negativeBase += value
				} else {
					positiveBase += value
				}
				showValue := showLabels && math.Abs(value) > stackedLabelThreshold
				chart.drawSignedHorizontalBar(painter, theme.GetSeriesColor(seriesIndex), labelStyle, y, barHeight, mapX(start), mapX(start+value), zeroX, value, showValue)
			}
			continue
		}

		groupHeight := max(1, slotHeight*74/100)
		gap := 2
		barHeight := max(1, (groupHeight-gap*(seriesCount-1))/seriesCount)
		groupTop := (slotTop + slotBottom - (barHeight*seriesCount + gap*(seriesCount-1))) / 2
		for seriesIndex, series := range chart.series {
			value := series.values[categoryIndex]
			if isMissing(value) {
				continue
			}
			y := groupTop + seriesIndex*(barHeight+gap)
			chart.drawSignedHorizontalBar(painter, theme.GetSeriesColor(seriesIndex), labelStyle, y, barHeight, zeroX, mapX(value), zeroX, value, showLabels)
		}
	}
	return nil
}

func (chart *Chart) drawSignedHorizontalBar(painter *charts.Painter, color charts.Color, labelStyle charts.FontStyle, y, height, startX, valueX, zeroX int, value float64, showLabel bool) {
	if value != 0 {
		left, right := min(startX, valueX), max(startX, valueX)
		if left == right {
			if value > 0 {
				right++
			} else {
				left--
			}
		}
		painter.FilledRect(left, y, right, y+height, color, charts.ColorTransparent, 0)
	}
	if !showLabel || height < 8 {
		return
	}
	text := formatValue(value)
	box := painter.MeasureText(text, 0, labelStyle)
	labelX := zeroX + 4
	if value > 0 {
		labelX = max(startX, valueX) + 5
	} else if value < 0 {
		labelX = min(startX, valueX) - box.Width() - 5
	}
	painter.Text(text, labelX, y+(height+box.Height())/2-1, 0, labelStyle)
}
