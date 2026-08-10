package chartmux

import (
	"bytes"
	"errors"
	"image/png"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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
	if !strings.Contains(plain, "█") || !strings.ContainsAny(plain, "─│╭╮╰╯") {
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
