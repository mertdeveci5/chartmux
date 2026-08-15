import {
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  CopyIcon,
  ExternalLinkIcon,
  GithubIcon,
  Maximize2Icon,
  PackageIcon,
  XIcon,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { demos, type DemoId } from "./demos";

type ExportFormat = "terminal" | "png" | "svg" | "html";

const repositoryUrl = "https://github.com/mertdeveci5/chartmux";
const installCommand = "brew install mertdeveci5/tap/chartmux";

const installOptions = [
  {
    description: "Install the native binary through the Homebrew tap.",
    label: "Homebrew",
    value: installCommand,
  },
  {
    description: "Install the platform wrapper globally with npm.",
    label: "npm",
    value: "npm install --global chartmux",
  },
  {
    description: "Build and install the latest command directly from Go.",
    label: "Go",
    value: "go install github.com/mertdeveci5/chartmux/cmd/chartmux@latest",
  },
] as const;

function demoAsset(id: DemoId) {
  return `/demos/${id}.ansi.txt`;
}

function chartCommand(demo: DemoId, format: ExportFormat) {
  if (format === "terminal") return `chartmux demo ${demo} --watch`;
  return `chartmux demo ${demo} --theme dark --export ${format} --output ${demo}.${format}`;
}

function BrandMark({ compact = false }: { compact?: boolean }) {
  return (
    <svg aria-hidden="true" className={compact ? "brand-mark compact" : "brand-mark"} viewBox="0 0 28 28">
      <rect fill="currentColor" height="5" opacity="0.38" rx="1.5" width="4" x="4" y="17" />
      <rect fill="currentColor" height="9" opacity="0.58" rx="1.5" width="4" x="10" y="13" />
      <rect fill="currentColor" height="14" opacity="0.82" rx="1.5" width="4" x="16" y="8" />
      <path d="m4 14 6-4 6 1 8-7" fill="none" stroke="white" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" />
      <circle cx="24" cy="4" fill="white" r="2" />
    </svg>
  );
}

function CopyButton({
  className = "",
  label,
  size = "medium",
  value,
  variant = "outline",
}: {
  className?: string;
  label: string;
  size?: "small" | "medium" | "large";
  value: string;
  variant?: "primary" | "outline" | "ghost";
}) {
  const [status, setStatus] = useState<"idle" | "copied" | "failed">("idle");

  useEffect(() => {
    if (status === "idle") return;
    const timer = window.setTimeout(() => setStatus("idle"), 1800);
    return () => window.clearTimeout(timer);
  }, [status]);

  function copy() {
    void navigator.clipboard.writeText(value).then(
      () => setStatus("copied"),
      () => setStatus("failed"),
    );
  }

  let message = label;
  if (status === "copied") message = "Copied";
  if (status === "failed") message = "Copy failed";

  return (
    <button
      aria-live="polite"
      className={`button ${variant} ${size} ${className}`.trim()}
      onClick={copy}
      type="button"
    >
      {status === "copied" ? <CheckIcon aria-hidden="true" /> : <CopyIcon aria-hidden="true" />}
      {message}
    </button>
  );
}

type AnsiSegment = {
  bold: boolean;
  color?: string;
  dim: boolean;
  text: string;
};

function parseAnsi(input: string): AnsiSegment[] {
  const segments: AnsiSegment[] = [];
  const expression = /\x1b\[([0-9;]*)m/g;
  let bold = false;
  let color: string | undefined;
  let dim = false;
  let cursor = 0;

  function append(text: string) {
    if (!text) return;
    const previous = segments.at(-1);
    if (previous && previous.bold === bold && previous.color === color && previous.dim === dim) {
      previous.text += text;
      return;
    }
    segments.push({ bold, color, dim, text });
  }

  for (const match of input.matchAll(expression)) {
    append(input.slice(cursor, match.index));
    const codes = match[1] === "" ? [0] : match[1].split(";").map(Number);
    for (let index = 0; index < codes.length; index += 1) {
      const code = codes[index];
      if (code === 0) {
        bold = false;
        color = undefined;
        dim = false;
      } else if (code === 1) {
        bold = true;
      } else if (code === 2) {
        dim = true;
      } else if (code === 22) {
        bold = false;
        dim = false;
      } else if (code === 39) {
        color = undefined;
      } else if (code === 38 && codes[index + 1] === 2) {
        color = `rgb(${codes[index + 2]} ${codes[index + 3]} ${codes[index + 4]})`;
        index += 4;
      }
    }
    cursor = (match.index ?? 0) + match[0].length;
  }
  append(input.slice(cursor));
  return segments;
}

function TerminalChart({ demo }: { demo: DemoId }) {
  const [content, setContent] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    fetch(demoAsset(demo), { signal: controller.signal })
      .then((response) => {
        if (!response.ok) throw new Error(`Unable to load terminal demo ${demo}`);
        return response.text();
      })
      .then(setContent)
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setContent("Terminal preview unavailable");
      });
    return () => controller.abort();
  }, [demo]);

  const segments = useMemo(() => parseAnsi(content ?? "Rendering terminal output…"), [content]);
  return (
    <div className="terminal-viewport">
      <pre className="terminal-output" aria-label={`Actual chartmux terminal output: ${demo}`} role="img">
        {segments.map((segment, index) => (
          <span
            key={`${index}-${segment.text.length}`}
            style={{
              color: segment.color,
              fontWeight: segment.bold ? 700 : undefined,
              opacity: segment.dim ? 0.48 : undefined,
            }}
          >
            {segment.text}
          </span>
        ))}
      </pre>
    </div>
  );
}

function SiteChrome() {
  const navItems = [
    { href: "#overview", label: "Overview" },
    { href: "#demo", label: "Demo" },
    { href: "#examples", label: "Examples" },
    { href: "#install", label: "Install" },
  ];

  return (
    <>
      <header className="site-header">
        <div className="site-header-row shell">
          <a className="header-brand" href="#overview">
            <BrandMark compact />
            <span className="hand-underline">chartmux</span>
          </a>
          <div className="header-actions">
            <a href={repositoryUrl} rel="noreferrer noopener" target="_blank">GitHub</a>
            <CopyButton className="site-header-install" label="Install" size="small" value={installCommand} variant="primary" />
          </div>
        </div>
        <nav className="mobile-nav shell" aria-label="On this page">
          {navItems.map((item) => <a href={item.href} key={item.href}>{item.label}</a>)}
        </nav>
      </header>

      <aside className="site-sidebar" aria-label="Primary navigation">
        <a className="sidebar-brand" href="#overview">
          <BrandMark />
          <span className="hand-underline">chartmux</span>
        </a>
        <nav className="sidebar-nav" aria-label="On this page">
          {navItems.map((item) => <a href={item.href} key={item.href}>{item.label}</a>)}
        </nav>
        <div className="sidebar-footer">
          <span className="sidebar-version">v0.1.0</span>
          <a href={repositoryUrl} rel="noreferrer noopener" target="_blank">
            <GithubIcon aria-hidden="true" /> GitHub <ExternalLinkIcon aria-hidden="true" className="sidebar-external" />
          </a>
          <a href={`${repositoryUrl}/releases`} rel="noreferrer noopener" target="_blank">
            Releases <ExternalLinkIcon aria-hidden="true" className="sidebar-external" />
          </a>
          <CopyButton className="sidebar-install" label="Install" size="small" value={installCommand} variant="primary" />
        </div>
      </aside>
    </>
  );
}

function TerminalStage({ demo }: { demo: DemoId }) {
  return (
    <div className="terminal-stage">
      <div className="terminal-window-bar" aria-hidden="true">
        <span className="traffic-lights"><i /><i /><i /></span>
        <span>chartmux demo {demo}</span>
      </div>
      <div className="terminal-scene">
        <TerminalChart demo={demo} />
      </div>
    </div>
  );
}

function DemoFrame({
  demo,
  format,
  onExpand,
  onFormatChange,
  onNext,
  onPrevious,
}: {
  demo: DemoId;
  format: ExportFormat;
  onExpand: () => void;
  onFormatChange: (format: ExportFormat) => void;
  onNext: () => void;
  onPrevious: () => void;
}) {
  const item = demos.find((candidate) => candidate.id === demo) ?? demos[0];
  const command = chartCommand(demo, format);

  return (
    <figure className="demo-figure">
      <div className="demo-frame">
        <div className="demo-viewport"><TerminalStage demo={demo} /></div>
        <div className="demo-bar">
          <div className="demo-controls">
            <button aria-label="Previous chart" className="demo-icon-button" onClick={onPrevious} type="button">
              <ChevronLeftIcon aria-hidden="true" />
            </button>
            <span className="demo-current">{item.label}</span>
            <button aria-label="Next chart" className="demo-icon-button" onClick={onNext} type="button">
              <ChevronRightIcon aria-hidden="true" />
            </button>
            <div aria-label="Export format" className="demo-segmented" role="group">
              {(["terminal", "png", "svg", "html"] as const).map((value) => (
                <button
                  aria-pressed={format === value}
                  className="demo-segment"
                  key={value}
                  onClick={() => onFormatChange(value)}
                  type="button"
                >
                  {value}
                </button>
              ))}
            </div>
          </div>
          <div className="demo-actions">
            <CopyButton label="Copy command" size="small" value={command} variant="ghost" />
            <button className="demo-expand" onClick={onExpand} type="button">
              <Maximize2Icon aria-hidden="true" /> Expand
            </button>
          </div>
        </div>
      </div>
      <figcaption className="demo-caption">{command}</figcaption>
    </figure>
  );
}

function ChartDialog({ demo, onClose }: { demo: DemoId; onClose: () => void }) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const item = demos.find((candidate) => candidate.id === demo) ?? demos[0];

  useEffect(() => {
    dialogRef.current?.showModal();
    return () => dialogRef.current?.close();
  }, []);

  return (
    <dialog
      aria-labelledby="dialog-title"
      className="chart-dialog"
      onCancel={(event) => { event.preventDefault(); onClose(); }}
      onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}
      ref={dialogRef}
    >
      <div className="dialog-card">
        <div className="dialog-header">
          <div>
            <h2 id="dialog-title">{item.label}</h2>
            <p>Actual 80-column terminal output from the package.</p>
          </div>
          <button aria-label="Close preview" className="dialog-close" onClick={onClose} type="button"><XIcon aria-hidden="true" /></button>
        </div>
        <TerminalStage demo={demo} />
      </div>
    </dialog>
  );
}

function App() {
  const [selected, setSelected] = useState<DemoId>("line");
  const [format, setFormat] = useState<ExportFormat>("terminal");
  const [expanded, setExpanded] = useState(false);
  const selectedIndex = demos.findIndex((item) => item.id === selected);

  function selectDemo(id: DemoId) {
    setSelected(id);
    window.requestAnimationFrame(() => document.querySelector("#demo")?.scrollIntoView({ behavior: "smooth" }));
  }

  function moveDemo(offset: number) {
    const nextIndex = (selectedIndex + offset + demos.length) % demos.length;
    setSelected(demos[nextIndex].id);
  }

  return (
    <div className="page">
      <a className="skip-link" href="#demo">Skip to the demo</a>
      <SiteChrome />

      <div className="site-content">
        <main>
          <section className="overview-section shell" id="overview">
            <p className="eyebrow">Terminal · PNG · SVG · HTML</p>
            <h1 className="hero-title"><span className="marker">Terminal-native charts</span> for every surface.</h1>
            <p className="lede">Chartmux turns CSV, JSON, or a versioned chart spec into polished output with one deterministic renderer. The demo below is captured from the CLI itself.</p>
            <div className="hero-actions">
              <CopyButton className="mobile-primary-action" label="Copy install command" size="large" value={installCommand} variant="primary" />
              <a className="button outline large mobile-primary-action" href={repositoryUrl} rel="noreferrer noopener" target="_blank">
                <GithubIcon aria-hidden="true" /> View on GitHub
              </a>
              <span className="release-meta"><span>v0.1.0</span> macOS · Linux · Windows</span>
            </div>
          </section>

          <section className="demo-section shell" id="demo">
            <DemoFrame
              demo={selected}
              format={format}
              onExpand={() => setExpanded(true)}
              onFormatChange={setFormat}
              onNext={() => moveDemo(1)}
              onPrevious={() => moveDemo(-1)}
            />
          </section>

          <section className="content-section shell" id="examples">
            <div className="section-header">
              <div>
                <p className="eyebrow">Examples</p>
                <h2>Explore the built-in charts</h2>
                <p>Choose any renderer-owned demo and it opens in the player above.</p>
              </div>
              <span className="section-count">{demos.length} demos</span>
            </div>
            <div className="demo-list">
              {demos.map((item, index) => (
                <button
                  aria-pressed={selected === item.id}
                  className="demo-list-item"
                  key={item.id}
                  onClick={() => selectDemo(item.id)}
                  type="button"
                >
                  <span className="demo-list-index">{String(index + 1).padStart(2, "0")}</span>
                  <span className="demo-list-copy"><strong>{item.label}</strong><small>{item.description}</small></span>
                  <ChevronRightIcon aria-hidden="true" />
                </button>
              ))}
            </div>
          </section>

          <section className="content-section install-section shell" id="install">
            <div className="section-header">
              <div>
                <p className="eyebrow">Install</p>
                <h2>Get chartmux running</h2>
                <p>Pick the package manager already used in your workflow.</p>
              </div>
              <PackageIcon aria-hidden="true" className="section-icon" />
            </div>
            <ol className="install-timeline">
              {installOptions.map((option, index) => (
                <li key={option.label}>
                  <span className="timeline-indicator">{index + 1}</span>
                  <span className="timeline-copy"><strong>{option.label}</strong><small>{option.description}</small></span>
                  <CopyButton label="Copy" size="small" value={option.value} variant="outline" />
                </li>
              ))}
            </ol>
            <p className="install-note">Prebuilt binaries are also available from <a className="text-link" href={`${repositoryUrl}/releases`} rel="noreferrer noopener" target="_blank">GitHub Releases</a>.</p>
          </section>
        </main>

        <footer className="site-footer">
          <div className="site-footer-content shell">
            <span>chartmux v0.1.0</span>
            <span><a href={repositoryUrl} rel="noreferrer noopener" target="_blank">GitHub</a><a href={`${repositoryUrl}/releases`} rel="noreferrer noopener" target="_blank">Releases</a></span>
          </div>
        </footer>
      </div>

      {expanded ? <ChartDialog demo={selected} onClose={() => setExpanded(false)} /> : null}
    </div>
  );
}

export default App;
