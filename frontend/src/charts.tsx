import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ComposedChart,
  Funnel,
  FunnelChart,
  LabelList,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  PolarAngleAxis,
  PolarGrid,
  Radar,
  RadarChart,
  ResponsiveContainer,
  Scatter,
  ScatterChart,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export type DemoId =
  | "line"
  | "grouped-bar"
  | "stacked-bar"
  | "normalized-bar"
  | "horizontal-bar"
  | "area"
  | "stacked-area"
  | "combo"
  | "scatter"
  | "histogram"
  | "pie"
  | "donut"
  | "heatmap"
  | "radar"
  | "funnel";

export type Demo = {
  id: DemoId;
  label: string;
  description: string;
  dataSummary: string;
};

export const demos: Demo[] = [
  { id: "line", label: "Line", description: "Compare trends over time", dataSummary: "6 rows · 2 series" },
  { id: "grouped-bar", label: "Grouped bar", description: "Compare series side by side", dataSummary: "6 rows · 2 series" },
  { id: "stacked-bar", label: "Stacked bar", description: "Show composition and totals", dataSummary: "6 rows · 2 series" },
  { id: "normalized-bar", label: "100% stacked", description: "Compare proportional mixes", dataSummary: "6 rows · 2 series" },
  { id: "horizontal-bar", label: "Horizontal bar", description: "Make long labels readable", dataSummary: "6 rows · 2 series" },
  { id: "area", label: "Area", description: "Emphasize volume over time", dataSummary: "6 rows · 2 series" },
  { id: "stacked-area", label: "Stacked area", description: "Track cumulative growth", dataSummary: "6 rows · 2 series" },
  { id: "combo", label: "Bar + line", description: "Mix measures on shared axes", dataSummary: "6 rows · 2 marks" },
  { id: "scatter", label: "Scatter", description: "Reveal numeric relationships", dataSummary: "8 points · 1 series" },
  { id: "histogram", label: "Histogram", description: "Inspect a distribution", dataSummary: "12 values · 6 bins" },
  { id: "pie", label: "Pie", description: "Show categorical proportions", dataSummary: "5 slices · 1 series" },
  { id: "donut", label: "Donut", description: "Show proportions as a ring", dataSummary: "5 slices · 1 series" },
  { id: "heatmap", label: "Heatmap", description: "Scan intensity across a matrix", dataSummary: "5 days · 3 periods" },
  { id: "radar", label: "Radar", description: "Compare shared dimensions", dataSummary: "5 metrics · 2 series" },
  { id: "funnel", label: "Funnel", description: "Follow conversion stages", dataSummary: "4 stages · 1 series" },
];

const visitors = [
  { month: "Jan", desktop: 186, mobile: 80, target: 230 },
  { month: "Feb", desktop: 305, mobile: 200, target: 250 },
  { month: "Mar", desktop: 237, mobile: 120, target: 260 },
  { month: "Apr", desktop: 173, mobile: 190, target: 280 },
  { month: "May", desktop: 309, mobile: 230, target: 310 },
  { month: "Jun", desktop: 364, mobile: 280, target: 340 },
];

const scatterData = [
  { minutes: 1, pages: 2 },
  { minutes: 2, pages: 4 },
  { minutes: 3, pages: 3 },
  { minutes: 4, pages: 6 },
  { minutes: 5, pages: 7 },
  { minutes: 7, pages: 8 },
  { minutes: 9, pages: 12 },
  { minutes: 11, pages: 13 },
];

const histogramData = [
  { bin: "$0–20", orders: 1 },
  { bin: "$20–30", orders: 3 },
  { bin: "$30–40", orders: 3 },
  { bin: "$40–50", orders: 3 },
  { bin: "$50–70", orders: 1 },
  { bin: "$70+", orders: 1 },
];

const browserData = [
  { browser: "Chrome", visitors: 275 },
  { browser: "Safari", visitors: 200 },
  { browser: "Firefox", visitors: 187 },
  { browser: "Edge", visitors: 173 },
  { browser: "Other", visitors: 90 },
];

const radarData = [
  { metric: "Reach", desktop: 82, mobile: 91 },
  { metric: "Engage", desktop: 76, mobile: 84 },
  { metric: "Convert", desktop: 88, mobile: 69 },
  { metric: "Retain", desktop: 80, mobile: 72 },
  { metric: "Revenue", desktop: 93, mobile: 74 },
];

const funnelData = [
  { stage: "Visitors", users: 2400, fill: "#7367f0" },
  { stage: "Signups", users: 1320, fill: "#6f82ed" },
  { stage: "Trials", users: 760, fill: "#56a8e5" },
  { stage: "Customers", users: 410, fill: "#3fc5d8" },
];

const colors = ["#7367f0", "#3fc5d8", "#8f7cf4", "#55a2e5", "#9f9cf8"];
const axis = { fill: "#888f9d", fontSize: 11, fontFamily: "var(--font-mono)" };
const tooltipStyle = {
  background: "#15161a",
  border: "1px solid #33343a",
  borderRadius: "8px",
  boxShadow: "0 16px 48px rgba(0,0,0,.42)",
  color: "#f2f2f3",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
};

function ChartFrame({ children }: { children: React.ReactElement }) {
  return (
    <ResponsiveContainer height="100%" minHeight={80} minWidth={0} width="100%">
      {children}
    </ResponsiveContainer>
  );
}

function Axes({ compact, percent = false }: { compact: boolean; percent?: boolean }) {
  if (compact) {
    return <CartesianGrid stroke="#292a30" strokeDasharray="2 6" vertical={false} />;
  }

  return (
    <>
      <CartesianGrid stroke="#292a30" strokeDasharray="3 6" vertical={false} />
      <XAxis axisLine={false} dataKey="month" dy={8} tick={axis} tickLine={false} />
      <YAxis
        axisLine={false}
        tick={axis}
        tickFormatter={percent ? (value: number) => `${Math.round(value * 100)}%` : undefined}
        tickLine={false}
        width={54}
      />
      <Tooltip contentStyle={tooltipStyle} cursor={{ fill: "rgba(255,255,255,.035)" }} />
      <Legend iconSize={7} iconType="circle" wrapperStyle={{ fontSize: 11, paddingTop: 16 }} />
    </>
  );
}

function chartMargin(compact: boolean) {
  return compact
    ? { top: 12, right: 8, bottom: 5, left: 8 }
    : { top: 18, right: 18, bottom: 8, left: -8 };
}

function HeatmapPreview({ compact }: { compact: boolean }) {
  const values = [
    [22, 48, 35],
    [31, 56, 42],
    [28, 63, 51],
    [35, 58, 47],
    [41, 72, 65],
  ];
  const days = ["Mon", "Tue", "Wed", "Thu", "Fri"];
  const periods = ["Morning", "Afternoon", "Evening"];
  const left = compact ? 8 : 88;
  const top = compact ? 8 : 28;
  const cellWidth = compact ? 54 : 126;
  const cellHeight = compact ? 34 : 46;
  const gap = compact ? 5 : 8;

  return (
    <svg aria-hidden="true" className="heatmap-svg" preserveAspectRatio="xMidYMid meet" viewBox="0 0 560 280">
      {!compact && periods.map((period, index) => (
        <text fill="#888f9d" fontSize="11" key={period} textAnchor="middle" x={left + index * (cellWidth + gap) + cellWidth / 2} y="14">
          {period}
        </text>
      ))}
      {values.map((row, rowIndex) => (
        <g key={days[rowIndex]}>
          {!compact && <text fill="#888f9d" fontSize="11" textAnchor="end" x={left - 14} y={top + rowIndex * (cellHeight + gap) + cellHeight / 2 + 4}>{days[rowIndex]}</text>}
          {row.map((value, columnIndex) => (
            <rect
              fill="#7367f0"
              height={cellHeight}
              key={`${rowIndex}-${columnIndex}`}
              opacity={0.18 + value / 100}
              rx={compact ? 2 : 4}
              width={cellWidth}
              x={left + columnIndex * (cellWidth + gap)}
              y={top + rowIndex * (cellHeight + gap)}
            />
          ))}
        </g>
      ))}
    </svg>
  );
}

export function ChartPreview({ demo, compact = false }: { demo: DemoId; compact?: boolean }) {
  const margin = chartMargin(compact);
  const animation = !compact;

  switch (demo) {
    case "grouped-bar":
      return (
        <ChartFrame>
          <BarChart barGap={compact ? 2 : 5} data={visitors} margin={margin}>
            <Axes compact={compact} />
            <Bar dataKey="desktop" fill="#7367f0" isAnimationActive={animation} name="Desktop" radius={[4, 4, 1, 1]} />
            <Bar dataKey="mobile" fill="#3fc5d8" isAnimationActive={animation} name="Mobile" radius={[4, 4, 1, 1]} />
          </BarChart>
        </ChartFrame>
      );
    case "stacked-bar":
      return (
        <ChartFrame>
          <BarChart data={visitors} margin={margin}>
            <Axes compact={compact} />
            <Bar dataKey="desktop" fill="#7367f0" isAnimationActive={animation} name="Desktop" stackId="traffic" />
            <Bar dataKey="mobile" fill="#3fc5d8" isAnimationActive={animation} name="Mobile" radius={[4, 4, 0, 0]} stackId="traffic" />
          </BarChart>
        </ChartFrame>
      );
    case "normalized-bar":
      return (
        <ChartFrame>
          <BarChart data={visitors} margin={margin} stackOffset="expand">
            <Axes compact={compact} percent />
            <Bar dataKey="desktop" fill="#7367f0" isAnimationActive={animation} name="Desktop" stackId="traffic" />
            <Bar dataKey="mobile" fill="#3fc5d8" isAnimationActive={animation} name="Mobile" radius={[4, 4, 0, 0]} stackId="traffic" />
          </BarChart>
        </ChartFrame>
      );
    case "horizontal-bar":
      return (
        <ChartFrame>
          <BarChart data={visitors} layout="vertical" margin={compact ? { top: 8, right: 8, bottom: 8, left: 8 } : { top: 16, right: 24, bottom: 8, left: 0 }}>
            <CartesianGrid horizontal={false} stroke="#292a30" strokeDasharray="3 6" />
            <XAxis axisLine={false} hide={compact} tick={axis} tickLine={false} type="number" />
            <YAxis axisLine={false} dataKey="month" hide={compact} tick={axis} tickLine={false} type="category" width={52} />
            {!compact && <Tooltip contentStyle={tooltipStyle} cursor={{ fill: "rgba(255,255,255,.035)" }} />}
            {!compact && <Legend iconSize={7} iconType="circle" wrapperStyle={{ fontSize: 11, paddingTop: 16 }} />}
            <Bar dataKey="desktop" fill="#7367f0" isAnimationActive={animation} name="Desktop" radius={[0, 4, 4, 0]} />
            <Bar dataKey="mobile" fill="#3fc5d8" isAnimationActive={animation} name="Mobile" radius={[0, 4, 4, 0]} />
          </BarChart>
        </ChartFrame>
      );
    case "area":
    case "stacked-area":
      return (
        <ChartFrame>
          <AreaChart data={visitors} margin={margin}>
            <defs>
              <linearGradient id={`${demo}-desktop`} x1="0" x2="0" y1="0" y2="1">
                <stop offset="0%" stopColor="#7367f0" stopOpacity={0.72} />
                <stop offset="100%" stopColor="#7367f0" stopOpacity={0.05} />
              </linearGradient>
              <linearGradient id={`${demo}-mobile`} x1="0" x2="0" y1="0" y2="1">
                <stop offset="0%" stopColor="#3fc5d8" stopOpacity={0.6} />
                <stop offset="100%" stopColor="#3fc5d8" stopOpacity={0.04} />
              </linearGradient>
            </defs>
            <Axes compact={compact} />
            <Area dataKey="desktop" fill={`url(#${demo}-desktop)`} isAnimationActive={animation} name="Desktop" stackId={demo === "stacked-area" ? "traffic" : undefined} stroke="#8177f2" strokeWidth={2.2} type="monotone" />
            <Area dataKey="mobile" fill={`url(#${demo}-mobile)`} isAnimationActive={animation} name="Mobile" stackId={demo === "stacked-area" ? "traffic" : undefined} stroke="#3fc5d8" strokeWidth={2.2} type="monotone" />
          </AreaChart>
        </ChartFrame>
      );
    case "combo":
      return (
        <ChartFrame>
          <ComposedChart data={visitors} margin={margin}>
            <Axes compact={compact} />
            <Bar dataKey="desktop" fill="#7367f0" isAnimationActive={animation} maxBarSize={42} name="Revenue" radius={[4, 4, 1, 1]} />
            <Line dataKey="target" dot={compact ? false : { fill: "#111216", r: 3, strokeWidth: 2 }} isAnimationActive={animation} name="Target" stroke="#3fc5d8" strokeWidth={2.5} type="monotone" />
          </ComposedChart>
        </ChartFrame>
      );
    case "scatter":
      return (
        <ChartFrame>
          <ScatterChart margin={margin}>
            <CartesianGrid stroke="#292a30" strokeDasharray="3 6" />
            <XAxis axisLine={false} dataKey="minutes" hide={compact} name="Minutes" tick={axis} tickLine={false} type="number" />
            <YAxis axisLine={false} dataKey="pages" hide={compact} name="Pages" tick={axis} tickLine={false} type="number" width={50} />
            {!compact && <Tooltip contentStyle={tooltipStyle} cursor={{ stroke: "#555862", strokeDasharray: "3 4" }} />}
            <Scatter data={scatterData} fill="#7367f0" isAnimationActive={animation} name="Sessions" />
          </ScatterChart>
        </ChartFrame>
      );
    case "histogram":
      return (
        <ChartFrame>
          <BarChart barCategoryGap={compact ? 2 : 5} data={histogramData} margin={margin}>
            <CartesianGrid stroke="#292a30" strokeDasharray="3 6" vertical={false} />
            <XAxis axisLine={false} dataKey="bin" hide={compact} tick={axis} tickLine={false} />
            <YAxis axisLine={false} hide={compact} tick={axis} tickLine={false} width={42} />
            {!compact && <Tooltip contentStyle={tooltipStyle} cursor={{ fill: "rgba(255,255,255,.035)" }} />}
            <Bar dataKey="orders" fill="#7367f0" isAnimationActive={animation} name="Orders" radius={[3, 3, 0, 0]} />
          </BarChart>
        </ChartFrame>
      );
    case "pie":
    case "donut":
      return (
        <ChartFrame>
          <PieChart>
            <Pie
              cx="50%"
              cy="48%"
              data={browserData}
              dataKey="visitors"
              innerRadius={demo === "donut" ? (compact ? "42%" : "48%") : 0}
              isAnimationActive={animation}
              nameKey="browser"
              outerRadius={compact ? "72%" : "74%"}
              paddingAngle={demo === "donut" ? 2 : 1}
              stroke="#111216"
              strokeWidth={2}
            >
              {browserData.map((item, index) => <Cell fill={colors[index]} key={item.browser} />)}
            </Pie>
            {!compact && <Tooltip contentStyle={tooltipStyle} />}
            {!compact && <Legend iconSize={7} iconType="circle" wrapperStyle={{ fontSize: 11 }} />}
          </PieChart>
        </ChartFrame>
      );
    case "heatmap":
      return <HeatmapPreview compact={compact} />;
    case "radar":
      return (
        <ChartFrame>
          <RadarChart cx="50%" cy="50%" data={radarData} outerRadius={compact ? "75%" : "70%"}>
            <PolarGrid stroke="#33343a" />
            {!compact && <PolarAngleAxis dataKey="metric" tick={axis} />}
            <Radar dataKey="desktop" fill="#7367f0" fillOpacity={0.3} isAnimationActive={animation} name="Desktop" stroke="#8177f2" strokeWidth={2} />
            <Radar dataKey="mobile" fill="#3fc5d8" fillOpacity={0.2} isAnimationActive={animation} name="Mobile" stroke="#3fc5d8" strokeWidth={2} />
            {!compact && <Legend iconSize={7} iconType="circle" wrapperStyle={{ fontSize: 11 }} />}
            {!compact && <Tooltip contentStyle={tooltipStyle} />}
          </RadarChart>
        </ChartFrame>
      );
    case "funnel":
      return (
        <ChartFrame>
          <FunnelChart>
            {!compact && <Tooltip contentStyle={tooltipStyle} />}
            <Funnel data={funnelData} dataKey="users" isAnimationActive={animation} nameKey="stage" stroke="#111216">
              {funnelData.map((item) => <Cell fill={item.fill} key={item.stage} />)}
              {!compact && <LabelList dataKey="stage" fill="#f2f2f3" fontFamily="var(--font-mono)" fontSize={11} position="right" />}
            </Funnel>
          </FunnelChart>
        </ChartFrame>
      );
    case "line":
      return (
        <ChartFrame>
          <LineChart data={visitors} margin={margin}>
            <Axes compact={compact} />
            <Line dataKey="desktop" dot={compact ? false : { fill: "#111216", r: 3, strokeWidth: 2 }} isAnimationActive={animation} name="Desktop" stroke="#8177f2" strokeWidth={2.7} type="monotone" />
            <Line dataKey="mobile" dot={compact ? false : { fill: "#111216", r: 3, strokeWidth: 2 }} isAnimationActive={animation} name="Mobile" stroke="#3fc5d8" strokeWidth={2.3} type="monotone" />
          </LineChart>
        </ChartFrame>
      );
  }
}
