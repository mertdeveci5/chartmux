# chartmux

`chartmux` turns CSV, JSON, or a versioned chart spec into terminal charts, PNG, SVG, a self-contained HTML page, or resolved JSON. The data contract uses the familiar `dataKey`, `series`, and chart configuration model used by React chart libraries.

## Install

With Go:

```bash
go install github.com/mertdeveci5/chartmux/cmd/chartmux@latest
```

Prebuilt macOS, Linux, and Windows binaries are published on [GitHub Releases](https://github.com/mertdeveci5/chartmux/releases). Homebrew and npm installation are described below once their registry packages are available.

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

## High-resolution terminal UI

One-shot output uses portable Unicode curves and solid terminal bars. The responsive UI uses a real chart image through the Kitty graphics protocol when the terminal supports it, and falls back to Unicode when it does not:

```bash
./chartmux demo line --watch
./chartmux demo stacked-bar --watch
```

Resize the window to resize the chart. When Kitty graphics are available, press `g` to switch between graphics and Unicode. Press `q` to quit. Presentation can also be selected explicitly:

```bash
./chartmux demo line --watch --terminal-mode kitty
./chartmux demo line --watch --terminal-mode unicode
```

`auto` is the default. A nested terminal UI may not pass the graphics protocol through, so Chartmux will use the connected Unicode chart there instead of printing half-block image pixels.

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
    { "dataKey": "mobile", "label": "Mobile", "color": "#60A5FA" }
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

The terminal-inspired React/Vite playground lives in [`frontend`](frontend). It uses smooth SVG charts in the browser and the official Cloudflare Vite plugin for Workers deployment.

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
