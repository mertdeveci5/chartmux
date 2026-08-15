export type DemoId =
  | "line"
  | "grouped-bar"
  | "stacked-bar"
  | "normalized-bar"
  | "annotated-bar"
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
  { id: "annotated-bar", label: "Annotated bar", description: "Pair data with narrative context", dataSummary: "6 rows · 2 notes" },
  { id: "horizontal-bar", label: "Horizontal bar", description: "Make long labels readable", dataSummary: "6 rows · 2 series" },
  { id: "area", label: "Area", description: "Emphasize volume over time", dataSummary: "6 rows · 2 series" },
  { id: "stacked-area", label: "Stacked area", description: "Track cumulative growth", dataSummary: "6 rows · 2 series" },
  { id: "combo", label: "Bar + line", description: "Mix measures on shared axes", dataSummary: "6 rows · 2 marks" },
  { id: "scatter", label: "Scatter", description: "Reveal numeric relationships", dataSummary: "6 points · 1 series" },
  { id: "histogram", label: "Histogram", description: "Inspect a distribution", dataSummary: "12 values · 6 bins" },
  { id: "pie", label: "Pie", description: "Show categorical proportions", dataSummary: "5 slices · 1 series" },
  { id: "donut", label: "Donut", description: "Show proportions as a ring", dataSummary: "5 slices · 1 series" },
  { id: "heatmap", label: "Heatmap", description: "Scan intensity across a matrix", dataSummary: "5 days · 3 periods" },
  { id: "radar", label: "Radar", description: "Compare shared dimensions", dataSummary: "5 metrics · 2 series" },
  { id: "funnel", label: "Funnel", description: "Follow conversion stages", dataSummary: "4 stages · 1 series" },
];
