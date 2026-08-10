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
	theme, err := chart.theme()
	if err != nil {
		return nil, err
	}
	if chart.spec.Type == Combo {
		option := chart.comboOption(theme, width, height, format)
		painter, err := charts.Render(option)
		if err != nil {
			return nil, fmt.Errorf("build combo chart: %w", err)
		}
		return painter.Bytes()
	}
	painter := charts.NewPainter(charts.PainterOptions{
		OutputFormat: format,
		Width:        width,
		Height:       height,
		Theme:        theme,
	})
	if err := chart.draw(painter, theme); err != nil {
		return nil, err
	}
	content, err := painter.Bytes()
	if err != nil {
		return nil, fmt.Errorf("encode %s chart: %w", format, err)
	}
	return content, nil
}

func (chart *Chart) draw(painter *charts.Painter, theme charts.ColorPalette) error {
	title := charts.TitleOption{Text: chart.spec.Title, Subtext: chart.spec.Description}
	legend := charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, len(chart.series) > 1)), SeriesNames: chart.seriesNames()}
	showAxes := displayValue(chart.spec.Axes, true)
	showLabels := displayValue(chart.spec.Labels, false)
	label := charts.SeriesLabel{Show: charts.Ptr(showLabels)}
	values := chart.seriesValues()

	switch chart.spec.Type {
	case Bar, Histogram:
		series := charts.NewSeriesListBar(values, charts.BarSeriesOption{Names: chart.seriesNames(), Label: label})
		option := charts.NewBarChartOptionWithSeries(series)
		option.Theme = theme
		option.Title = title
		option.Legend = legend
		option.Horizontal = chart.spec.Orientation == Horizontal
		option.CategoryAxis.Labels = chart.labels
		option.CategoryAxis.Show = charts.Ptr(showAxes)
		option.ValueAxis[0].Show = charts.Ptr(showAxes)
		option.ValueAxis[0].PreferNiceIntervals = charts.Ptr(true)
		if chart.allNonNegative() {
			option.ValueAxis[0].Min = charts.Ptr(0.0)
		}
		if chart.spec.Layout == Normalized {
			option.ValueAxis[0].Max = charts.Ptr(100.0)
		}
		option.StackSeries = charts.Ptr(chart.spec.Layout == Stacked || chart.spec.Layout == Normalized)
		option.RoundedBarCaps = charts.Ptr(true)
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
		points := make([][][]float64, len(values))
		for seriesIndex, series := range values {
			points[seriesIndex] = make([][]float64, len(series))
			for pointIndex, value := range series {
				points[seriesIndex][pointIndex] = []float64{value}
			}
		}
		series := charts.NewSeriesListScatterMultiValue(points, charts.ScatterSeriesOption{Names: chart.seriesNames(), Label: label})
		option := charts.NewScatterChartOptionWithSeries(series)
		option.Theme = theme
		option.Title = title
		option.Legend = legend
		option.XAxis.Labels = chart.labels
		option.XAxis.Show = charts.Ptr(showAxes)
		option.YAxis[0].Show = charts.Ptr(showAxes)
		option.Symbol = charts.Symbol{Shape: charts.SymbolCircle, Size: 3}
		return painter.ScatterChart(option)
	case Pie:
		series := charts.NewSeriesListPie(values[0], charts.PieSeriesOption{Names: chart.labels, Label: label})
		option := charts.PieChartOption{
			Theme:      theme,
			SeriesList: series,
			Title:      title,
			Legend:     charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, true)), SeriesNames: chart.labels},
		}
		return painter.PieChart(option)
	case Donut:
		series := charts.NewSeriesListDoughnut(values[0], charts.DoughnutSeriesOption{Names: chart.labels, Label: label})
		option := charts.DoughnutChartOption{
			Theme:        theme,
			SeriesList:   series,
			Title:        title,
			Legend:       charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, true)), SeriesNames: chart.labels},
			CenterValues: "sum",
		}
		return painter.DoughnutChart(option)
	case Heatmap:
		rows := make([][]float64, len(chart.labels))
		for rowIndex := range rows {
			rows[rowIndex] = make([]float64, len(chart.series))
			for seriesIndex := range chart.series {
				rows[rowIndex][seriesIndex] = chart.series[seriesIndex].values[rowIndex]
			}
		}
		option := charts.NewHeatMapOptionWithData(rows)
		option.Theme = theme
		option.Title = title
		option.XAxis.Labels = chart.seriesNames()
		option.YAxis.Labels = chart.labels
		option.ValuesLabel = label
		return painter.HeatMapChart(option)
	case Radar:
		maxima := make([]float64, len(chart.labels))
		for pointIndex := range maxima {
			maxima[pointIndex] = chart.spec.Max
			if maxima[pointIndex] == 0 {
				for seriesIndex := range chart.series {
					maxima[pointIndex] = math.Max(maxima[pointIndex], chart.series[seriesIndex].values[pointIndex])
				}
			}
		}
		series := charts.NewSeriesListRadar(values, charts.RadarSeriesOption{Names: chart.seriesNames(), Label: label})
		option := charts.RadarChartOption{
			Theme:           theme,
			SeriesList:      series,
			RadarIndicators: charts.NewRadarIndicators(chart.labels, maxima),
			Title:           title,
			Legend:          legend,
		}
		return painter.RadarChart(option)
	case Funnel:
		series := charts.NewSeriesListFunnel(values[0], charts.FunnelSeriesOption{Names: chart.labels, Label: label})
		option := charts.FunnelChartOption{
			Theme:      theme,
			SeriesList: series,
			Title:      title,
			Legend:     charts.LegendOption{Show: charts.Ptr(displayValue(chart.spec.Legend, false)), SeriesNames: chart.labels},
		}
		return painter.FunnelChart(option)
	default:
		return fmt.Errorf("chart type %q has no output engine", chart.spec.Type)
	}
}

func (chart *Chart) comboOption(theme charts.ColorPalette, width, height int, format string) charts.ChartOption {
	series := make(charts.GenericSeriesList, len(chart.series))
	for index, item := range chart.series {
		chartType := charts.ChartTypeLine
		if item.spec.Mark == MarkBar {
			chartType = charts.ChartTypeBar
		}
		series[index] = charts.GenericSeries{Type: chartType, Values: item.values, Name: item.spec.Label}
	}
	showAxes := displayValue(chart.spec.Axes, true)
	option := charts.ChartOption{
		OutputFormat:    format,
		Width:           width,
		Height:          height,
		Theme:           theme,
		Title:           charts.TitleOption{Text: chart.spec.Title, Subtext: chart.spec.Description},
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

func (chart *Chart) theme() (charts.ColorPalette, error) {
	colors := make([]charts.Color, len(chart.series))
	for index, series := range chart.series {
		color, err := colorHex(series.spec.Color)
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
			if value != math.MaxFloat64 && value < 0 {
				return false
			}
		}
	}
	return true
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
