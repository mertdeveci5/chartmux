package chartmux

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/go-analyze/charts"
)

const (
	minimumDirectSliceShare = 0.03
	minimumFunnelStageWidth = 3
)

func (chart *Chart) allValuesMissing() bool {
	for _, series := range chart.series {
		for _, value := range series.values {
			if !isMissing(value) {
				return false
			}
		}
	}
	return true
}

func drawGraphicalNoData(painter *charts.Painter, theme charts.ColorPalette) {
	style := charts.NewFontStyleWithSize(11).
		WithColor(theme.GetLabelTextColor()).
		WithFont(charts.GetFont(charts.FontFamilyNotoSansBold))
	text := "No data"
	box := painter.MeasureText(text, 0, style)
	painter.Text(text, (painter.Width()-box.Width())/2, (painter.Height()+box.Height())/2, 0, style)
}

func (chart *Chart) graphicalBarSeries(showLabels, horizontal bool) (charts.BarSeriesList, []string) {
	values := chart.seriesValues()
	labels := append([]string(nil), chart.labels...)
	stackedLabelThreshold := chart.graphicalStackedLabelThreshold()
	for seriesIndex := range values {
		for valueIndex, value := range values[seriesIndex] {
			if !isMissing(value) {
				continue
			}
			values[seriesIndex][valueIndex] = math.SmallestNonzeroFloat64
		}
		if horizontal {
			slices.Reverse(values[seriesIndex])
		}
	}
	if horizontal {
		slices.Reverse(labels)
	}

	series := charts.NewSeriesListBar(values, charts.BarSeriesOption{Names: chart.seriesNames()})
	for seriesIndex := range series {
		series[seriesIndex].Label.Show = charts.Ptr(showLabels)
		if !showLabels {
			continue
		}
		series[seriesIndex].Label.LabelFormatter = func(_ int, _ string, value float64) (string, *charts.LabelStyle) {
			if value == math.SmallestNonzeroFloat64 || (stackedLabelThreshold > 0 && value <= stackedLabelThreshold) {
				return "", nil
			}
			return formatValue(value), nil
		}
	}
	return series, labels
}

func (chart *Chart) graphicalStackedLabelThreshold() float64 {
	if chart.spec.Layout != Stacked && chart.spec.Layout != Normalized {
		return 0
	}
	maximum := 0.0
	for valueIndex := range chart.labels {
		total := 0.0
		for _, series := range chart.series {
			value := series.values[valueIndex]
			if !isMissing(value) && value > 0 {
				total += value
			}
		}
		maximum = math.Max(maximum, total)
	}
	return maximum * 0.01
}

func (chart *Chart) graphicalPolarValues() []float64 {
	values := append([]float64(nil), chart.series[0].values...)
	for index, value := range values {
		if isMissing(value) {
			values[index] = 0
		}
	}
	return values
}

func graphicalPolarLabel(values []float64, show bool) charts.SeriesLabel {
	label := charts.SeriesLabel{Show: charts.Ptr(show)}
	if !show {
		return label
	}
	total := 0.0
	for _, value := range values {
		if value > 0 {
			total += value
		}
	}
	label.LabelFormatter = func(_ int, name string, value float64) (string, *charts.LabelStyle) {
		if total <= 0 || value/total < minimumDirectSliceShare {
			return "", nil
		}
		return fmt.Sprintf("%s: %s", name, formatPercent(value/total)), nil
	}
	return label
}

func formatPercent(share float64) string {
	percent := share * 100
	switch {
	case percent == math.Trunc(percent):
		return fmt.Sprintf("%.0f%%", percent)
	case percent >= 1:
		return fmt.Sprintf("%.1f%%", percent)
	case percent >= 0.1:
		return fmt.Sprintf("%.2f%%", percent)
	case percent >= 0.01:
		return fmt.Sprintf("%.3f%%", percent)
	default:
		return fmt.Sprintf("%.4f%%", percent)
	}
}

func (chart *Chart) graphicalRadarSeries(showLabels bool) charts.RadarSeriesList {
	values := chart.seriesValues()
	for seriesIndex := range values {
		for valueIndex, value := range values[seriesIndex] {
			if !isMissing(value) {
				continue
			}
			values[seriesIndex][valueIndex] = -math.SmallestNonzeroFloat64
		}
	}
	series := charts.NewSeriesListRadar(values, charts.RadarSeriesOption{Names: chart.seriesNames()})
	for seriesIndex := range series {
		series[seriesIndex].Label.Show = charts.Ptr(showLabels)
		series[seriesIndex].Label.ValueFormatter = func(value float64) string {
			if value < 0 {
				return ""
			}
			return formatValue(value)
		}
	}
	return series
}

func (chart *Chart) drawGraphicalHeatmap(painter *charts.Painter, theme charts.ColorPalette, showAxes, showLabels bool) error {
	rows := len(chart.labels)
	columns := len(chart.series)
	if rows == 0 || columns == 0 {
		return nil
	}

	axisStyle := charts.NewFontStyleWithSize(10.5).WithColor(theme.GetXAxisTextColor())
	valueStyle := charts.NewFontStyleWithSize(10).WithColor(theme.GetLabelTextColor())
	left, top := 18, 18
	right, bottom := painter.Width()-18, painter.Height()-18
	rowLabelWidth := 0
	if showAxes {
		rowLabelLimit := max(72, painter.Width()/3)
		for _, label := range chart.labels {
			rowLabelWidth = max(rowLabelWidth, min(rowLabelLimit, painter.MeasureText(label, 0, axisStyle).Width()))
		}
		left += rowLabelWidth + 12
		bottom -= painter.MeasureText("Ag", 0, axisStyle).Height() + 12
	}
	gridWidth := right - left
	gridHeight := bottom - top
	if gridWidth/columns < 6 || gridHeight/rows < 6 {
		return fmt.Errorf("image is too small for %dx%d heatmap cells", rows, columns)
	}

	minimum, maximum, hasValue := chart.graphicalHeatmapRange()
	if !hasValue || minimum == maximum {
		minimum = 0
		maximum = 1
	}
	valueRange := maximum - minimum
	baseColor := theme.GetSeriesColor(0)
	missingColor := theme.GetAxisSplitLineColor()
	gap := 2
	if min(gridWidth/columns, gridHeight/rows) >= 48 {
		gap = 3
	}

	for row := 0; row < rows; row++ {
		y1 := top + row*gridHeight/rows
		y2 := top + (row+1)*gridHeight/rows
		for column := 0; column < columns; column++ {
			x1 := left + column*gridWidth/columns
			x2 := left + (column+1)*gridWidth/columns
			value := chart.series[column].values[row]
			cellColor := missingColor
			if !isMissing(value) {
				ratio := min(1.0, max(0.0, (value-minimum)/valueRange))
				lightness := (1 - ratio) * 0.42
				if theme.IsDark() {
					lightness *= -1
				}
				cellColor = baseColor.WithAdjustHSL(0, (1-ratio)*0.08, lightness)
			}
			painter.FilledRect(x1+gap/2, y1+gap/2, x2-(gap-gap/2), y2-(gap-gap/2), cellColor, charts.ColorTransparent, 0)

			if !showLabels {
				continue
			}
			text := "–"
			if !isMissing(value) {
				text = formatValue(value)
			}
			style := valueStyle.WithColor(heatmapTextColor(cellColor))
			box := painter.MeasureText(text, 0, style)
			if box.Width()+8 > x2-x1 || box.Height()+6 > y2-y1 {
				continue
			}
			painter.Text(text, (x1+x2-box.Width())/2, (y1+y2+box.Height())/2-1, 0, style)
		}
	}

	if !showAxes {
		return nil
	}
	for row, label := range chart.labels {
		y1 := top + row*gridHeight/rows
		y2 := top + (row+1)*gridHeight/rows
		text := fitPainterText(painter, label, rowLabelWidth, axisStyle)
		box := painter.MeasureText(text, 0, axisStyle)
		painter.Text(text, left-box.Width()-10, (y1+y2+box.Height())/2-1, 0, axisStyle)
	}
	for column, series := range chart.series {
		x1 := left + column*gridWidth/columns
		x2 := left + (column+1)*gridWidth/columns
		text := fitPainterText(painter, series.spec.Label, max(1, x2-x1-8), axisStyle)
		box := painter.MeasureText(text, 0, axisStyle)
		painter.Text(text, (x1+x2-box.Width())/2, bottom+box.Height()+8, 0, axisStyle)
	}
	return nil
}

func (chart *Chart) graphicalHeatmapRange() (float64, float64, bool) {
	minimum, maximum := math.MaxFloat64, -math.MaxFloat64
	for _, series := range chart.series {
		for _, value := range series.values {
			if isMissing(value) {
				continue
			}
			minimum = math.Min(minimum, value)
			maximum = math.Max(maximum, value)
		}
	}
	return minimum, maximum, minimum != math.MaxFloat64
}

func heatmapTextColor(fill charts.Color) charts.Color {
	red := float64(fill.R) / 255
	green := float64(fill.G) / 255
	blue := float64(fill.B) / 255
	luminance := 0.2126*red + 0.7152*green + 0.0722*blue
	if luminance < 0.5 {
		return charts.ColorWhite
	}
	return charts.Color{R: 24, G: 24, B: 27, A: 255}
}

func (chart *Chart) graphicalFunnelWidths(availableWidth int) []int {
	widths := make([]int, len(chart.labels))
	maximum := chart.graphicalFunnelMaximum()
	if maximum <= 0 || availableWidth <= 0 {
		return widths
	}
	for index, value := range chart.series[0].values {
		if isMissing(value) || value <= 0 {
			continue
		}
		widths[index] = max(minimumFunnelStageWidth, int(math.Round(value/maximum*float64(availableWidth))))
		widths[index] = min(availableWidth, widths[index])
	}
	return widths
}

func (chart *Chart) graphicalFunnelMaximum() float64 {
	maximum := 0.0
	for _, value := range chart.series[0].values {
		if !isMissing(value) {
			maximum = math.Max(maximum, value)
		}
	}
	return maximum
}

func (chart *Chart) drawGraphicalFunnel(painter *charts.Painter, theme charts.ColorPalette, showStageText bool) error {
	stageCount := len(chart.labels)
	if stageCount == 0 {
		return nil
	}
	margin := 24
	labelWidth := 0
	if showStageText {
		labelWidth = min(360, max(180, painter.Width()*38/100))
	}
	barLeft := margin
	barRight := painter.Width() - margin - labelWidth
	if barRight-barLeft < 80 {
		return fmt.Errorf("image is too narrow for funnel chart labels")
	}
	barCenter := (barLeft + barRight) / 2
	availableWidth := barRight - barLeft
	widths := chart.graphicalFunnelWidths(availableWidth)
	maximum := chart.graphicalFunnelMaximum()
	if maximum <= 0 {
		style := charts.NewFontStyleWithSize(11).WithColor(theme.GetXAxisTextColor())
		text := "No data"
		box := painter.MeasureText(text, 0, style)
		painter.Text(text, (painter.Width()-box.Width())/2, (painter.Height()+box.Height())/2, 0, style)
		return nil
	}

	rowHeight := painter.Height() / stageCount
	if rowHeight < 12 {
		return fmt.Errorf("image is too short for %d funnel stages", stageCount)
	}
	barHeight := min(36, max(8, rowHeight*48/100))
	trackColor := theme.GetAxisSplitLineColor()
	baseColor := theme.GetSeriesColor(0)
	nameStyle := charts.NewFontStyleWithSize(11).WithColor(theme.GetLabelTextColor()).WithFont(charts.GetFont(charts.FontFamilyNotoSansBold))
	valueStyle := charts.NewFontStyleWithSize(9.5).WithColor(theme.GetXAxisTextColor())
	initial := chart.series[0].values[0]
	denominatorLabel := "initial"
	if isMissing(initial) || initial <= 0 {
		initial = maximum
		denominatorLabel = "peak"
	}

	for index, value := range chart.series[0].values {
		centerY := index*rowHeight + rowHeight/2
		y1 := centerY - barHeight/2
		y2 := y1 + barHeight
		painter.FilledRect(barLeft, y1, barRight, y2, trackColor, charts.ColorTransparent, 0)
		if widths[index] > 0 {
			x1 := barCenter - widths[index]/2
			painter.FilledRect(x1, y1, x1+widths[index], y2, baseColor, charts.ColorTransparent, 0)
		} else if !isMissing(value) {
			painter.LineStroke([]charts.Point{{X: barCenter, Y: y1}, {X: barCenter, Y: y2}}, theme.GetLabelTextColor(), 1)
		}
		if !showStageText {
			continue
		}

		labelX := barRight + 18
		textWidth := max(1, painter.Width()-margin-labelX)
		name := fitPainterText(painter, chart.labels[index], textWidth, nameStyle)
		nameBox := painter.MeasureText(name, 0, nameStyle)
		if rowHeight >= 32 {
			painter.Text(name, labelX, centerY-2, 0, nameStyle)
			valueText := "–"
			if !isMissing(value) {
				valueText = fmt.Sprintf("%s · %s of %s", formatValue(value), formatPercent(value/initial), denominatorLabel)
			}
			valueText = fitPainterText(painter, valueText, textWidth, valueStyle)
			painter.Text(valueText, labelX, centerY+nameBox.Height()+2, 0, valueStyle)
			continue
		}
		inline := name
		if !isMissing(value) {
			inline += " · " + formatValue(value)
		}
		inline = fitPainterText(painter, inline, textWidth, valueStyle)
		inlineBox := painter.MeasureText(inline, 0, valueStyle)
		painter.Text(inline, labelX, centerY+inlineBox.Height()/2, 0, valueStyle)
	}
	return nil
}

func fitPainterText(painter *charts.Painter, text string, width int, style charts.FontStyle) string {
	if width <= 0 {
		return ""
	}
	if painter.MeasureText(text, 0, style).Width() <= width {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if ampersand := strings.LastIndex(candidate, "&"); ampersand > strings.LastIndex(candidate, ";") {
			candidate = candidate[:ampersand] + "…"
		}
		if painter.MeasureText(candidate, 0, style).Width() <= width {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}
