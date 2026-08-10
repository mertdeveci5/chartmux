package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/x/ansi"
	"github.com/mertdeveci5/chartmux"
)

func newParser(t *testing.T, cli *CLI) *kong.Kong {
	t.Helper()
	parser, err := kong.New(cli, kong.Vars{"version": buildVersion()})
	if err != nil {
		t.Fatal(err)
	}
	return parser
}

func TestBuildVersionOverride(t *testing.T) {
	previous := version
	version = "v1.2.3"
	t.Cleanup(func() { version = previous })
	if got := buildVersion(); got != "1.2.3" {
		t.Fatalf("build version = %q", got)
	}
}

func TestSavedChartFileDoesNotNeedSubcommand(t *testing.T) {
	cli := &CLI{}
	parser := newParser(t, cli)
	context, err := parser.Parse([]string{"examples/area.json", "--width", "90"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := context.Command(), "show <input>"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	if cli.Show.Input != "examples/area.json" || cli.Show.Width != 90 {
		t.Fatalf("saved chart flags were not parsed: %+v", cli.Show)
	}
}

func TestNamedDemoIsExplicit(t *testing.T) {
	cli := &CLI{}
	parser := newParser(t, cli)
	context, err := parser.Parse([]string{"demo", "stacked-bar", "--width", "100"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := context.Command(), "demo <name>"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
	if cli.Demo.Name != "stacked-bar" || cli.Demo.Width != 100 {
		t.Fatalf("demo flags were not parsed: %+v", cli.Demo)
	}
}

func TestRemovedCommandsAreRejected(t *testing.T) {
	for _, command := range []string{"render", "hbar", "gauge"} {
		cli := &CLI{}
		parser := newParser(t, cli)
		if _, err := parser.Parse([]string{command, "data.csv"}); err == nil {
			t.Fatalf("removed command %q was accepted", command)
		}
	}
}

func TestTypedCommandsRejectIrrelevantFlags(t *testing.T) {
	tests := [][]string{
		{"line", "data.csv", "--type", "bar"},
		{"line", "data.csv", "--layout", "stacked"},
		{"bar", "data.csv", "--curve", "smooth"},
		{"pie", "data.csv", "--bins", "4"},
	}
	for _, arguments := range tests {
		cli := &CLI{}
		if _, err := newParser(t, cli).Parse(arguments); err == nil {
			t.Fatalf("irrelevant flags were accepted: %v", arguments)
		}
	}
}

func TestNegativeHistogramBinsFailValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.csv")
	if err := os.WriteFile(path, []byte("value\n1\n2\n3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	value := -1
	command := HistogramCommand{Bins: &value}
	command.Input = path
	if err := command.Run(); err == nil || !strings.Contains(err.Error(), "bins must be") {
		t.Fatalf("negative bins error = %v", err)
	}
}

func TestTerminalOutputToNonTTYHasNoANSI(t *testing.T) {
	spec, _ := chartmux.Demo("line")
	chart, err := chartmux.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	flags := PresentationFlags{OutputFlags: OutputFlags{Width: 84, Height: 12, Export: "terminal"}}
	if err := present(chart, flags, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != ansi.Strip(output.String()) {
		t.Fatal("non-TTY terminal output contains ANSI sequences")
	}
}

func TestJSONCanStreamToStdout(t *testing.T) {
	spec, _ := chartmux.Demo("line")
	chart, err := chartmux.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	flags := PresentationFlags{OutputFlags: OutputFlags{Export: "json", Output: "-"}}
	if err := present(chart, flags, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"version": 1`) || strings.Contains(output.String(), "Wrote ") {
		t.Fatalf("stdout JSON was polluted or incomplete:\n%s", output.String())
	}
}

func TestOutputSpecificFlagsAreNotSilentlyIgnored(t *testing.T) {
	tests := []OutputFlags{
		{Export: "terminal", ImageWidth: 800, TerminalMode: "auto"},
		{Export: "terminal", TerminalMode: "kitty"},
		{Export: "png", Width: 80, TerminalMode: "auto"},
		{Export: "json", ImageHeight: 400, TerminalMode: "auto"},
	}
	for _, flags := range tests {
		if err := validateOutputFlags(flags); err == nil {
			t.Fatalf("irrelevant output flags were accepted: %+v", flags)
		}
	}
}

func TestWatchDoesNotAdvertiseUnavailableKittyToggle(t *testing.T) {
	picture.ForceKittyCapability(picture.KittyCapabilityUnsupported)
	t.Cleanup(func() { picture.ForceKittyCapability(picture.KittyCapabilityUnknown) })
	spec, _ := chartmux.Demo("line")
	chart, err := chartmux.New(spec)
	if err != nil {
		t.Fatal(err)
	}
	view := newWatchModel(chart, "auto").View().Content
	if strings.Contains(view, "g switch presentation") {
		t.Fatal("watch UI advertised an unavailable Kitty toggle")
	}
}

func TestEmptyInputIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPath(path); err == nil {
		t.Fatal("empty input did not fail")
	}
}
