package chartmux

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/go-analyze/charts"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestParseSpecIsStrictAndVersioned(t *testing.T) {
	_, err := ParseSpec(strings.NewReader(`{"version":1,"type":"line","unknown":true}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
	_, err = ParseSpec(strings.NewReader(`{"type":"line"}`))
	if err == nil || !strings.Contains(err.Error(), "missing version") {
		t.Fatalf("missing version error = %v", err)
	}
}

func TestWriteJSONReturnsResolvedVersionedSpec(t *testing.T) {
	spec, _ := Demo("line")
	spec.Theme = ""
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := chart.WriteJSON(&output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, expected := range []string{`"$schema": "` + SchemaURL + `"`, `"version": 1`, `"theme": "light"`, `"axes": {`, `"labels": {`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("resolved JSON is missing %s:\n%s", expected, text)
		}
	}
}

func TestEmbeddedSchema(t *testing.T) {
	if !strings.Contains(string(SchemaJSON()), `"$id": "`+SchemaURL+`"`) {
		t.Fatal("embedded schema does not contain its public identifier")
	}
}

func TestEveryResolvedDemoMatchesEmbeddedSchema(t *testing.T) {
	var schemaDocument any
	if err := json.Unmarshal(SchemaJSON(), &schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource(SchemaURL, schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiledSchema, err := compiler.Compile(SchemaURL)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range DemoNames() {
		t.Run(name, func(t *testing.T) {
			spec, err := Demo(name)
			if err != nil {
				t.Fatal(err)
			}
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := chart.WriteJSON(&output); err != nil {
				t.Fatal(err)
			}
			var document any
			if err := json.Unmarshal(output.Bytes(), &document); err != nil {
				t.Fatal(err)
			}
			if err := compiledSchema.Validate(document); err != nil {
				t.Fatalf("resolved chart does not match embedded schema: %v\n%s", err, output.String())
			}
			parsed, err := ParseSpec(bytes.NewReader(output.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := New(parsed); err != nil {
				t.Fatalf("resolved chart does not round-trip: %v", err)
			}
		})
	}
}

func TestEmbeddedSchemaRejectsInvalidSeriesColors(t *testing.T) {
	var schemaDocument any
	if err := json.Unmarshal(SchemaJSON(), &schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource(SchemaURL, schemaDocument); err != nil {
		t.Fatal(err)
	}
	compiledSchema, err := compiler.Compile(SchemaURL)
	if err != nil {
		t.Fatal(err)
	}
	document := map[string]any{
		"version": 1,
		"type":    "line",
		"xAxis":   map[string]any{"dataKey": "period"},
		"series":  []any{map[string]any{"dataKey": "value", "color": "red"}},
		"data":    []any{map[string]any{"period": "Q1", "value": 1}},
	}
	if err := compiledSchema.Validate(document); err == nil {
		t.Fatal("embedded schema accepted a color outside #RRGGBB and var(--chart-N)")
	}
}

func TestSVGAndHTMLEscapeChartText(t *testing.T) {
	payload := `</text><script>alert(1)</script>`
	spec := Spec{
		Version:     SpecVersion,
		Type:        Line,
		Title:       "R&D " + payload,
		Description: payload,
		Footer:      payload,
		XAxis:       AxisSpec{DataKey: "x"},
		Series: []SeriesSpec{
			{DataKey: "y", Label: payload},
			{DataKey: "z", Label: "Safe"},
		},
		Data:        []Row{{"x": payload, "y": 1, "z": 2}, {"x": "safe", "y": 2, "z": 3}},
		Annotations: []Annotation{{Text: payload}},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var svg bytes.Buffer
	if err := chart.WriteSVG(&svg, ImageOptions{Width: 800, Height: 480}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(svg.String(), "<script") || !strings.Contains(svg.String(), "&lt;script&gt;") {
		t.Fatalf("SVG contains executable or missing escaped chart text:\n%s", svg.String())
	}
	if !strings.Contains(svg.String(), "R&amp;D") || strings.Contains(svg.String(), "R&amp;amp;D") {
		t.Fatalf("SVG chart text was not escaped exactly once:\n%s", svg.String())
	}
	var html bytes.Buffer
	if err := chart.WriteHTML(&html, HTMLOptions{Width: 800, Height: 480}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html.String(), "<script") || !strings.Contains(html.String(), "Content-Security-Policy") {
		t.Fatalf("HTML contains executable chart text or lacks CSP:\n%s", html.String())
	}
	var second bytes.Buffer
	if err := chart.WriteSVG(&second, ImageOptions{Width: 800, Height: 480}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(svg.Bytes(), second.Bytes()) {
		t.Fatal("rendering the same chart twice produced different SVG output")
	}
}

func TestInlineChartLabelsRejectControlCharacters(t *testing.T) {
	seriesLabel, _ := Demo("line")
	seriesLabel.Series[0].Label = "Revenue\x1b[31m"
	if _, err := New(seriesLabel); err == nil || !strings.Contains(err.Error(), "control character") {
		t.Fatalf("series label control error = %v", err)
	}

	axisLabel, _ := Demo("line")
	axisLabel.Data[0][axisLabel.XAxis.DataKey] = "Jan\x1b[31m"
	if _, err := New(axisLabel); err == nil || !strings.Contains(err.Error(), "control character") {
		t.Fatalf("axis label control error = %v", err)
	}
}

func TestAnnotationsValidateAndResolveDeterministically(t *testing.T) {
	spec, err := Demo("line")
	if err != nil {
		t.Fatal(err)
	}
	index := 1
	spec.Annotations = []Annotation{{Text: "Growth accelerated", DataIndex: &index, Series: "desktop"}}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if got := chart.ResolvedSpec().Annotations[0].Position; got != AnnotationBottom {
		t.Fatalf("resolved annotation position = %q, want %q", got, AnnotationBottom)
	}
	index = 99
	if got := *chart.Spec().Annotations[0].DataIndex; got != 1 {
		t.Fatalf("chart retained caller-owned annotation pointer: dataIndex = %d", got)
	}

	tests := []struct {
		name       string
		annotation Annotation
		contains   string
	}{
		{name: "empty", annotation: Annotation{}, contains: "must not be empty"},
		{name: "index", annotation: Annotation{Text: "note", DataIndex: intPointer(99)}, contains: "dataIndex"},
		{name: "series", annotation: Annotation{Text: "note", Series: "missing"}, contains: "unknown series"},
		{name: "position", annotation: Annotation{Text: "note", Position: "middle"}, contains: "position"},
		{name: "color", annotation: Annotation{Text: "note", Color: "red"}, contains: "#RRGGBB"},
		{name: "control", annotation: Annotation{Text: "note\x1b[31m"}, contains: "control"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalid, _ := Demo("line")
			invalid.Annotations = []Annotation{test.annotation}
			if _, err := New(invalid); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("annotation error = %v, want containing %q", err, test.contains)
			}
		})
	}

	histogram, _ := Demo("histogram")
	histogram.Annotations = []Annotation{{Text: "note", DataIndex: intPointer(0)}}
	if _, err := New(histogram); err == nil || !strings.Contains(err.Error(), "histogram bins") {
		t.Fatalf("histogram annotation error = %v", err)
	}
}

func TestAnnotationsRenderWithoutTerminalOverflow(t *testing.T) {
	spec, _ := Demo("grouped-bar")
	index := 5
	spec.Title = "Global Infrastructure & Energy Transition — Financing Overview"
	spec.Description = "A long investment committee description that must wrap cleanly instead of colliding with the plot or legend."
	spec.Annotations = []Annotation{
		{Text: "Mobile visitors accelerated materially during the latest reporting period — 投資委員会向け注記.", Position: AnnotationTop, DataIndex: &index, Series: "mobile"},
		{Text: "Illustrative analysis only; figures may not sum due to rounding.", Position: AnnotationBottom},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := chart.Terminal(TerminalOptions{Width: 64, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(terminal)
	for index, line := range strings.Split(plain, "\n") {
		if width := ansi.StringWidth(line); width > 64 {
			t.Fatalf("terminal line %d is %d cells wide:\n%s", index+1, width, terminal)
		}
	}
	for _, expected := range []string{"Jun · Mobile", "Illustrative analysis"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("terminal annotation is missing %q:\n%s", expected, terminal)
		}
	}
	var svg bytes.Buffer
	if err := chart.WriteSVG(&svg, ImageOptions{Width: 800, Height: 600}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg.String(), "Mobile visitors accelerated") || !strings.Contains(svg.String(), "Illustrative analysis") {
		t.Fatalf("SVG annotations are missing:\n%s", svg.String())
	}
}

func TestGraphicalTextLayoutReservesSpaceInRasterAndVectorOutputs(t *testing.T) {
	spec, _ := Demo("annotated-bar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	theme, err := chart.theme()
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{charts.ChartOutputPNG, charts.ChartOutputSVG} {
		painter := charts.NewPainter(charts.PainterOptions{OutputFormat: format, Width: 1200, Height: 720, Theme: theme})
		layout, err := chart.graphicalTextLayout(painter, theme, 1200, 720)
		if err != nil {
			t.Fatal(err)
		}
		if layout.topHeight < 60 || layout.bottomHeight < 40 {
			t.Fatalf("%s text layout did not reserve space: top=%d bottom=%d", format, layout.topHeight, layout.bottomHeight)
		}
	}
}

func TestPNGRenderingDoesNotMutateChartText(t *testing.T) {
	spec, _ := Demo("annotated-bar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	before := chart.Spec()
	var pngOutput bytes.Buffer
	if err := chart.WritePNG(&pngOutput, ImageOptions{Width: 1200, Height: 720}); err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(bytes.NewReader(pngOutput.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if pixels := nonWhitePixels(image, 0, 0, 500, 100); pixels < 100 {
		t.Fatalf("PNG header contains only %d non-white pixels; chart text was not rendered", pixels)
	}
	if pixels := nonWhitePixels(image, 0, 650, 1000, 720); pixels < 100 {
		t.Fatalf("PNG footer contains only %d non-white pixels; annotations were not rendered", pixels)
	}
	after := chart.Spec()
	if after.Title != before.Title || after.Description != before.Description || after.Footer != before.Footer || len(after.Annotations) != len(before.Annotations) {
		t.Fatalf("PNG rendering mutated chart text: before=%+v after=%+v", before, after)
	}
	var svg bytes.Buffer
	if err := chart.WriteSVG(&svg, ImageOptions{Width: 1200, Height: 720}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg.String(), "Illustrative Operating Performance") {
		t.Fatal("chart text disappeared after PNG rendering")
	}
}

func TestComboAnnotationsUseOpaqueReservedBands(t *testing.T) {
	spec, _ := Demo("combo")
	spec.Annotations = []Annotation{{Text: "Revenue remained ahead of the underwriting case."}}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := chart.WritePNG(&output, ImageOptions{Width: 1200, Height: 720}); err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range [][2]int{{0, 0}, {1199, 0}, {0, 719}, {1199, 719}} {
		_, _, _, alpha := image.At(point[0], point[1]).RGBA()
		if alpha != 0xffff {
			t.Fatalf("combo frame pixel %v has alpha %#x, want opaque", point, alpha)
		}
	}
}

func TestDenseTextFailsClearlyInsteadOfCollapsingPlot(t *testing.T) {
	spec, _ := Demo("line")
	for index := 0; index < maxAnnotations; index++ {
		spec.Annotations = append(spec.Annotations, Annotation{Text: strings.Repeat("Long investment committee annotation ", 5)})
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = chart.WritePNG(&output, ImageOptions{Width: 320, Height: 240})
	if err == nil || !strings.Contains(err.Error(), "too short for chart text") {
		t.Fatalf("dense text error = %v", err)
	}
}

func TestChartLabelsCannotInjectEscapeSequences(t *testing.T) {
	spec, _ := Demo("line")
	spec.Data[0]["month"] = "Jan\x1b]52;c;payload\a"
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "control character") {
		t.Fatalf("chart label control error = %v", err)
	}
}

func nonWhitePixels(image interface{ At(int, int) color.Color }, left, top, right, bottom int) int {
	count := 0
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			red, green, blue, alpha := image.At(x, y).RGBA()
			if alpha > 0 && (red < 0xf000 || green < 0xf000 || blue < 0xf000) {
				count++
			}
		}
	}
	return count
}

func TestRadarMissingValuesDoNotDestroyScale(t *testing.T) {
	spec, _ := Demo("radar")
	spec.Max = 0
	spec.Data[0]["desktop"] = nil
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	maxima := chart.radarMaxima()
	if maxima[0] != 91 {
		t.Fatalf("first radar maximum = %v, want 91", maxima[0])
	}
	var svg bytes.Buffer
	if err := chart.WriteSVG(&svg, ImageOptions{Width: 800, Height: 480}); err != nil {
		t.Fatal(err)
	}
}

func TestRadarRejectsValuesAboveExplicitMaximum(t *testing.T) {
	spec, _ := Demo("radar")
	spec.Max = 50
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "exceeds radar max") {
		t.Fatalf("radar max error = %v", err)
	}
}

func intPointer(value int) *int {
	return &value
}

func TestGraphicalDisplayOptionsChangeOutput(t *testing.T) {
	render := func(t *testing.T, spec Spec) string {
		t.Helper()
		chart, err := New(spec)
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := chart.WriteSVG(&output, ImageOptions{Width: 800, Height: 480}); err != nil {
			t.Fatal(err)
		}
		return output.String()
	}

	combo, err := Demo("combo")
	if err != nil {
		t.Fatal(err)
	}
	withoutLabels := render(t, combo)
	combo.Labels = &DisplaySpec{Show: true}
	if withLabels := render(t, combo); withLabels == withoutLabels {
		t.Fatal("combo value labels did not change SVG output")
	}

	heatmap, err := Demo("heatmap")
	if err != nil {
		t.Fatal(err)
	}
	withAxes := render(t, heatmap)
	heatmap.Axes = &DisplaySpec{Show: false}
	if withoutAxes := render(t, heatmap); withoutAxes == withAxes {
		t.Fatal("hidden heatmap axes did not change SVG output")
	}

	line, err := Demo("line")
	if err != nil {
		t.Fatal(err)
	}
	line.Footer = "first footer"
	firstFooter := render(t, line)
	line.Footer = "second footer"
	secondFooter := render(t, line)
	if firstFooter == secondFooter || !strings.Contains(firstFooter, "first footer") {
		t.Fatal("footer was not rendered into SVG output")
	}
}

func TestScatterUsesNumericXValues(t *testing.T) {
	spec := Spec{
		Version: SpecVersion,
		Type:    Scatter,
		XAxis:   AxisSpec{DataKey: "x", Kind: "number"},
		Series:  []SeriesSpec{{DataKey: "y"}},
		Data: []Row{
			{"x": 0, "y": 0},
			{"x": 10, "y": 50},
			{"x": 100, "y": 100},
		},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var svg bytes.Buffer
	if err := chart.WriteSVG(&svg, ImageOptions{Width: 800, Height: 480}); err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`<circle cx="(\d+)"`).FindAllStringSubmatch(svg.String(), -1)
	if len(matches) != 3 {
		t.Fatalf("scatter SVG circles = %d, want 3\n%s", len(matches), svg.String())
	}
	positions := make([]int, len(matches))
	for index, match := range matches {
		positions[index], err = strconv.Atoi(match[1])
		if err != nil {
			t.Fatal(err)
		}
	}
	firstGap := positions[1] - positions[0]
	secondGap := positions[2] - positions[1]
	if firstGap <= 0 || secondGap < firstGap*5 {
		t.Fatalf("scatter x positions are not proportional: %v", positions)
	}

	terminal, err := chart.Terminal(TerminalOptions{Width: 84, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(ansi.Strip(terminal), "●") != 3 {
		t.Fatalf("terminal scatter did not render three independent points:\n%s", terminal)
	}
}

func TestNormalizedLayoutRejectsInvalidRows(t *testing.T) {
	spec := Spec{
		Version: SpecVersion,
		Type:    Bar,
		Layout:  Normalized,
		XAxis:   AxisSpec{DataKey: "x"},
		Series:  []SeriesSpec{{DataKey: "a"}, {DataKey: "b"}},
		Data:    []Row{{"x": "mixed", "a": 2, "b": -1}},
	}
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("mixed-sign normalized row error = %v", err)
	}
	spec.Data = []Row{{"x": "zero", "a": 0, "b": 0}}
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "no positive values") {
		t.Fatalf("zero-total normalized row error = %v", err)
	}
	spec.Data = []Row{{"x": "missing", "a": nil, "b": nil}}
	if _, err := New(spec); err != nil {
		t.Fatalf("all-missing normalized row should remain a gap: %v", err)
	}
}

func TestHistogramRejectsIncompleteXAxis(t *testing.T) {
	spec, err := Demo("histogram")
	if err != nil {
		t.Fatal(err)
	}
	spec.XAxis = AxisSpec{Kind: "number"}
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "requires xAxis.dataKey") {
		t.Fatalf("incomplete histogram xAxis error = %v", err)
	}
}

func TestSignedAndMissingCartesianValues(t *testing.T) {
	spec := Spec{
		Version: SpecVersion,
		Type:    Line,
		XAxis:   AxisSpec{DataKey: "month"},
		Series:  []SeriesSpec{{DataKey: "change"}},
		Data: []Row{
			{"month": "Jan", "change": -12},
			{"month": "Feb", "change": nil},
			{"month": "Mar", "change": 18},
		},
	}
	if _, err := New(spec); err != nil {
		t.Fatal(err)
	}
	spec.Type = Pie
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("pie negative error = %v", err)
	}
}

func TestSignedBarsUseDivergingNativeAxis(t *testing.T) {
	spec := Spec{
		Version: SpecVersion,
		Type:    Bar,
		XAxis:   AxisSpec{DataKey: "period"},
		Series:  []SeriesSpec{{DataKey: "variance", Label: "Variance"}},
		Data: []Row{
			{"period": "Q1", "variance": -18},
			{"period": "Q2", "variance": 12},
		},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "-18") || !strings.Contains(plain, "│") || !strings.Contains(plain, "Q2 · Variance") {
		t.Fatalf("signed bar chart is not a diverging view:\n%s", output)
	}
}

func TestSignedBarScalePrioritizesZeroOverACollidingMinimum(t *testing.T) {
	spec := Spec{
		Version: SpecVersion,
		Type:    Bar,
		XAxis:   AxisSpec{DataKey: "period"},
		Series:  []SeriesSpec{{DataKey: "variance", Label: "Long variance"}},
		Data: []Row{
			{"period": "Downside", "variance": -5},
			{"period": "Upside", "variance": 100},
		},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 30, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	scale := strings.Split(ansi.Strip(output), "\n")[0]
	if strings.Contains(scale, "-0") || !strings.Contains(scale, "0") {
		t.Fatalf("signed scale did not preserve an unambiguous zero label:\n%s", output)
	}
	if strings.Contains(scale, "-5") {
		t.Fatalf("signed scale should omit a minimum that cannot fit beside zero:\n%s", output)
	}
}

func TestHorizontalBarScaleSkipsCollidingMiddleLabel(t *testing.T) {
	const maximum = 850_000_000_000.0
	spec := Spec{
		Version:     SpecVersion,
		Type:        Bar,
		Orientation: Horizontal,
		XAxis:       AxisSpec{DataKey: "period"},
		Series:      []SeriesSpec{{DataKey: "value", Label: "Long value"}},
		Data:        []Row{{"period": "Long category", "value": maximum}},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 30, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	scale := strings.Split(ansi.Strip(output), "\n")[0]
	maximumText := formatValue(maximum)
	if !strings.Contains(scale, maximumText) {
		t.Fatalf("horizontal scale lost its maximum label:\n%s", output)
	}
	middleText := formatValue(maximum / 2)
	validScale := regexp.MustCompile(`^0[ ┄]+(?:` + regexp.QuoteMeta(middleText) + `[ ┄]+)?` + regexp.QuoteMeta(maximumText) + `$`)
	if !validScale.MatchString(strings.TrimSpace(scale)) {
		t.Fatalf("horizontal scale fused its middle and maximum labels:\n%s", output)
	}
}

func TestStackedBarsUseContiguousPatternCells(t *testing.T) {
	spec := Spec{
		Version: SpecVersion,
		Type:    Bar,
		Layout:  Stacked,
		XAxis:   AxisSpec{DataKey: "period"},
		Series: []SeriesSpec{
			{DataKey: "revenue", Label: "Revenue"},
			{DataKey: "cost", Label: "Cost"},
		},
		Data: []Row{{"period": "FY25", "revenue": 37, "cost": 63}},
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.terminalBarsWithState(40, 10, terminalRenderState{})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if strings.ContainsAny(plain, "▁▂▃▅▆▇") {
		t.Fatalf("stacked bars contain fractional internal gaps:\n%s", plain)
	}
	if !strings.Contains(plain, "█") || !strings.Contains(plain, "▓") {
		t.Fatalf("stacked series are not represented by stable fill patterns:\n%s", plain)
	}
}

func TestTerminalChartsUseAConsistentLayeredGrid(t *testing.T) {
	for _, name := range []string{"line", "area", "grouped-bar", "stacked-bar", "combo"} {
		t.Run(name, func(t *testing.T) {
			spec, _ := Demo(name)
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
			if err != nil {
				t.Fatal(err)
			}
			plain := ansi.Strip(output)
			if !strings.Contains(plain, "┄") {
				t.Fatalf("%s chart has no terminal-native grid:\n%s", name, plain)
			}
			if !strings.Contains(plain, "└") {
				t.Fatalf("%s chart has no coherent axis baseline:\n%s", name, plain)
			}
		})
	}
}

func TestVerticalBarsUseAlignedCaps(t *testing.T) {
	spec, _ := Demo("stacked-bar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "▄") {
		t.Fatalf("stacked bars have no deliberate cap treatment:\n%s", plain)
	}
	for _, line := range strings.Split(plain, "\n") {
		if strings.Contains(line, "▄") && strings.ContainsAny(line, "▁▂▃▅▆▇") {
			t.Fatalf("bar cap row contains incoherent fractional glyphs:\n%s", plain)
		}
	}
}

func TestRadarTerminalUsesActualRadialGeometry(t *testing.T) {
	spec, _ := Demo("radar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if strings.Contains(plain, "Reach      Desktop") {
		t.Fatalf("radar chart fell back to a comparative bar table:\n%s", plain)
	}
	for _, label := range []string{"Reach", "Engagement", "Conversion", "Retention", "Revenue"} {
		if !strings.Contains(plain, label) {
			t.Fatalf("radar chart is missing metric %q:\n%s", label, plain)
		}
	}
	if !strings.ContainsAny(plain, "·•┄") || !strings.ContainsAny(plain, "●◆") {
		t.Fatalf("radar chart has no radial grid and series geometry:\n%s", plain)
	}
}

func TestHeatmapUsesACompactMatrixAndDensityKey(t *testing.T) {
	spec, _ := Demo("heatmap")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.terminalHeatmapWithState(64, 14, terminalRenderState{})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	for _, expected := range []string{"Morning", "Mon │", "Low", "░▒▓█", "High"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("heatmap is missing matrix affordance %q:\n%s", expected, plain)
		}
	}
	for _, line := range strings.Split(plain, "\n") {
		if ansi.StringWidth(line) > 64 {
			t.Fatalf("heatmap row is %d columns wide:\n%s", ansi.StringWidth(line), plain)
		}
	}
}

func TestFunnelShowsStageToStageConversionWhenSpaceAllows(t *testing.T) {
	spec, _ := Demo("funnel")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.terminalFunnelWithState(64, 14, terminalRenderState{})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	for _, expected := range []string{"Visitors", "Signups", "↓  55.0%", "↓  57.6%", "↓  53.9%"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("funnel is missing conversion context %q:\n%s", expected, plain)
		}
	}
}

func TestTerminalInspectionIsBoundedAndContextual(t *testing.T) {
	spec, _ := Demo("line")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{
		Width:       80,
		Height:      14,
		Inspect:     true,
		FocusIndex:  1,
		FocusSeries: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	for _, expected := range []string{"Feb", "Desktop", "305", "┊"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("inspection output is missing %q:\n%s", expected, plain)
		}
	}
	if _, err := chart.Terminal(TerminalOptions{
		Width:       80,
		Height:      14,
		Inspect:     true,
		FocusIndex:  10_000,
		FocusSeries: 10_000,
	}); err != nil {
		t.Fatalf("out-of-range inspection should clamp safely: %v", err)
	}
}

func TestDataBoundTerminalAnnotationMarksThePlot(t *testing.T) {
	spec, _ := Demo("line")
	index := 1
	spec.Annotations = []Annotation{{
		Text:      "Inflection",
		DataIndex: &index,
		Series:    "desktop",
		Color:     "#F59E0B",
	}}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "✦") || !strings.Contains(plain, "Feb · Desktop") {
		t.Fatalf("data-bound annotation is not connected to its plot mark:\n%s", plain)
	}
}

func TestDataBoundAnnotationsMarkEveryNativePlot(t *testing.T) {
	demo := func(name string) Spec {
		t.Helper()
		spec, err := Demo(name)
		if err != nil {
			t.Fatal(err)
		}
		return spec
	}
	signed := Spec{
		Version: SpecVersion,
		Type:    Bar,
		XAxis:   AxisSpec{DataKey: "period"},
		Series:  []SeriesSpec{{DataKey: "variance", Label: "Variance"}},
		Data: []Row{
			{"period": "Q1", "variance": -18},
			{"period": "Q2", "variance": 12},
		},
	}
	tests := []struct {
		name   string
		spec   Spec
		series string
		point  int
	}{
		{name: "horizontal-bar", spec: demo("horizontal-bar"), series: "desktop"},
		{name: "signed-negative-bar", spec: signed, series: "variance"},
		{name: "signed-positive-bar", spec: signed, series: "variance", point: 1},
		{name: "heatmap", spec: demo("heatmap"), series: "morning"},
		{name: "funnel", spec: demo("funnel"), series: "users"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index := test.point
			test.spec.Annotations = []Annotation{{
				Text: "Material event", DataIndex: &index, Series: test.series, Color: "#F59E0B",
			}}
			chart, err := New(test.spec)
			if err != nil {
				t.Fatal(err)
			}
			output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(ansi.Strip(output), "✦") {
				t.Fatalf("data-bound annotation has no plot marker:\n%s", output)
			}
			for lineIndex, line := range strings.Split(output, "\n") {
				if lineWidth := ansi.StringWidth(line); lineWidth > 80 {
					t.Fatalf("annotated line %d is %d cells wide, want at most 80:\n%s", lineIndex+1, lineWidth, output)
				}
			}
		})
	}
}

func TestTinyPositiveSignedAnnotationPreservesTheValueColumn(t *testing.T) {
	index := 1
	chart, err := New(Spec{
		Version: SpecVersion,
		Type:    Bar,
		XAxis:   AxisSpec{DataKey: "period"},
		Series:  []SeriesSpec{{DataKey: "variance", Label: "Variance"}},
		Data: []Row{
			{"period": "Downside", "variance": -10},
			{"period": "Tiny upside", "variance": 1},
			{"period": "Upside", "variance": 100},
		},
		Annotations: []Annotation{{
			Text: "Small but material", DataIndex: &index, Series: "variance", Color: "#F59E0B",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 40, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	foundRow := false
	for _, line := range strings.Split(ansi.Strip(output), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "Tiny upside") {
			continue
		}
		foundRow = true
		if !strings.Contains(line, "✦") || !strings.HasSuffix(line, " 1") {
			t.Fatalf("tiny annotated positive bar lost its marker or value column:\n%s", output)
		}
		break
	}
	if !foundRow {
		t.Fatalf("tiny annotated positive row was not rendered:\n%s", output)
	}
}

func TestZeroHorizontalBarAnnotationMarksTheBaseline(t *testing.T) {
	index := 0
	chart, err := New(Spec{
		Version:     SpecVersion,
		Type:        Bar,
		Orientation: Horizontal,
		XAxis:       AxisSpec{DataKey: "period"},
		Series:      []SeriesSpec{{DataKey: "value", Label: "Value"}},
		Data: []Row{
			{"period": "Zero", "value": 0},
			{"period": "Maximum", "value": 100},
		},
		Annotations: []Annotation{{
			Text: "No activity", DataIndex: &index, Series: "value", Color: "#F59E0B",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 40, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	foundRow := false
	for _, line := range strings.Split(ansi.Strip(output), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "Zero · Value") {
			continue
		}
		foundRow = true
		if !strings.Contains(line, "✦") {
			t.Fatalf("zero-value horizontal annotation has no baseline marker:\n%s", output)
		}
		break
	}
	if !foundRow {
		t.Fatalf("zero-value horizontal row was not rendered:\n%s", output)
	}
}

func TestZeroSignedAnnotationMarksTheAxisWithoutPositiveRange(t *testing.T) {
	index := 1
	chart, err := New(Spec{
		Version: SpecVersion,
		Type:    Bar,
		XAxis:   AxisSpec{DataKey: "period"},
		Series:  []SeriesSpec{{DataKey: "variance", Label: "Variance"}},
		Data: []Row{
			{"period": "Downside", "variance": -100},
			{"period": "Breakeven", "variance": 0},
		},
		Annotations: []Annotation{{
			Text: "At plan", DataIndex: &index, Series: "variance", Color: "#F59E0B",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 40, Height: 8})
	if err != nil {
		t.Fatal(err)
	}
	foundRow := false
	for _, line := range strings.Split(ansi.Strip(output), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "Breakeven") {
			continue
		}
		foundRow = true
		if !strings.Contains(line, "✦") || !strings.HasSuffix(line, " 0") {
			t.Fatalf("zero signed annotation did not mark the axis and preserve its value:\n%s", output)
		}
		break
	}
	if !foundRow {
		t.Fatalf("zero signed row was not rendered:\n%s", output)
	}
}

func TestHorizontalBarsLabelEverySeriesRow(t *testing.T) {
	spec, _ := Demo("horizontal-bar")
	spec.Data = spec.Data[:1]
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.terminalBarsWithState(64, 8, terminalRenderState{})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	for _, label := range []string{"Jan · Desktop", "Jan · Mobile"} {
		if !strings.Contains(plain, label) {
			t.Fatalf("horizontal bars are missing %q:\n%s", label, plain)
		}
	}
}

func TestAreaTerminalUsesFillAndZeroBaseline(t *testing.T) {
	spec, _ := Demo("area")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "░") || !strings.Contains(plain, "▒") {
		t.Fatalf("area chart does not render accessible fill patterns:\n%s", plain)
	}
	if regexp.MustCompile(`(?m)^\s*-\d`).MatchString(plain) {
		t.Fatalf("non-negative area chart extends below zero:\n%s", plain)
	}
}

func TestComboLegendDistinguishesBarsFromLines(t *testing.T) {
	spec, _ := Demo("combo")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "█ Desktop") || !strings.Contains(plain, "◆·· Mobile") {
		t.Fatalf("combo legend does not distinguish bar and line marks:\n%s", plain)
	}
	axisColumn := -1
	for _, line := range strings.Split(plain, "\n") {
		if strings.Contains(line, "└") {
			break
		}
		axisIndex := strings.IndexAny(line, "│├┤")
		if axisIndex < 0 {
			continue
		}
		column := ansi.StringWidth(line[:axisIndex])
		if axisColumn < 0 {
			axisColumn = column
		} else if column != axisColumn {
			t.Fatalf("combo bars overwrite the y axis at column %d, want %d:\n%s", column, axisColumn, plain)
		}
	}
}

func TestNarrowComboRejectsOverflowingBarGroups(t *testing.T) {
	series := make([]SeriesSpec, 6)
	data := make([]Row, 6)
	for seriesIndex := range series {
		key := fmt.Sprintf("s%d", seriesIndex)
		series[seriesIndex] = SeriesSpec{DataKey: key, Label: key, Mark: MarkBar}
	}
	for pointIndex := range data {
		row := Row{"period": fmt.Sprintf("P%d", pointIndex)}
		for seriesIndex := range series {
			row[series[seriesIndex].DataKey] = float64((pointIndex + 1) * (seriesIndex + 1))
		}
		data[pointIndex] = row
	}
	chart, err := New(Spec{
		Version: SpecVersion,
		Type:    Combo,
		XAxis:   AxisSpec{DataKey: "period"},
		Series:  series,
		Data:    data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chart.Terminal(TerminalOptions{Width: MinTerminalWidth, Height: 14}); err == nil || !strings.Contains(err.Error(), "combo bar series") {
		t.Fatalf("narrow combo overflow error = %v", err)
	}
}

func TestPieAndDonutUseDistinctNativeShapes(t *testing.T) {
	pieSpec, _ := Demo("pie")
	pieChart, err := New(pieSpec)
	if err != nil {
		t.Fatal(err)
	}
	pieOutput, err := pieChart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	donutSpec, _ := Demo("donut")
	donutChart, err := New(donutSpec)
	if err != nil {
		t.Fatal(err)
	}
	donutOutput, err := donutChart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	pieLines := strings.Split(ansi.Strip(pieOutput), "\n")
	donutLines := strings.Split(ansi.Strip(donutOutput), "\n")
	if strings.Join(pieLines[1:], "\n") == strings.Join(donutLines[1:], "\n") {
		t.Fatal("pie and donut terminal charts use the same body")
	}
	if !strings.Contains(ansi.Strip(pieOutput), "▓") {
		t.Fatalf("pie segments are not pattern-distinct without color:\n%s", ansi.Strip(pieOutput))
	}
	if !strings.Contains(ansi.Strip(donutOutput), "Total") {
		t.Fatalf("donut chart does not expose its hollow-center total:\n%s", ansi.Strip(donutOutput))
	}
}

func TestZeroTotalPolarChartsShowAnEmptyState(t *testing.T) {
	for _, name := range []string{"pie", "donut"} {
		t.Run(name, func(t *testing.T) {
			spec, _ := Demo(name)
			for rowIndex := range spec.Data {
				spec.Data[rowIndex][spec.Series[0].DataKey] = 0
			}
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 14})
			if err != nil {
				t.Fatal(err)
			}
			plain := ansi.Strip(output)
			if !strings.Contains(plain, "No data") || strings.Contains(plain, "████") {
				t.Fatalf("zero-total %s chart rendered a false dominant slice:\n%s", name, plain)
			}
		})
	}
}

func TestHistogramTerminalUsesPresentationPrecision(t *testing.T) {
	spec, _ := Demo("histogram")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 80, Height: 14})
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`\d+\.\d{2}`).MatchString(ansi.Strip(output)) {
		t.Fatalf("histogram bin labels use excessive precision:\n%s", ansi.Strip(output))
	}
	for index := 1; index < len(chart.labels); index++ {
		previous := strings.Split(chart.labels[index-1], "–")
		current := strings.Split(chart.labels[index], "–")
		if len(previous) != 2 || len(current) != 2 || previous[1] != current[0] {
			t.Fatalf("histogram ranges are not contiguous: %q then %q", chart.labels[index-1], chart.labels[index])
		}
	}
}

func TestBarCategoryLabelsRespectUnicodeDisplayWidth(t *testing.T) {
	spec, _ := Demo("grouped-bar")
	for index := range spec.Data {
		spec.Data[index][spec.XAxis.DataKey] = []string{"東京", "München", "São Paulo", "서울", "Zürich", "دبي"}[index]
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.terminalBarsWithState(50, 12, terminalRenderState{})
	if err != nil {
		t.Fatal(err)
	}
	for index, line := range strings.Split(ansi.Strip(output), "\n") {
		if width := ansi.StringWidth(line); width > 50 {
			t.Fatalf("bar line %d is %d cells wide:\n%s", index+1, width, output)
		}
	}
}

func TestTerminalSeriesRemainDistinctWithoutColor(t *testing.T) {
	spec, _ := Demo("line")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "●") || !strings.Contains(plain, "◆") {
		t.Fatalf("series are not shape-distinct without color:\n%s", output)
	}
}

func TestTerminalColorResolvesChartPaletteTokens(t *testing.T) {
	r, g, b, a := terminalColor("var(--chart-2)").RGBA()
	if r != 0x6060 || g != 0xA5A5 || b != 0xFAFA || a != 0xFFFF {
		t.Fatalf("terminal palette color = #%04X%04X%04X alpha %04X", r, g, b, a)
	}
}

func TestMissingValueDoesNotBreakAnyDemoRenderer(t *testing.T) {
	for _, name := range DemoNames() {
		t.Run(name, func(t *testing.T) {
			spec, err := Demo(name)
			if err != nil {
				t.Fatal(err)
			}
			key := spec.Series[0].DataKey
			spec.Data[0][key] = nil
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := chart.Terminal(TerminalOptions{Width: 96, Height: 14}); err != nil {
				t.Fatalf("terminal: %v", err)
			}
			var svg bytes.Buffer
			if err := chart.WriteSVG(&svg, ImageOptions{Width: 900, Height: 600}); err != nil {
				t.Fatalf("SVG: %v", err)
			}
		})
	}
}

func TestAllMissingSeriesRenderAsEmptyData(t *testing.T) {
	for _, name := range []string{"line", "pie", "donut", "heatmap", "radar", "funnel"} {
		t.Run(name, func(t *testing.T) {
			spec, _ := Demo(name)
			for rowIndex := range spec.Data {
				for _, series := range spec.Series {
					spec.Data[rowIndex][series.DataKey] = nil
				}
			}
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := chart.Terminal(TerminalOptions{Width: 96, Height: 14}); err != nil {
				t.Fatalf("terminal: %v", err)
			}
			var svg bytes.Buffer
			if err := chart.WriteSVG(&svg, ImageOptions{Width: 900, Height: 600}); err != nil {
				t.Fatalf("SVG: %v", err)
			}
		})
	}
}

func TestTerminalEdgeDimensionsNeverOverflow(t *testing.T) {
	for _, name := range DemoNames() {
		for _, width := range []int{MinTerminalWidth, 40, 64, 120, 240} {
			for _, height := range []int{MinTerminalHeight, 10, 14, 30} {
				t.Run(fmt.Sprintf("%s-%dx%d", name, width, height), func(t *testing.T) {
					spec, _ := Demo(name)
					chart, err := New(spec)
					if err != nil {
						t.Fatal(err)
					}
					output, err := chart.Terminal(TerminalOptions{Width: width, Height: height})
					if err != nil {
						return
					}
					for lineIndex, line := range strings.Split(output, "\n") {
						if lineWidth := ansi.StringWidth(line); lineWidth > width {
							t.Fatalf("line %d is %d cells wide, want at most %d:\n%s", lineIndex+1, lineWidth, width, output)
						}
					}
				})
			}
		}
	}
}

func TestExtremeValuesFailClearly(t *testing.T) {
	spec, _ := Demo("line")
	spec.Data[0]["desktop"] = math.MaxFloat64
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("maximum float error = %v", err)
	}

	spec, _ = Demo("normalized-bar")
	spec.Data[0]["desktop"] = 1e308
	spec.Data[0]["mobile"] = 1e308
	if _, err := New(spec); err == nil || !strings.Contains(err.Error(), "total is too large") {
		t.Fatalf("overflowing normalized total error = %v", err)
	}
	if got := formatValue(2_500_000_000); got != "2.5bn" {
		t.Fatalf("billion formatting = %q", got)
	}
	if got := formatValue(1_250_000_000_000); got != "1.2tn" {
		t.Fatalf("trillion formatting = %q", got)
	}
}

func TestNarrowHeatmapFailsInsteadOfDroppingSeries(t *testing.T) {
	spec, _ := Demo("heatmap")
	for index := 0; index < 20; index++ {
		key := fmt.Sprintf("extra_%d", index)
		spec.Series = append(spec.Series, SeriesSpec{DataKey: key})
		for rowIndex := range spec.Data {
			spec.Data[rowIndex][key] = index
		}
	}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chart.Terminal(TerminalOptions{Width: 40, Height: 14}); err == nil || !strings.Contains(err.Error(), "too narrow") {
		t.Fatalf("narrow heatmap error = %v", err)
	}
}

func TestEveryDemoOutput(t *testing.T) {
	for _, name := range DemoNames() {
		t.Run(name, func(t *testing.T) {
			spec, err := Demo(name)
			if err != nil {
				t.Fatal(err)
			}
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			terminal, err := chart.Terminal(TerminalOptions{Width: 84, Height: 12})
			if err != nil || terminal == "" {
				t.Fatalf("terminal output: %q, %v", terminal, err)
			}
			var svg bytes.Buffer
			if err := chart.WriteSVG(&svg, ImageOptions{Width: 800, Height: 480}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(svg.String(), "<svg") {
				t.Fatal("SVG output has no svg element")
			}
			var image bytes.Buffer
			if err := chart.WritePNG(&image, ImageOptions{Width: 800, Height: 480}); err != nil {
				t.Fatal(err)
			}
			config, err := png.DecodeConfig(bytes.NewReader(image.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if config.Width != 800 || config.Height != 480 {
				t.Fatalf("PNG size = %dx%d", config.Width, config.Height)
			}
			var html bytes.Buffer
			if err := chart.WriteHTML(&html, HTMLOptions{Width: 800, Height: 480}); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(html.String(), "data-chart-type=\""+string(spec.Type)+"\"") {
				t.Fatal("HTML output has no chart type")
			}
		})
	}
}

func TestLineTerminalUsesConnectedUnicodeCurves(t *testing.T) {
	spec, _ := Demo("line")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsAny(output, "─│╭╮╰╯") {
		t.Fatalf("terminal line has no connected curve runes:\n%s", output)
	}
}

func TestTerminalLineCurveChangesRunes(t *testing.T) {
	spec, _ := Demo("line")
	smooth, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	smoothOutput, err := smooth.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	spec.Curve = Linear
	linear, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	linearOutput, err := linear.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	if ansi.Strip(smoothOutput) == ansi.Strip(linearOutput) {
		t.Fatal("linear and smooth terminal curves are identical")
	}
}

func TestHorizontalGroupedBarsContainBars(t *testing.T) {
	spec, _ := Demo("horizontal-bar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsAny(ansi.Strip(output), "█▉▊▋▌▍▎▏") {
		t.Fatalf("horizontal chart has no bar runes:\n%s", output)
	}
	if !strings.Contains(ansi.Strip(output), "Jan") {
		t.Fatalf("horizontal chart has no category labels:\n%s", output)
	}
}

func TestPieTerminalUsesProportionalBlocks(t *testing.T) {
	spec, _ := Demo("pie")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "█") || !strings.Contains(plain, "%") || !strings.Contains(plain, "Chrome") {
		t.Fatalf("pie chart is not a proportional block chart:\n%s", output)
	}
}

func TestHeatmapTerminalUsesDensityCells(t *testing.T) {
	spec, _ := Demo("heatmap")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.ContainsAny(plain, "░▒▓█") || !strings.Contains(plain, "Morning") || !strings.Contains(plain, "Mon") {
		t.Fatalf("heatmap is not a native density grid:\n%s", output)
	}
}

func TestRadarTerminalUsesComparableSeries(t *testing.T) {
	spec, _ := Demo("radar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.ContainsAny(plain, "●◆") || !strings.ContainsAny(plain, "·•") || !strings.Contains(plain, "Desktop") || !strings.Contains(plain, "Reach") {
		t.Fatalf("radar chart is not a native radial comparison:\n%s", output)
	}
}

func TestFunnelTerminalUsesCenteredBlocks(t *testing.T) {
	spec, _ := Demo("funnel")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 64, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.ContainsAny(plain, "█▉▊▋▌▍▎▏") || !strings.Contains(plain, "Visitors") || !strings.Contains(plain, "Customers") {
		t.Fatalf("funnel chart is not a native centered block view:\n%s", output)
	}
}

func TestNativeTerminalLayoutsRespectWidth(t *testing.T) {
	const width = 64
	for _, name := range []string{"pie", "donut", "heatmap", "radar", "funnel"} {
		t.Run(name, func(t *testing.T) {
			spec, err := Demo(name)
			if err != nil {
				t.Fatal(err)
			}
			chart, err := New(spec)
			if err != nil {
				t.Fatal(err)
			}
			output, err := chart.Terminal(TerminalOptions{Width: width, Height: 10})
			if err != nil {
				t.Fatal(err)
			}
			for index, line := range strings.Split(output, "\n") {
				if lineWidth := ansi.StringWidth(line); lineWidth > width {
					t.Fatalf("line %d is %d cells wide, want at most %d:\n%s", index+1, lineWidth, width, output)
				}
			}
		})
	}
}

func TestRowBasedNativeLayoutsDoNotSilentlyOmitData(t *testing.T) {
	spec, _ := Demo("radar")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chart.Terminal(TerminalOptions{Width: 64, Height: 8}); err == nil || !strings.Contains(err.Error(), "10 radar rows") {
		t.Fatalf("short radar error = %v", err)
	}
}

func TestComboTerminalContainsBarsAndLines(t *testing.T) {
	spec, _ := Demo("combo")
	spec.Axes = &DisplaySpec{Show: false}
	spec.Legend = &DisplaySpec{Show: false}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	output, err := chart.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	plain := ansi.Strip(output)
	if !strings.Contains(plain, "█") || !strings.ContainsAny(plain, "·•") || !strings.Contains(plain, "◆") {
		t.Fatalf("combo chart does not contain distinct bars and lines:\n%s", output)
	}
}

func TestHideAxesChangesTerminalLineChart(t *testing.T) {
	spec, _ := Demo("line")
	withAxes, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	withAxesOutput, err := withAxes.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	spec.Axes = &DisplaySpec{Show: false}
	withoutAxes, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	withoutAxesOutput, err := withoutAxes.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err != nil {
		t.Fatal(err)
	}
	if ansi.Strip(withAxesOutput) == ansi.Strip(withoutAxesOutput) {
		t.Fatal("hiding axes did not change terminal output")
	}
}

func TestTerminalRejectsUnsupportedValueLabels(t *testing.T) {
	spec, _ := Demo("line")
	spec.Labels = &DisplaySpec{Show: true}
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, err = chart.Terminal(TerminalOptions{Width: 84, Height: 12})
	if err == nil || !strings.Contains(err.Error(), "value labels") {
		t.Fatalf("value label error = %v", err)
	}
}

func TestSmallTerminalIsAnError(t *testing.T) {
	spec, _ := Demo("pie")
	chart, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, err = chart.Terminal(TerminalOptions{Width: 20, Height: 4})
	var sizeErr *TerminalSizeError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("small terminal error = %v", err)
	}
}
