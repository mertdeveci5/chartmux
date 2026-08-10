import { useEffect, useMemo, useRef, useState } from "react";
import { ChartPreview, demos, type DemoId } from "./charts";

type ExportFormat = "terminal" | "png" | "svg" | "html";

function chartCommand(demo: DemoId, format: ExportFormat) {
  if (format === "terminal") {
    return `chartmux demo ${demo} --watch`;
  }
  return `chartmux demo ${demo} --export ${format} --output ${demo}.${format}`;
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), 1600);
    return () => window.clearTimeout(timer);
  }, [copied]);

  async function copy() {
    await navigator.clipboard.writeText(value);
    setCopied(true);
  }

  return (
    <button className="copy-button" onClick={copy} type="button">
      <span aria-hidden="true">{copied ? "✓" : "⧉"}</span>
      <span>{copied ? "copied" : "copy"}</span>
    </button>
  );
}

function PixelLogo() {
  return (
    <svg
      aria-hidden="true"
      className="pixel-logo"
      shapeRendering="crispEdges"
      viewBox="0 0 120 72"
    >
      <rect fill="currentColor" height="20" opacity="0.32" width="12" x="8" y="44" />
      <rect fill="currentColor" height="30" opacity="0.48" width="12" x="28" y="34" />
      <rect fill="currentColor" height="25" opacity="0.62" width="12" x="48" y="39" />
      <rect fill="currentColor" height="42" opacity="0.78" width="12" x="68" y="22" />
      <rect fill="currentColor" height="50" width="12" x="88" y="14" />
      <path d="M8 37h20v-9h20v5h20V17h20V8h24" fill="none" stroke="var(--chart-cyan)" strokeWidth="4" />
      <rect fill="var(--foreground)" height="4" width="8" x="104" y="60" />
    </svg>
  );
}

function columnsForViewport() {
  if (window.innerWidth <= 620) return 1;
  if (window.innerWidth <= 960) return 2;
  return 3;
}

function App() {
  const [selected, setSelected] = useState<DemoId | null>(null);
  const [format, setFormat] = useState<ExportFormat>("terminal");
  const tileRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const demo = useMemo(
    () => demos.find((item) => item.id === selected) ?? null,
    [selected],
  );
  const command = demo ? chartCommand(demo.id, format) : "chartmux demo --all";

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target?.matches("input, textarea, select, [contenteditable='true']")) return;

      if (event.key === "Escape" && selected) {
        setSelected(null);
        setFormat("terminal");
        return;
      }

      if (!selected && /^[1-9]$/.test(event.key)) {
        const nextDemo = demos[Number(event.key) - 1];
        if (nextDemo) setSelected(nextDemo.id);
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [selected]);

  function moveGridFocus(event: React.KeyboardEvent, index: number) {
    const columns = columnsForViewport();
    let nextIndex = index;

    switch (event.key) {
      case "ArrowRight":
        nextIndex = Math.min(index + 1, demos.length - 1);
        break;
      case "ArrowLeft":
        nextIndex = Math.max(index - 1, 0);
        break;
      case "ArrowDown":
        nextIndex = Math.min(index + columns, demos.length - 1);
        break;
      case "ArrowUp":
        nextIndex = Math.max(index - columns, 0);
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = demos.length - 1;
        break;
      default:
        return;
    }

    event.preventDefault();
    tileRefs.current[nextIndex]?.focus();
  }

  function openDemo(id: DemoId) {
    setSelected(id);
    setFormat("terminal");
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function closeDemo() {
    const previousIndex = demos.findIndex((item) => item.id === selected);
    setSelected(null);
    setFormat("terminal");
    window.requestAnimationFrame(() => tileRefs.current[previousIndex]?.focus());
  }

  return (
    <main className="terminal-app">
      <header className="terminal-topbar">
        <a className="app-name" href="#top" aria-label="Chartmux home">
          <span className="prompt-mark" aria-hidden="true">›_</span>
          <span>chartmux</span>
          <span className="version">0.1.0</span>
        </a>
        <span className="view-name">{demo ? `gallery / ${demo.id}` : "gallery"}</span>
        <span className="chart-count">{demos.length} charts</span>
      </header>

      {demo ? (
        <section className="workspace" aria-labelledby="workspace-title">
          <div className="workspace-heading">
            <button className="back-button" onClick={closeDemo} type="button">
              <span aria-hidden="true">←</span> gallery
            </button>
            <div className="workspace-title-group">
              <span className="chart-index">{String(demos.indexOf(demo) + 1).padStart(2, "0")}</span>
              <div>
                <h1 id="workspace-title">{demo.label}</h1>
                <p>{demo.description}</p>
              </div>
            </div>
            <span className="live-status"><i aria-hidden="true" /> smooth SVG</span>
          </div>

          <div className="workspace-chart" role="img" aria-label={`${demo.label} demonstration chart`}>
            <ChartPreview demo={demo.id} />
          </div>

          <div className="workspace-meta">
            <span>{demo.dataSummary}</span>
            <span>responsive</span>
            <span>dark theme</span>
          </div>

          <div className="command-console">
            <div className="format-picker" aria-label="Output format">
              {(["terminal", "png", "svg", "html"] as const).map((item) => (
                <button
                  aria-pressed={format === item}
                  className={format === item ? "active" : ""}
                  key={item}
                  onClick={() => setFormat(item)}
                  type="button"
                >
                  {item}
                </button>
              ))}
            </div>
            <div className="command-line">
              <span className="prompt-symbol" aria-hidden="true">$</span>
              <code>{command}</code>
              <CopyButton value={command} />
            </div>
          </div>
        </section>
      ) : (
        <>
          <section className="intro" id="top" aria-labelledby="page-title">
            <PixelLogo />
            <div>
              <h1 id="page-title">CHARTMUX</h1>
              <p>Beautiful, deterministic charts from your terminal.</p>
            </div>
            <div className="boot-status" aria-label="Application ready">
              <span><i aria-hidden="true" /> graphics ready</span>
              <span>choose a chart to begin</span>
            </div>
          </section>

          <section aria-labelledby="gallery-title">
            <div className="section-bar">
              <div>
                <span className="section-path">~/charts</span>
                <h2 id="gallery-title">Choose a chart</h2>
              </div>
              <div className="gallery-command">
                <span className="prompt-symbol" aria-hidden="true">$</span>
                <code>{command}</code>
                <CopyButton value={command} />
              </div>
            </div>

            <div className="chart-grid">
              {demos.map((item, index) => (
                <button
                  aria-label={`Open ${item.label} demo`}
                  className="chart-tile"
                  key={item.id}
                  onClick={() => openDemo(item.id)}
                  onKeyDown={(event) => moveGridFocus(event, index)}
                  ref={(node) => {
                    tileRefs.current[index] = node;
                  }}
                  type="button"
                >
                  <span className="tile-heading">
                    <span className="tile-index">{String(index + 1).padStart(2, "0")}</span>
                    <span className="tile-title">{item.label}</span>
                    <span className="tile-open" aria-hidden="true">↗</span>
                  </span>
                  <span className="tile-chart" aria-hidden="true">
                    <ChartPreview compact demo={item.id} />
                  </span>
                  <span className="tile-footer">
                    <span>{item.description}</span>
                    <span>{item.dataSummary}</span>
                  </span>
                </button>
              ))}
            </div>
          </section>
        </>
      )}

      <footer className="terminal-statusbar">
        <span><i aria-hidden="true" /> ready</span>
        <span className="shortcut"><kbd>↑↓←→</kbd> navigate</span>
        <span className="shortcut"><kbd>enter</kbd> open</span>
        {demo ? <span className="shortcut"><kbd>esc</kbd> gallery</span> : <span className="shortcut"><kbd>1–9</kbd> quick open</span>}
        <a href="https://github.com/mertdeveci5/chartmux">source ↗</a>
      </footer>
    </main>
  );
}

export default App;
