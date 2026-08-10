package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/mertdeveci5/chartmux"
)

var version string

type CLI struct {
	Version   kong.VersionFlag `name:"version" help:"Show version and exit"`
	Show      ShowCommand      `cmd:"" default:"withargs" help:"Open a saved chart or a CSV, TSV, or JSON dataset"`
	Demo      DemoCommand      `cmd:"" help:"Show named chart examples"`
	Validate  ValidateCommand  `cmd:"" help:"Validate a saved chart or dataset without drawing it"`
	Schema    SchemaCommand    `cmd:"" help:"Print the chartmux v1 JSON Schema"`
	Bar       BarCommand       `cmd:"" help:"Show grouped, stacked, normalized, or horizontal bars"`
	Line      LineCommand      `cmd:"" help:"Show a single or multi-series line chart"`
	Area      AreaCommand      `cmd:"" help:"Show an overlay, stacked, or normalized area chart"`
	Combo     ComboCommand     `cmd:"" help:"Show bars and lines on shared axes"`
	Scatter   ScatterCommand   `cmd:"" help:"Show relationships between numeric values"`
	Histogram HistogramCommand `cmd:"" help:"Show a numeric distribution"`
	Pie       PieCommand       `cmd:"" help:"Show categorical proportions"`
	Donut     DonutCommand     `cmd:"" help:"Show categorical proportions as a ring"`
	Heatmap   HeatmapCommand   `cmd:"" help:"Show values across a categorical matrix"`
	Radar     RadarCommand     `cmd:"" help:"Show series across shared metrics"`
	Funnel    FunnelCommand    `cmd:"" help:"Show decreasing conversion stages"`
}

type DatasetFlags struct {
	Input  string   `arg:"" optional:"" help:"CSV, TSV, or JSON dataset; use - for stdin"`
	X      string   `help:"Column used for the x axis" placeholder:"COLUMN"`
	Series []string `help:"Numeric series; comma-separated or repeated" placeholder:"COLUMN" sep:","`
}

type AppearanceFlags struct {
	Title       string `help:"Chart title" placeholder:"TEXT"`
	Description string `help:"Text below the title" placeholder:"TEXT"`
	Footer      string `help:"Footer text" placeholder:"TEXT"`
	HideLegend  bool   `help:"Hide the legend"`
	HideAxes    bool   `help:"Hide axes and grid lines"`
	ShowValues  bool   `help:"Show numeric labels; image exports only for graphical charts"`
	Theme       string `help:"Color theme: light or dark" enum:"light,dark," default:"" placeholder:"THEME"`
}

type OutputFlags struct {
	Width        int    `help:"Terminal output width; defaults to terminal width"`
	Height       int    `help:"Terminal chart height; defaults to 14"`
	Watch        bool   `help:"Open the responsive terminal UI"`
	TerminalMode string `help:"Terminal UI presentation: auto, unicode, or kitty" default:"auto" enum:"auto,unicode,kitty" placeholder:"MODE"`
	Export       string `help:"Output format: terminal, png, svg, html, or json" default:"terminal" enum:"terminal,png,svg,html,json" placeholder:"FORMAT"`
	Output       string `help:"Output path; use - for stdout" placeholder:"FILE"`
	Copy         bool   `help:"Copy an exported PNG file to the clipboard"`
	NoColor      bool   `help:"Disable ANSI color in one-shot terminal output"`
	ImageWidth   int    `help:"PNG, SVG, or HTML width in pixels"`
	ImageHeight  int    `help:"PNG, SVG, or HTML height in pixels"`
}

type PresentationFlags struct {
	AppearanceFlags `embed:""`
	OutputFlags     `embed:""`
}

type CommonChartFlags struct {
	DatasetFlags      `embed:""`
	PresentationFlags `embed:""`
}

type SpecOptions struct {
	Type        string   `help:"Chart type for a direct dataset" enum:"bar,line,area,combo,scatter,histogram,pie,donut,heatmap,radar,funnel," default:"" placeholder:"TYPE"`
	Marks       []string `help:"Combo marks aligned with series: bar or line" enum:"bar,line" placeholder:"MARK" sep:","`
	Layout      string   `help:"Layout: grouped, overlay, stacked, or normalized" enum:"grouped,overlay,stacked,normalized," default:"" placeholder:"LAYOUT"`
	Orientation string   `help:"Bar orientation: vertical or horizontal" enum:"vertical,horizontal," default:"" placeholder:"ORIENTATION"`
	Curve       string   `help:"Line curve: linear or smooth" enum:"linear,smooth," default:"" placeholder:"CURVE"`
	Bins        *int     `help:"Histogram bins: 0 for automatic, or 1-100" placeholder:"COUNT"`
}

type ShowCommand struct {
	Input             string   `arg:"" optional:"" help:"Saved chart, CSV, TSV, or JSON dataset; use - for stdin"`
	X                 string   `help:"Column used for the x axis" placeholder:"COLUMN"`
	Series            []string `help:"Numeric series; comma-separated or repeated" placeholder:"COLUMN" sep:","`
	SpecOptions       `embed:""`
	PresentationFlags `embed:""`
}

type ValidateCommand struct {
	Input           string   `arg:"" help:"Saved chart, CSV, TSV, or JSON dataset; use - for stdin"`
	X               string   `help:"Column used for the x axis" placeholder:"COLUMN"`
	Series          []string `help:"Numeric series; comma-separated or repeated" placeholder:"COLUMN" sep:","`
	SpecOptions     `embed:""`
	AppearanceFlags `embed:""`
}

type SchemaCommand struct{}

type BarCommand struct {
	CommonChartFlags `embed:""`
	Layout           string `help:"Bar layout: grouped, stacked, or normalized" enum:"grouped,stacked,normalized," default:"" placeholder:"LAYOUT"`
	Orientation      string `help:"Bar orientation: vertical or horizontal" enum:"vertical,horizontal," default:"" placeholder:"ORIENTATION"`
}

type LineCommand struct {
	CommonChartFlags `embed:""`
	Curve            string `help:"Line curve: linear or smooth" enum:"linear,smooth," default:"" placeholder:"CURVE"`
}

type AreaCommand struct {
	CommonChartFlags `embed:""`
	Layout           string `help:"Area layout: overlay, stacked, or normalized" enum:"overlay,stacked,normalized," default:"" placeholder:"LAYOUT"`
	Curve            string `help:"Area curve: linear or smooth" enum:"linear,smooth," default:"" placeholder:"CURVE"`
}

type ComboCommand struct {
	CommonChartFlags `embed:""`
	Marks            []string `help:"Marks aligned with series: bar or line" enum:"bar,line" placeholder:"MARK" sep:","`
}

type ScatterCommand struct {
	CommonChartFlags `embed:""`
}

type HistogramCommand struct {
	CommonChartFlags `embed:""`
	Bins             *int   `help:"Bins: 0 for automatic, or 1-100" placeholder:"COUNT"`
	Orientation      string `help:"Bar orientation: vertical or horizontal" enum:"vertical,horizontal," default:"" placeholder:"ORIENTATION"`
}

type PieCommand struct {
	CommonChartFlags `embed:""`
}
type DonutCommand struct {
	CommonChartFlags `embed:""`
}
type HeatmapCommand struct {
	CommonChartFlags `embed:""`
}
type RadarCommand struct {
	CommonChartFlags `embed:""`
}
type FunnelCommand struct {
	CommonChartFlags `embed:""`
}

type DemoCommand struct {
	Name              string `arg:"" optional:"" default:"line" help:"Demo name; use --list to see names"`
	List              bool   `help:"List available demos"`
	All               bool   `help:"Show every demo"`
	PresentationFlags `embed:""`
}

var errorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F87171"))

func main() {
	cli := &CLI{}
	context := kong.Parse(
		cli,
		kong.Name("chartmux"),
		kong.Description("Draw charts from files or stdin. Example: chartmux line sales.csv --x month --series revenue"),
		kong.Vars{"version": buildVersion()},
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true, Summary: false}),
	)
	if err := context.Run(); err != nil {
		lipgloss.Fprintln(os.Stderr, errorStyle.Render("Error:"), err)
		os.Exit(1)
	}
}

func buildVersion() string {
	if version != "" {
		return strings.TrimPrefix(version, "v")
	}
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

func (command *ShowCommand) Run() error {
	if command.Input == "" {
		return fmt.Errorf("pass a chart file or dataset; use 'chartmux demo --list' for examples")
	}
	spec, err := loadFlexibleSpec(command.Input, command.X, command.Series, command.SpecOptions)
	if err != nil {
		return err
	}
	applyAppearanceOverrides(&spec, command.AppearanceFlags)
	return buildAndPresent(spec, command.PresentationFlags)
}

func (command *ValidateCommand) Run() error {
	spec, err := loadFlexibleSpec(command.Input, command.X, command.Series, command.SpecOptions)
	if err != nil {
		return err
	}
	applyAppearanceOverrides(&spec, command.AppearanceFlags)
	if _, err := chartmux.New(spec); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "valid")
	return nil
}

func (command *SchemaCommand) Run() error {
	_, err := os.Stdout.Write(chartmux.SchemaJSON())
	return err
}

func (command *BarCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Bar, func(spec *chartmux.Spec) error {
		if command.Layout != "" {
			spec.Layout = chartmux.Layout(command.Layout)
		}
		if command.Orientation != "" {
			spec.Orientation = chartmux.Orientation(command.Orientation)
		}
		return nil
	})
}

func (command *LineCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Line, func(spec *chartmux.Spec) error {
		if command.Curve != "" {
			spec.Curve = chartmux.Curve(command.Curve)
		}
		return nil
	})
}

func (command *AreaCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Area, func(spec *chartmux.Spec) error {
		if command.Layout != "" {
			spec.Layout = chartmux.Layout(command.Layout)
		}
		if command.Curve != "" {
			spec.Curve = chartmux.Curve(command.Curve)
		}
		return nil
	})
}

func (command *ComboCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Combo, func(spec *chartmux.Spec) error {
		return applyComboMarks(spec, command.Marks)
	})
}

func (command *ScatterCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Scatter, nil)
}

func (command *HistogramCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Histogram, func(spec *chartmux.Spec) error {
		if command.Bins != nil {
			spec.Bins = *command.Bins
		}
		if command.Orientation != "" {
			spec.Orientation = chartmux.Orientation(command.Orientation)
		}
		return nil
	})
}

func (command *PieCommand) Run() error   { return command.CommonChartFlags.run(chartmux.Pie, nil) }
func (command *DonutCommand) Run() error { return command.CommonChartFlags.run(chartmux.Donut, nil) }
func (command *HeatmapCommand) Run() error {
	return command.CommonChartFlags.run(chartmux.Heatmap, nil)
}
func (command *RadarCommand) Run() error  { return command.CommonChartFlags.run(chartmux.Radar, nil) }
func (command *FunnelCommand) Run() error { return command.CommonChartFlags.run(chartmux.Funnel, nil) }

func (command *DemoCommand) Run() error {
	if command.List {
		fmt.Fprintln(os.Stdout, strings.Join(chartmux.DemoNames(), "\n"))
		return nil
	}
	if command.All {
		if command.Watch {
			return fmt.Errorf("--all cannot be combined with --watch")
		}
		if command.Output != "" {
			return fmt.Errorf("--all cannot use one --output path")
		}
		for index, name := range chartmux.DemoNames() {
			spec, err := chartmux.Demo(name)
			if err != nil {
				return err
			}
			flags := command.PresentationFlags
			applyAppearanceOverrides(&spec, flags.AppearanceFlags)
			if index > 0 && flags.Export == "terminal" {
				fmt.Fprintln(os.Stdout)
			}
			if flags.Export != "terminal" {
				flags.Output = name + "." + flags.Export
			}
			if err := buildAndPresent(spec, flags); err != nil {
				return err
			}
		}
		return nil
	}
	spec, err := chartmux.Demo(command.Name)
	if err != nil {
		return fmt.Errorf("%w; use 'chartmux demo --list' to see available demos", err)
	}
	applyAppearanceOverrides(&spec, command.AppearanceFlags)
	return buildAndPresent(spec, command.PresentationFlags)
}

func (flags CommonChartFlags) run(chartType chartmux.Type, configure func(*chartmux.Spec) error) error {
	if flags.Input == "" {
		return fmt.Errorf("pass a dataset or use 'chartmux demo %s'", demoForType(chartType))
	}
	content, err := readPath(flags.Input)
	if err != nil {
		return err
	}
	dataset, err := chartmux.ParseDataset(bytes.NewReader(content))
	if err != nil {
		return err
	}
	spec, err := chartmux.SpecFromDataset(dataset, chartType, flags.X, flags.Series)
	if err != nil {
		return err
	}
	if configure != nil {
		if err := configure(&spec); err != nil {
			return err
		}
	}
	applyAppearanceOverrides(&spec, flags.AppearanceFlags)
	return buildAndPresent(spec, flags.PresentationFlags)
}

func loadFlexibleSpec(path, x string, series []string, options SpecOptions) (chartmux.Spec, error) {
	content, err := readPath(path)
	if err != nil {
		return chartmux.Spec{}, err
	}
	var spec chartmux.Spec
	if firstNonSpace(content) == '{' {
		spec, err = chartmux.ParseSpec(bytes.NewReader(content))
		if err != nil {
			return chartmux.Spec{}, err
		}
		if options.Type != "" {
			return chartmux.Spec{}, fmt.Errorf("--type is only valid for dataset input; the saved chart already has type %s", spec.Type)
		}
		if x != "" {
			spec.XAxis.DataKey = x
		}
		if len(series) > 0 {
			spec.Series = seriesSpecs(series)
		}
	} else {
		if options.Type == "" {
			return chartmux.Spec{}, fmt.Errorf("a dataset needs --type; for example: chartmux sales.csv --type line --x month --series revenue")
		}
		dataset, parseErr := chartmux.ParseDataset(bytes.NewReader(content))
		if parseErr != nil {
			return chartmux.Spec{}, parseErr
		}
		spec, err = chartmux.SpecFromDataset(dataset, chartmux.Type(options.Type), x, series)
		if err != nil {
			return chartmux.Spec{}, err
		}
	}
	if options.Layout != "" {
		spec.Layout = chartmux.Layout(options.Layout)
	}
	if options.Orientation != "" {
		spec.Orientation = chartmux.Orientation(options.Orientation)
	}
	if options.Curve != "" {
		spec.Curve = chartmux.Curve(options.Curve)
	}
	if options.Bins != nil {
		spec.Bins = *options.Bins
	}
	if len(options.Marks) > 0 {
		if spec.Type != chartmux.Combo {
			return chartmux.Spec{}, fmt.Errorf("--marks is only valid for combo charts")
		}
		if err := applyComboMarks(&spec, options.Marks); err != nil {
			return chartmux.Spec{}, err
		}
	} else if spec.Type == chartmux.Combo {
		applyComboMarks(&spec, nil)
	}
	return spec, nil
}

func seriesSpecs(keys []string) []chartmux.SeriesSpec {
	series := make([]chartmux.SeriesSpec, len(keys))
	for index, key := range keys {
		series[index] = chartmux.SeriesSpec{DataKey: key}
	}
	return series
}

func applyComboMarks(spec *chartmux.Spec, marks []string) error {
	if len(marks) > 0 && len(marks) != len(spec.Series) {
		return fmt.Errorf("--marks needs one mark for each of the %d series", len(spec.Series))
	}
	for index := range spec.Series {
		if len(marks) > 0 {
			spec.Series[index].Mark = chartmux.Mark(marks[index])
		} else if spec.Series[index].Mark == "" {
			if index == 0 {
				spec.Series[index].Mark = chartmux.MarkBar
			} else {
				spec.Series[index].Mark = chartmux.MarkLine
			}
		}
	}
	return nil
}

func applyAppearanceOverrides(spec *chartmux.Spec, flags AppearanceFlags) {
	if flags.Title != "" {
		spec.Title = flags.Title
	}
	if flags.Description != "" {
		spec.Description = flags.Description
	}
	if flags.Footer != "" {
		spec.Footer = flags.Footer
	}
	if flags.HideLegend {
		spec.Legend = &chartmux.DisplaySpec{Show: false}
	}
	if flags.HideAxes {
		spec.Axes = &chartmux.DisplaySpec{Show: false}
	}
	if flags.ShowValues {
		spec.Labels = &chartmux.DisplaySpec{Show: true}
	}
	if flags.Theme != "" {
		spec.Theme = flags.Theme
	}
}

func buildAndPresent(spec chartmux.Spec, flags PresentationFlags) error {
	chart, err := chartmux.New(spec)
	if err != nil {
		return err
	}
	return present(chart, flags, os.Stdout)
}

func present(chart *chartmux.Chart, flags PresentationFlags, stdout io.Writer) error {
	if err := validateOutputFlags(flags.OutputFlags); err != nil {
		return err
	}
	if flags.Export == "terminal" {
		if flags.Copy {
			return fmt.Errorf("--copy requires --export png")
		}
		if flags.Watch {
			if flags.Output != "" {
				return fmt.Errorf("--output cannot be combined with --watch")
			}
			return watch(chart, flags.TerminalMode)
		}
		width, _ := terminalSize()
		if flags.Width > 0 {
			width = flags.Width
		}
		output, err := chart.Terminal(chartmux.TerminalOptions{Width: width, Height: flags.Height})
		if err != nil {
			return err
		}
		if flags.Output == "" || flags.Output == "-" {
			return writeTerminal(stdout, output, flags.NoColor)
		}
		var plain bytes.Buffer
		if err := writeTerminal(&plain, output, true); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Clean(flags.Output), plain.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", flags.Output, err)
		}
		fmt.Fprintf(stdout, "Wrote %s\n", flags.Output)
		return nil
	}
	if flags.Watch {
		return fmt.Errorf("--watch requires --export terminal")
	}
	if flags.Copy && flags.Export != "png" {
		return fmt.Errorf("--copy requires --export png")
	}
	if flags.Copy && flags.Output == "-" {
		return fmt.Errorf("--copy requires a PNG file, not --output -")
	}
	var content bytes.Buffer
	options := chartmux.ImageOptions{Width: flags.ImageWidth, Height: flags.ImageHeight}
	var err error
	switch flags.Export {
	case "png":
		err = chart.WritePNG(&content, options)
	case "svg":
		err = chart.WriteSVG(&content, options)
	case "html":
		err = chart.WriteHTML(&content, chartmux.HTMLOptions(options))
	case "json":
		err = chart.WriteJSON(&content)
	default:
		err = fmt.Errorf("unsupported output format %q", flags.Export)
	}
	if err != nil {
		return err
	}
	if flags.Output == "-" {
		_, err := stdout.Write(content.Bytes())
		return err
	}
	path := flags.Output
	if path == "" {
		path = "chart." + flags.Export
	}
	if err := os.WriteFile(filepath.Clean(path), content.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if flags.Copy {
		if err := copyPNG(path); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "Wrote %s and copied it to the clipboard\n", path)
		return nil
	}
	fmt.Fprintf(stdout, "Wrote %s\n", path)
	return nil
}

func validateOutputFlags(flags OutputFlags) error {
	terminalMode := flags.TerminalMode
	if terminalMode == "" {
		terminalMode = "auto"
	}
	if flags.Export == "terminal" {
		if flags.ImageWidth != 0 || flags.ImageHeight != 0 {
			return fmt.Errorf("--image-width and --image-height require --export png, svg, or html")
		}
		if flags.Watch && flags.NoColor {
			return fmt.Errorf("--no-color cannot be combined with --watch")
		}
		if !flags.Watch && terminalMode != "auto" {
			return fmt.Errorf("--terminal-mode requires --watch")
		}
		return nil
	}
	if flags.Width != 0 || flags.Height != 0 {
		return fmt.Errorf("--width and --height require --export terminal")
	}
	if terminalMode != "auto" {
		return fmt.Errorf("--terminal-mode requires --watch")
	}
	if flags.NoColor {
		return fmt.Errorf("--no-color requires --export terminal")
	}
	if flags.Export == "json" && (flags.ImageWidth != 0 || flags.ImageHeight != 0) {
		return fmt.Errorf("--image-width and --image-height are not valid for JSON output")
	}
	return nil
}

func writeTerminal(writer io.Writer, output string, noColor bool) error {
	environment := os.Environ()
	if noColor {
		environment = withoutEnvironment(environment, "NO_COLOR")
		environment = append(environment, "NO_COLOR=1")
	}
	_, err := fmt.Fprintln(colorprofile.NewWriter(writer, environment), output)
	return err
}

func withoutEnvironment(environment []string, name string) []string {
	prefix := name + "="
	filtered := make([]string, 0, len(environment))
	for _, value := range environment {
		if !strings.HasPrefix(value, prefix) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func readPath(path string) ([]byte, error) {
	var reader io.Reader = os.Stdin
	var file *os.File
	var err error
	if path != "-" {
		file, err = os.Open(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		defer file.Close()
		reader = file
	}
	content, err := io.ReadAll(io.LimitReader(reader, chartmux.MaxInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(content) > chartmux.MaxInputBytes {
		return nil, fmt.Errorf("input exceeds %d MB", chartmux.MaxInputBytes>>20)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return nil, fmt.Errorf("input is empty")
	}
	return content, nil
}

func firstNonSpace(content []byte) byte {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return 0
	}
	return trimmed[0]
}

func terminalSize() (int, int) {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width < chartmux.MinTerminalWidth {
		return chartmux.DefaultTerminalWidth, chartmux.DefaultTerminalHeight
	}
	return width, height
}

func demoForType(chartType chartmux.Type) string {
	if chartType == chartmux.Bar {
		return "grouped-bar"
	}
	return string(chartType)
}
