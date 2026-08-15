# chartmux

`chartmux` turns CSV, JSON, or a versioned chart spec into terminal charts, PNG, SVG, a self-contained HTML page, or resolved JSON. The data contract uses the familiar `dataKey`, `series`, and chart configuration model used by React chart libraries.

## Install

With Homebrew:

```bash
brew install mertdeveci5/tap/chartmux
```

With npm:

```bash
npm install --global chartmux
```

With Go:

```bash
go install github.com/mertdeveci5/chartmux/cmd/chartmux@latest
```

Prebuilt macOS, Linux, and Windows binaries are published on [GitHub Releases](https://github.com/mertdeveci5/chartmux/releases).

## Build and try it

```bash
cd ~/Desktop/Code/chartmux
go build -o chartmux ./cmd/chartmux

./chartmux demo line
./chartmux demo grouped-bar
./chartmux demo stacked-bar
./chartmux demo normalized-bar
```

List every built-in example or display them all:

```bash
./chartmux demo --list
./chartmux demo --all
```

## Native terminal UI

One-shot output and the responsive UI both use the same portable Unicode cell renderer. It layers sparse grids, axes, stable series textures, marks, crosshairs, and annotations deterministically, so labels never overwrite data and stacked bars never acquire accidental gaps. Lines and areas use a dithered terminal grammar; heatmaps use compact density matrices; pie, donut, and radar charts use real radial geometry. The interactive shell is powered by Bubble Tea and remains text-native in every terminal:

```bash
./chartmux demo line --watch
./chartmux demo stacked-bar --watch
```

Resize the window to resize the chart. Press `←` or `→` to inspect categories and `↑` or `↓` to switch the focused series; the inspector wraps safely and shows all values for the selected category. Press `esc` to close it, `c` to copy the plain-text chart through the terminal clipboard protocol, `?` for help, `ctrl+z` to suspend, or `q` to quit. PNG, SVG, and HTML remain available as explicit export formats, but interactive terminal charts never switch to an image protocol.

Add collision-safe narrative notes from the CLI. Repeat `--annotation` to add more than one; annotations receive their own reserved band instead of covering chart marks:

```bash
./chartmux demo annotated-bar --watch
./chartmux demo line --annotation "Growth accelerated after the refinancing"
```

Saved chart specs can place an annotation above or below the plot and optionally tie its context to a zero-based data row and series key:

```json
{"text":"Mobile mix expanded","position":"top","dataIndex":5,"series":"mobile","color":"#3FC5D8"}
```

## Use your own data

Pass a CSV, TSV, semicolon-delimited file, or JSON array directly:

```bash
./chartmux sales.csv --type line --x month --series revenue,cost
./chartmux bar sales.csv --x month --series desktop,mobile --layout grouped
./chartmux bar sales.csv --x month --series desktop,mobile --layout stacked
./chartmux bar sales.csv --x month --series desktop,mobile --orientation horizontal
./chartmux combo sales.csv --x month --series revenue,target --marks bar,line
```

Use `-` to read the dataset from stdin. Empty input is an error; examples are only available through the explicit `demo` command.

Each typed command only exposes options that apply to that chart. For example, `line` accepts `--curve`, `bar` accepts `--layout` and `--orientation`, and `histogram` accepts `--bins`. Run `./chartmux <command> --help` for the exact contract.

## Chart types

- `line`
- `bar` with `grouped`, `stacked`, or `normalized` layout
- horizontal `bar`
- `area` with `overlay`, `stacked`, or `normalized` layout
- `combo` with per-series `bar` or `line` marks
- `scatter`
- `histogram`
- `pie` and `donut`
- `heatmap`
- `radar`
- `funnel`

## Output and automation

All chart types use the same output engine, so an exported chart preserves the same data, layout, colors, axes, and legend:

```bash
./chartmux demo stacked-area --export png --output chart.png
./chartmux demo grouped-bar --export svg --output chart.svg
./chartmux examples/area.json --export html --output chart.html
./chartmux demo line --export png --output chart.png --copy
./chartmux examples/area.json --export json --output -
```

PNG defaults to 1200×720. SVG and HTML default to 960×540. Override either with `--image-width` and `--image-height`. HTML output is responsive, self-contained, and has no JavaScript or external assets.

Use `--output -` to stream PNG, SVG, HTML, or resolved JSON without status text. One-shot terminal output automatically removes ANSI color when redirected; `--no-color` also disables it explicitly. Terminal output can be saved with `--export terminal --output chart.txt`.

For scripts and editors:

```bash
./chartmux --version
./chartmux validate examples/area.json
./chartmux schema > chartmux-v1.schema.json
```

`validate` prints `valid` and exits zero only after data inference and chart validation succeed. `--export json` emits the final versioned spec after defaults and CLI overrides have been resolved. Histogram `--bins 0` means automatic bin selection; negative values and values above 100 are errors.

On macOS and Windows, `--copy` uses the native clipboard. Linux requires `wl-copy` or `xclip`.

## Saved chart contract

Saved files are strict, versioned JSON. Unknown fields fail early instead of being silently ignored. See [`examples/area.json`](examples/area.json) and [`schema/v1.json`](schema/v1.json).

```json
{
  "$schema": "https://chartmux.dev/schema/v1.json",
  "version": 1,
  "type": "line",
  "title": "Visitors",
  "xAxis": { "dataKey": "month", "kind": "category" },
  "series": [
    { "dataKey": "desktop", "label": "Desktop", "color": "var(--chart-1)" },
    { "dataKey": "mobile", "label": "Mobile", "color": "#3FC5D8" }
  ],
  "curve": "smooth",
  "data": [
    { "month": "January", "desktop": 186, "mobile": 80 },
    { "month": "February", "desktop": 305, "mobile": 200 }
  ]
}
```

Combo charts add a `mark` to each series. Bar and area charts use `layout`; bars also support `orientation`. Display flags use explicit objects such as `"legend": { "show": false }`.

## Go package

The CLI and importable package share the same validated `Spec` and output engine:

```go
spec := chartmux.Spec{
    Version: chartmux.SpecVersion,
    Type:    chartmux.Line,
    XAxis:   chartmux.AxisSpec{DataKey: "month"},
    Series:  []chartmux.SeriesSpec{{DataKey: "revenue", Label: "Revenue"}},
    Data: []chartmux.Row{
        {"month": "Jan", "revenue": 120},
        {"month": "Feb", "revenue": 180},
    },
}

chart, err := chartmux.New(spec)
if err != nil {
    return err
}
return chart.WriteSVG(writer, chartmux.ImageOptions{Width: 960, Height: 540})
```

Use `chart.WriteJSON(writer)` for the resolved chart contract and `chartmux.SchemaJSON()` for the embedded v1 schema.

The previous implicit demos, `hbar`, `gauge`, and `--stacked` paths were removed. Use `demo`, `bar --orientation horizontal`, and `--layout stacked`; gauge is intentionally absent until it has an engine-backed implementation.

## Frontend playground

The React/Vite gallery lives in [`frontend`](frontend). Its chart previews are truecolor ANSI captures from the real terminal renderer, not browser-drawn approximations. The surrounding interface uses the official Cloudflare Vite plugin for Workers deployment.

```bash
cd frontend
npm install
npm run dev
```

Validate the production bundle and Cloudflare deployment without publishing:

```bash
npm run check
npm run build
npm run deploy:dry
```
