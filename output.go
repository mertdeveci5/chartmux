package chartmux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/go-analyze/charts"
)

type ImageOptions struct {
	Width  int
	Height int
}

type HTMLOptions struct {
	Width  int
	Height int
}

func (chart *Chart) WriteJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(chart.ResolvedSpec()); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

func (chart *Chart) WritePNG(writer io.Writer, options ImageOptions) error {
	width, height, err := imageDimensions(options, 1200, 720)
	if err != nil {
		return err
	}
	content, err := chart.bytes(charts.ChartOutputPNG, width, height)
	if err != nil {
		return err
	}
	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("write PNG: %w", err)
	}
	return nil
}

func (chart *Chart) WriteSVG(writer io.Writer, options ImageOptions) error {
	width, height, err := imageDimensions(options, 960, 540)
	if err != nil {
		return err
	}
	content, err := chart.bytes(charts.ChartOutputSVG, width, height)
	if err != nil {
		return err
	}
	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("write SVG: %w", err)
	}
	return nil
}

func (chart *Chart) WriteHTML(writer io.Writer, options HTMLOptions) error {
	width, height, err := imageDimensions(ImageOptions(options), 960, 540)
	if err != nil {
		return err
	}
	svg, err := chart.bytes(charts.ChartOutputSVG, width, height)
	if err != nil {
		return err
	}
	if index := bytes.Index(svg, []byte("<svg")); index >= 0 {
		svg = svg[index:]
	}
	background := "#fafafa"
	card := "#ffffff"
	foreground := "#09090b"
	border := "#e4e4e7"
	shadow := "rgba(24,24,27,.08)"
	if chart.spec.Theme == "dark" {
		background = "#09090b"
		card = "#18181b"
		foreground = "#fafafa"
		border = "#3f3f46"
		shadow = "rgba(0,0,0,.32)"
	}
	document := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="%s">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src data:">
<title>%s</title>
<style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:clamp(12px,4vw,48px);background:%s;color:%s;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}.chart-card{width:min(100%%,1120px);padding:clamp(12px,2vw,24px);overflow:hidden;background:%s;border:1px solid %s;border-radius:18px;box-shadow:0 18px 50px %s}.chart-card svg{display:block;width:100%%;height:auto}@media(max-width:560px){body{padding:8px;place-items:start center}.chart-card{padding:8px;border-radius:12px}}
</style>
</head>
<body>
<main class="chart-card" data-chart-type="%s">%s</main>
</body>
</html>
`, chart.spec.Theme, html.EscapeString(chart.spec.Title), background, foreground, card, border, shadow, chart.spec.Type, svg)
	if _, err := io.WriteString(writer, document); err != nil {
		return fmt.Errorf("write HTML: %w", err)
	}
	return nil
}

func (chart *Chart) PNG(options ImageOptions) ([]byte, error) {
	var buffer bytes.Buffer
	if err := chart.WritePNG(&buffer, options); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (chart *Chart) bytes(format string, width, height int) ([]byte, error) {
	renderChart := chart.cloneForRender()
	if format == charts.ChartOutputSVG {
		renderChart = renderChart.escapedForSVG()
	}
	renderChart.spec.Title = ""
	renderChart.spec.Description = ""
	renderChart.spec.Footer = ""
	renderChart.spec.Annotations = nil
	theme, err := renderChart.theme()
	if err != nil {
		return nil, err
	}
	measurementPainter := charts.NewPainter(charts.PainterOptions{
		OutputFormat: format,
		Width:        width,
		Height:       height,
		Theme:        theme,
	})
	textLayout, err := chart.graphicalTextLayout(measurementPainter, theme, width, height)
	if err != nil {
		return nil, err
	}
	if renderChart.allValuesMissing() {
		painter := measurementPainter
		painter.FilledRect(0, 0, width, height, theme.GetBackgroundColor(), charts.ColorTransparent, 0)
		chart.drawGraphicalText(painter, textLayout, theme, format == charts.ChartOutputSVG, width, height, 0)
		plotPainter := painter.Child(charts.PainterBoxOption(textLayout.plotBox(width, height)))
		drawGraphicalNoData(plotPainter, theme)
		content, err := painter.Bytes()
		if err != nil {
			return nil, fmt.Errorf("encode %s chart: %w", format, err)
		}
		return content, nil
	}
	if renderChart.spec.Type == Combo {
		if renderChart.comboHasBarsOnSignedRange() {
			return nil, fmt.Errorf("image export cannot render a signed combo accurately; use a signed bar chart or keep the shared combo range non-negative")
		}
		option := renderChart.comboOption(theme, width, height, format)
		option.Box = textLayout.plotBox(width, height)
		painter, err := charts.Render(option)
		if err != nil {
			return nil, fmt.Errorf("build combo chart: %w", err)
		}
		if textLayout.topHeight > 0 {
			painter.FilledRect(0, -textLayout.topHeight, width, 0, theme.GetBackgroundColor(), charts.ColorTransparent, 0)
		}
		if textLayout.bottomHeight > 0 {
			painter.FilledRect(0, painter.Height(), width, height-textLayout.topHeight, theme.GetBackgroundColor(), charts.ColorTransparent, 0)
		}
		chart.drawGraphicalText(painter, textLayout, theme, format == charts.ChartOutputSVG, width, height, textLayout.topHeight)
		return painter.Bytes()
	}
	painter := measurementPainter
	painter.FilledRect(0, 0, width, height, theme.GetBackgroundColor(), charts.ColorTransparent, 0)
	chart.drawGraphicalText(painter, textLayout, theme, format == charts.ChartOutputSVG, width, height, 0)
	plotPainter := painter.Child(charts.PainterBoxOption(textLayout.plotBox(width, height)))
	if err := renderChart.draw(plotPainter, theme); err != nil {
		return nil, err
	}
	content, err := painter.Bytes()
	if err != nil {
		return nil, fmt.Errorf("encode %s chart: %w", format, err)
	}
	return content, nil
}

func (chart *Chart) draw(painter *charts.Painter, theme charts.ColorPalette) error {
	title := charts.TitleOption{}
	legend := charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, len(chart.series) > 1)), SeriesNames: chart.seriesNames()}
	showAxes := displayValue(chart.spec.Axes, true)
	showLabels := displayValue(chart.spec.Labels, false)
	label := charts.SeriesLabel{Show: charts.Ptr(showLabels)}
	values := chart.seriesValues()

	switch chart.spec.Type {
	case Bar, Histogram:
		if !chart.allNonNegative() {
			return chart.drawGraphicalSignedBars(painter, theme, showAxes, showLabels)
		}
		horizontal := chart.spec.Orientation == Horizontal
		series, categoryLabels := chart.graphicalBarSeries(showLabels, horizontal)
		option := charts.NewBarChartOptionWithSeries(series)
		option.Theme = theme
		option.Title = title
		option.Legend = legend
		option.Horizontal = horizontal
		option.CategoryAxis.Labels = categoryLabels
		option.CategoryAxis.Show = charts.Ptr(showAxes)
		if horizontal {
			option.CategoryAxis.LabelFontStyle = charts.NewFontStyleWithSize(9.5)
			if len(categoryLabels) > 0 && painter.Height()/len(categoryLabels) >= 24 {
				option.CategoryAxis.LabelCount = len(categoryLabels)
			}
		}
		option.ValueAxis[0].Show = charts.Ptr(showAxes)
		option.ValueAxis[0].PreferNiceIntervals = charts.Ptr(true)
		if chart.allNonNegative() {
			option.ValueAxis[0].Min = charts.Ptr(0.0)
		}
		if chart.spec.Layout == Normalized {
			option.ValueAxis[0].Max = charts.Ptr(100.0)
			option.ValueAxis[0].Unit = 25
			option.ValueAxis[0].LabelCount = 5
			option.ValueAxis[0].PreferNiceIntervals = charts.Ptr(false)
			option.ValueAxis[0].ValueFormatter = func(value float64) string {
				return formatValue(value) + "%"
			}
		}
		option.StackSeries = charts.Ptr(chart.spec.Layout == Stacked || chart.spec.Layout == Normalized)
		option.RoundedBarCaps = charts.Ptr(false)
		return painter.BarChart(option)
	case Line, Area:
		series := charts.NewSeriesListLine(values, charts.LineSeriesOption{Names: chart.seriesNames(), Label: label})
		option := charts.NewLineChartOptionWithSeries(series)
		option.Theme = theme
		option.Title = title
		option.Legend = legend
		option.XAxis.Labels = chart.labels
		option.XAxis.Show = charts.Ptr(showAxes)
		option.YAxis[0].Show = charts.Ptr(showAxes)
		option.YAxis[0].PreferNiceIntervals = charts.Ptr(true)
		option.LineStrokeWidth = 2.5
		if chart.spec.Curve == Smooth {
			option.StrokeSmoothingTension = 0.35
		}
		if chart.spec.Type == Area {
			option.FillArea = charts.Ptr(true)
			option.FillOpacity = 72
			option.StackSeries = charts.Ptr(chart.spec.Layout == Stacked || chart.spec.Layout == Normalized)
			if chart.allNonNegative() {
				option.YAxis[0].Min = charts.Ptr(0.0)
			}
			if chart.spec.Layout == Normalized {
				option.YAxis[0].Max = charts.Ptr(100.0)
			}
		}
		return painter.LineChart(option)
	case Scatter:
		return chart.drawScatter(painter, theme, showAxes, showLabels)
	case Pie:
		polarValues := chart.graphicalPolarValues()
		polarLabel := graphicalPolarLabel(polarValues, showLabels)
		series := charts.NewSeriesListPie(polarValues, charts.PieSeriesOption{Names: chart.labels, Label: polarLabel})
		option := charts.PieChartOption{
			Theme:      theme,
			SeriesList: series,
			Title:      title,
			Legend:     charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, true)), SeriesNames: chart.labels},
		}
		return painter.PieChart(option)
	case Donut:
		polarValues := chart.graphicalPolarValues()
		polarLabel := graphicalPolarLabel(polarValues, showLabels)
		series := charts.NewSeriesListDoughnut(polarValues, charts.DoughnutSeriesOption{Names: chart.labels, Label: polarLabel})
		option := charts.DoughnutChartOption{
			Theme:        theme,
			SeriesList:   series,
			Title:        title,
			Legend:       charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, true)), SeriesNames: chart.labels},
			CenterValues: "sum",
		}
		return painter.DoughnutChart(option)
	case Heatmap:
		return chart.drawGraphicalHeatmap(painter, theme, showAxes, showLabels)
	case Radar:
		maxima := chart.radarMaxima()
		series := chart.graphicalRadarSeries(showLabels)
		option := charts.RadarChartOption{
			Theme:           theme,
			SeriesList:      series,
			RadarIndicators: charts.NewRadarIndicators(chart.labels, maxima),
			Title:           title,
			Legend:          legend,
		}
		return painter.RadarChart(option)
	case Funnel:
		showStageText := showLabels || displayValue(chart.spec.Legend, false)
		return chart.drawGraphicalFunnel(painter, theme, showStageText)
	default:
		return fmt.Errorf("chart type %q has no output engine", chart.spec.Type)
	}
}

func (chart *Chart) comboOption(theme charts.ColorPalette, width, height int, format string) charts.ChartOption {
	series := make(charts.GenericSeriesList, len(chart.series))
	showLabels := displayValue(chart.spec.Labels, false)
	for index, item := range chart.series {
		chartType := charts.ChartTypeLine
		values := append([]float64(nil), item.values...)
		label := charts.SeriesLabel{Show: charts.Ptr(showLabels)}
		if item.spec.Mark == MarkBar {
			chartType = charts.ChartTypeBar
			for valueIndex, value := range values {
				if isMissing(value) {
					values[valueIndex] = math.SmallestNonzeroFloat64
				}
			}
			if showLabels {
				label.LabelFormatter = func(_ int, _ string, value float64) (string, *charts.LabelStyle) {
					if value == math.SmallestNonzeroFloat64 {
						return "", nil
					}
					return formatValue(value), nil
				}
			}
		}
		series[index] = charts.GenericSeries{Type: chartType, Values: values, Name: item.spec.Label, Label: label}
	}
	showAxes := displayValue(chart.spec.Axes, true)
	option := charts.ChartOption{
		OutputFormat:    format,
		Width:           width,
		Height:          height,
		Theme:           theme,
		Title:           charts.TitleOption{},
		Legend:          charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, true)), SeriesNames: chart.seriesNames()},
		XAxis:           charts.XAxisOption{Show: charts.Ptr(showAxes), Labels: chart.labels},
		YAxis:           []charts.YAxisOption{{Show: charts.Ptr(showAxes), PreferNiceIntervals: charts.Ptr(true)}},
		SeriesList:      series,
		LineStrokeWidth: 2.5,
	}
	if chart.allNonNegative() {
		option.YAxis[0].Min = charts.Ptr(0.0)
	}
	return option
}

func (chart *Chart) comboHasBarsOnSignedRange() bool {
	hasBar := false
	hasNegative := false
	for _, series := range chart.series {
		hasBar = hasBar || series.spec.Mark == MarkBar
		for _, value := range series.values {
			if !isMissing(value) && value < 0 {
				hasNegative = true
			}
		}
	}
	return hasBar && hasNegative
}

func (chart *Chart) drawScatter(painter *charts.Painter, theme charts.ColorPalette, showAxes, showLabels bool) error {
	if len(chart.xValues) == 0 {
		return fmt.Errorf("scatter chart has no x values")
	}

	textStyle := charts.NewFontStyleWithSize(11).WithColor(theme.GetLabelTextColor())
	mutedStyle := charts.NewFontStyleWithSize(10).WithColor(theme.GetXAxisTextColor())
	top := 16

	bottomPadding := 18
	if showAxes {
		bottomPadding += 28
	}
	showLegend := displayValue(chart.spec.Legend, len(chart.series) > 1)
	legendRows := 0
	if showLegend {
		legendRows = chart.scatterLegendRows(painter, textStyle, max(1, painter.Width()-86))
		bottomPadding += legendRows * 20
	}
	left := 20
	if showAxes {
		left = 62
	}
	right := painter.Width() - 24
	bottom := painter.Height() - bottomPadding
	if right-left < 40 || bottom-top < 40 {
		return fmt.Errorf("image is too small for scatter chart")
	}

	minX, maxX := chart.xValues[0], chart.xValues[0]
	minY, maxY := chart.valueRange(nil)
	for _, value := range chart.xValues[1:] {
		minX = math.Min(minX, value)
		maxX = math.Max(maxX, value)
	}
	xAxis := newGraphicalValueAxisWithIntervals(minX, maxX, 4, false)
	yAxis := newGraphicalValueAxisWithIntervals(minY, maxY, 4, false)
	mapX := func(value float64) int {
		return left + int(math.Round((value-xAxis.minimum)/(xAxis.maximum-xAxis.minimum)*float64(right-left)))
	}
	mapY := func(value float64) int {
		return bottom - int(math.Round((value-yAxis.minimum)/(yAxis.maximum-yAxis.minimum)*float64(bottom-top)))
	}

	if showAxes {
		axisColor := theme.GetXAxisStrokeColor()
		gridColor := theme.GetAxisSplitLineColor()
		painter.LineStroke([]charts.Point{{X: left, Y: top}, {X: left, Y: bottom}, {X: right, Y: bottom}}, axisColor, 1)
		for _, yValue := range yAxis.ticks() {
			y := mapY(yValue)
			painter.LineStroke([]charts.Point{{X: left, Y: y}, {X: right, Y: y}}, gridColor, 0.5)
			yLabel := formatValue(yValue)
			yLabelWidth := painter.MeasureText(yLabel, 0, mutedStyle).Width()
			painter.Text(yLabel, max(0, left-yLabelWidth-8), y+4, 0, mutedStyle)
		}
		for _, xValue := range xAxis.ticks() {
			x := mapX(xValue)
			xLabel := formatValue(xValue)
			xLabelWidth := painter.MeasureText(xLabel, 0, mutedStyle).Width()
			xLabelX := min(right-xLabelWidth, max(left, x-(xLabelWidth>>1)))
			painter.Text(xLabel, xLabelX, bottom+18, 0, mutedStyle)
		}
	}

	var labelBoxes []charts.Box
	for seriesIndex, series := range chart.series {
		color := theme.GetSeriesColor(seriesIndex)
		for pointIndex, value := range series.values {
			if value == math.MaxFloat64 {
				continue
			}
			x := mapX(chart.xValues[pointIndex])
			y := mapY(value)
			painter.Circle(5, x, y, color, color, 1)
			if showLabels {
				text := formatValue(value)
				textBox := painter.MeasureText(text, 0, textStyle)
				candidates := []charts.Box{
					charts.NewBox(x+7, y-textBox.Height(), x+7+textBox.Width(), y),
					charts.NewBox(x-textBox.Width()-7, y-textBox.Height(), x-7, y),
					charts.NewBox(x+7, y+4, x+7+textBox.Width(), y+4+textBox.Height()),
				}
				for _, box := range candidates {
					if box.Left < left || box.Right > right || box.Top < top || box.Bottom > bottom || overlapsAny(box, labelBoxes) {
						continue
					}
					painter.Text(text, box.Left, box.Bottom, 0, textStyle)
					labelBoxes = append(labelBoxes, box)
					break
				}
			}
		}
	}

	if showLegend {
		x := left
		y := painter.Height() - (legendRows-1)*20 - 8
		for seriesIndex, series := range chart.series {
			itemWidth := painter.MeasureText(series.spec.Label, 0, textStyle).Width() + 34
			if x > left && x+itemWidth > right {
				x = left
				y += 20
			}
			color := theme.GetSeriesColor(seriesIndex)
			painter.Circle(3, x, y-4, color, color, 1)
			painter.Text(series.spec.Label, x+8, y, 0, textStyle)
			x += itemWidth
		}
	}
	return nil
}

func (chart *Chart) scatterLegendRows(painter *charts.Painter, style charts.FontStyle, width int) int {
	rows := 1
	used := 0
	for _, series := range chart.series {
		itemWidth := painter.MeasureText(series.spec.Label, 0, style).Width() + 34
		if used > 0 && used+itemWidth > width {
			rows++
			used = 0
		}
		used += itemWidth
	}
	return rows
}

func overlapsAny(candidate charts.Box, existing []charts.Box) bool {
	for _, box := range existing {
		if candidate.Left < box.Right && candidate.Right > box.Left && candidate.Top < box.Bottom && candidate.Bottom > box.Top {
			return true
		}
	}
	return false
}

func (chart *Chart) escapedForSVG() *Chart {
	clone := chart.cloneForRender()
	clone.spec.Title = html.EscapeString(chart.spec.Title)
	clone.spec.Description = html.EscapeString(chart.spec.Description)
	clone.spec.Footer = html.EscapeString(chart.spec.Footer)
	clone.labels = make([]string, len(chart.labels))
	for index, label := range chart.labels {
		clone.labels[index] = html.EscapeString(label)
	}
	clone.series = make([]compiledSeries, len(chart.series))
	for index, series := range chart.series {
		clone.series[index] = series
		clone.series[index].spec.Label = html.EscapeString(series.spec.Label)
	}
	return clone
}

func (chart *Chart) cloneForRender() *Chart {
	clone := *chart
	clone.spec = cloneSpec(chart.spec)
	clone.labels = append([]string(nil), chart.labels...)
	clone.xValues = append([]float64(nil), chart.xValues...)
	clone.series = make([]compiledSeries, len(chart.series))
	for index, series := range chart.series {
		clone.series[index] = series
		clone.series[index].values = append([]float64(nil), series.values...)
	}
	return &clone
}

func (chart *Chart) theme() (charts.ColorPalette, error) {
	colorCount := len(chart.series)
	if chart.spec.Type == Pie || chart.spec.Type == Donut {
		colorCount = max(colorCount, len(chart.labels))
	}
	colors := make([]charts.Color, colorCount)
	for index := range colors {
		value := defaultColors[index%len(defaultColors)]
		if index < len(chart.series) {
			value = chart.series[index].spec.Color
		}
		color, err := colorHex(value)
		if err != nil {
			return nil, err
		}
		colors[index] = color
	}
	return charts.GetTheme(chart.spec.Theme).WithSeriesColors(colors), nil
}

func colorHex(value string) (charts.Color, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "var(--chart-") && strings.HasSuffix(value, ")") {
		indexText := strings.TrimSuffix(strings.TrimPrefix(value, "var(--chart-"), ")")
		index, err := strconv.Atoi(indexText)
		if err == nil && index >= 1 && index <= len(defaultColors) {
			value = defaultColors[index-1]
		}
	}
	if len(value) != 7 || value[0] != '#' {
		return charts.Color{}, fmt.Errorf("color %q must be #RRGGBB or var(--chart-N)", value)
	}
	decoded, err := strconv.ParseUint(value[1:], 16, 24)
	if err != nil {
		return charts.Color{}, fmt.Errorf("color %q must be #RRGGBB", value)
	}
	return charts.Color{R: uint8(decoded >> 16), G: uint8(decoded >> 8), B: uint8(decoded), A: 255}, nil
}

func (chart *Chart) seriesNames() []string {
	names := make([]string, len(chart.series))
	for index, series := range chart.series {
		names[index] = series.spec.Label
	}
	return names
}

func (chart *Chart) seriesValues() [][]float64 {
	values := make([][]float64, len(chart.series))
	for index, series := range chart.series {
		values[index] = append([]float64(nil), series.values...)
	}
	return values
}

func (chart *Chart) allNonNegative() bool {
	for _, series := range chart.series {
		for _, value := range series.values {
			if !isMissing(value) && value < 0 {
				return false
			}
		}
	}
	return true
}

func (chart *Chart) radarMaxima() []float64 {
	maxima := make([]float64, len(chart.labels))
	for pointIndex := range maxima {
		maxima[pointIndex] = chart.spec.Max
		if maxima[pointIndex] > 0 {
			continue
		}
		for seriesIndex := range chart.series {
			value := chart.series[seriesIndex].values[pointIndex]
			if !isMissing(value) {
				maxima[pointIndex] = math.Max(maxima[pointIndex], value)
			}
		}
		if maxima[pointIndex] <= 0 {
			maxima[pointIndex] = 1
		}
	}
	return maxima
}

func imageDimensions(options ImageOptions, defaultWidth, defaultHeight int) (int, int, error) {
	if options.Width == 0 {
		options.Width = defaultWidth
	}
	if options.Height == 0 {
		options.Height = defaultHeight
	}
	if options.Width < 320 || options.Width > 4096 {
		return 0, 0, fmt.Errorf("image width must be between 320 and 4096")
	}
	if options.Height < 240 || options.Height > 4096 {
		return 0, 0, fmt.Errorf("image height must be between 240 and 4096")
	}
	return options.Width, options.Height, nil
}
