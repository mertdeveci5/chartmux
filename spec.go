package chartmux

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const SpecVersion = 1

type Type string

const (
	Bar       Type = "bar"
	Line      Type = "line"
	Area      Type = "area"
	Combo     Type = "combo"
	Scatter   Type = "scatter"
	Histogram Type = "histogram"
	Pie       Type = "pie"
	Donut     Type = "donut"
	Heatmap   Type = "heatmap"
	Radar     Type = "radar"
	Funnel    Type = "funnel"
)

type Layout string

const (
	Grouped    Layout = "grouped"
	Overlay    Layout = "overlay"
	Stacked    Layout = "stacked"
	Normalized Layout = "normalized"
)

type Orientation string

const (
	Vertical   Orientation = "vertical"
	Horizontal Orientation = "horizontal"
)

type Curve string

const (
	Linear Curve = "linear"
	Smooth Curve = "smooth"
)

type Mark string

const (
	MarkBar  Mark = "bar"
	MarkLine Mark = "line"
)

type Row map[string]any

type AxisSpec struct {
	DataKey string `json:"dataKey,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

type SeriesSpec struct {
	DataKey string `json:"dataKey"`
	Label   string `json:"label,omitempty"`
	Color   string `json:"color,omitempty"`
	Mark    Mark   `json:"mark,omitempty"`
}

type DisplaySpec struct {
	Show bool `json:"show"`
}

type Spec struct {
	Schema      string       `json:"$schema,omitempty"`
	Version     int          `json:"version"`
	Type        Type         `json:"type"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Footer      string       `json:"footer,omitempty"`
	Data        []Row        `json:"data"`
	XAxis       AxisSpec     `json:"xAxis,omitzero"`
	Series      []SeriesSpec `json:"series"`
	Layout      Layout       `json:"layout,omitempty"`
	Orientation Orientation  `json:"orientation,omitempty"`
	Curve       Curve        `json:"curve,omitempty"`
	Legend      *DisplaySpec `json:"legend,omitempty"`
	Axes        *DisplaySpec `json:"axes,omitempty"`
	Labels      *DisplaySpec `json:"labels,omitempty"`
	Theme       string       `json:"theme,omitempty"`
	Bins        int          `json:"bins,omitempty"`
	Max         float64      `json:"max,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type compiledSeries struct {
	spec   SeriesSpec
	values []float64
}

type Chart struct {
	spec    Spec
	labels  []string
	xValues []float64
	series  []compiledSeries
}

func Types() []Type {
	return []Type{Bar, Line, Area, Combo, Scatter, Histogram, Pie, Donut, Heatmap, Radar, Funnel}
}

func New(spec Spec) (*Chart, error) {
	spec = cloneSpec(spec)
	spec.applyDefaults()
	if err := spec.validate(); err != nil {
		return nil, err
	}
	chart := &Chart{spec: spec}
	if spec.Type == Histogram {
		if err := chart.compileHistogram(); err != nil {
			return nil, err
		}
		return chart, nil
	}

	chart.labels = make([]string, len(spec.Data))
	if spec.Type == Scatter {
		chart.xValues = make([]float64, len(spec.Data))
	}
	chart.series = make([]compiledSeries, len(spec.Series))
	for index, series := range spec.Series {
		chart.series[index] = compiledSeries{spec: series, values: make([]float64, len(spec.Data))}
	}
	for rowIndex, row := range spec.Data {
		label, ok := row[spec.XAxis.DataKey]
		if !ok {
			return nil, fmt.Errorf("data row %d is missing xAxis.dataKey %q", rowIndex+1, spec.XAxis.DataKey)
		}
		chart.labels[rowIndex] = scalarLabel(label)
		if err := validateInlineText(fmt.Sprintf("data row %d x-axis label", rowIndex+1), chart.labels[rowIndex], maxAxisLabelLength); err != nil {
			return nil, err
		}
		if spec.Type == Scatter {
			value, missing, err := numericValue(label)
			if err != nil || missing {
				return nil, fmt.Errorf("data row %d field %q must be numeric for scatter charts", rowIndex+1, spec.XAxis.DataKey)
			}
			chart.xValues[rowIndex] = value
		}
		for seriesIndex := range chart.series {
			key := chart.series[seriesIndex].spec.DataKey
			raw, ok := row[key]
			if !ok {
				return nil, fmt.Errorf("data row %d is missing series dataKey %q", rowIndex+1, key)
			}
			value, missing, err := numericValue(raw)
			if err != nil {
				return nil, fmt.Errorf("data row %d field %q: %w", rowIndex+1, key, err)
			}
			if missing {
				chart.series[seriesIndex].values[rowIndex] = math.MaxFloat64
				continue
			}
			if requiresNonNegative(spec.Type) && value < 0 {
				return nil, fmt.Errorf("data row %d field %q must be non-negative for %s charts", rowIndex+1, key, spec.Type)
			}
			if spec.Layout == Normalized && value < 0 {
				return nil, fmt.Errorf("data row %d field %q must be non-negative for normalized charts", rowIndex+1, key)
			}
			chart.series[seriesIndex].values[rowIndex] = value
		}
	}
	if spec.Layout == Normalized {
		if err := normalizeSeries(chart.series); err != nil {
			return nil, err
		}
	}
	if spec.Type == Radar && spec.Max > 0 {
		for _, series := range chart.series {
			for pointIndex, value := range series.values {
				if !isMissing(value) && value > spec.Max {
					return nil, fmt.Errorf("data row %d field %q exceeds radar max %s", pointIndex+1, series.spec.DataKey, formatValue(spec.Max))
				}
			}
		}
	}
	return chart, nil
}

func (chart *Chart) Spec() Spec {
	return cloneSpec(chart.spec)
}

func (chart *Chart) ResolvedSpec() Spec {
	spec := cloneSpec(chart.spec)
	if spec.Schema == "" {
		spec.Schema = SchemaURL
	}
	if spec.Legend == nil {
		show := len(chart.series) > 1
		if spec.Type == Pie || spec.Type == Donut || spec.Type == Combo {
			show = true
		}
		if spec.Type == Funnel {
			show = false
		}
		spec.Legend = &DisplaySpec{Show: show}
	}
	if spec.Axes == nil {
		spec.Axes = &DisplaySpec{Show: true}
	}
	if spec.Labels == nil {
		spec.Labels = &DisplaySpec{Show: false}
	}
	for index := range spec.Annotations {
		if spec.Annotations[index].Position == "" {
			spec.Annotations[index].Position = AnnotationBottom
		}
	}
	return spec
}

func (spec *Spec) applyDefaults() {
	if spec.Version == 0 {
		spec.Version = SpecVersion
	}
	if spec.Theme == "" {
		spec.Theme = "light"
	}
	if spec.Orientation == "" && (spec.Type == Bar || spec.Type == Histogram) {
		spec.Orientation = Vertical
	}
	if spec.Curve == "" && (spec.Type == Line || spec.Type == Area) {
		spec.Curve = Linear
	}
	if spec.Layout == "" {
		if spec.Type == Area {
			spec.Layout = Overlay
		} else if spec.Type == Bar || spec.Type == Histogram {
			spec.Layout = Grouped
		}
	}
	for index := range spec.Series {
		if spec.Series[index].Label == "" {
			spec.Series[index].Label = humanize(spec.Series[index].DataKey)
		}
		if spec.Series[index].Color == "" {
			spec.Series[index].Color = defaultColors[index%len(defaultColors)]
		}
	}
}

func (spec Spec) validate() error {
	if spec.Version != SpecVersion {
		return fmt.Errorf("unsupported chart spec version %d; expected %d", spec.Version, SpecVersion)
	}
	if !validType(spec.Type) {
		return fmt.Errorf("unsupported chart type %q", spec.Type)
	}
	if len(spec.Data) == 0 {
		return fmt.Errorf("chart spec has no data")
	}
	if len(spec.Data) > 1000 {
		return fmt.Errorf("chart spec supports at most 1000 data rows")
	}
	if len(spec.Series) == 0 {
		return fmt.Errorf("chart spec has no series")
	}
	if err := validateTextFields(spec); err != nil {
		return err
	}
	if spec.Type != Histogram && spec.XAxis.DataKey == "" {
		return fmt.Errorf("chart spec is missing xAxis.dataKey")
	}
	if spec.Type == Histogram && spec.XAxis.DataKey == "" && spec.XAxis.Kind != "" {
		return fmt.Errorf("histogram xAxis.kind requires xAxis.dataKey")
	}
	if spec.XAxis.DataKey != "" {
		if err := validateInlineText("xAxis.dataKey", spec.XAxis.DataKey, maxDataKeyLength); err != nil {
			return err
		}
	}
	if spec.Theme != "light" && spec.Theme != "dark" {
		return fmt.Errorf("theme must be light or dark")
	}
	if spec.XAxis.Kind != "" && spec.XAxis.Kind != "category" && spec.XAxis.Kind != "number" && spec.XAxis.Kind != "time" {
		return fmt.Errorf("xAxis.kind must be category, number, or time")
	}
	if spec.Type == Scatter && spec.XAxis.Kind != "" && spec.XAxis.Kind != "number" {
		return fmt.Errorf("scatter xAxis.kind must be number")
	}
	if spec.Curve != "" {
		if spec.Type != Line && spec.Type != Area {
			return fmt.Errorf("curve is not valid for %s charts", spec.Type)
		}
		if spec.Curve != Linear && spec.Curve != Smooth {
			return fmt.Errorf("curve must be linear or smooth")
		}
	}
	if spec.Orientation != "" {
		if spec.Type != Bar && spec.Type != Histogram {
			return fmt.Errorf("orientation is only valid for bar and histogram charts")
		}
		if spec.Orientation != Vertical && spec.Orientation != Horizontal {
			return fmt.Errorf("orientation must be vertical or horizontal")
		}
	}
	if err := validateLayout(spec.Type, spec.Layout); err != nil {
		return err
	}
	if spec.Bins < 0 || spec.Bins > 100 {
		return fmt.Errorf("bins must be 0 for automatic or between 1 and 100")
	}
	if spec.Bins != 0 && spec.Type != Histogram {
		return fmt.Errorf("bins is only valid for histogram charts")
	}
	if spec.Max != 0 && spec.Type != Radar {
		return fmt.Errorf("max is only valid for radar charts")
	}
	if spec.Max < 0 {
		return fmt.Errorf("max must be zero for automatic or greater than zero")
	}
	if spec.Type == Histogram && len(spec.Series) != 1 {
		return fmt.Errorf("histogram charts require exactly one series")
	}
	if (spec.Type == Pie || spec.Type == Donut || spec.Type == Funnel) && len(spec.Series) != 1 {
		return fmt.Errorf("%s charts require exactly one series", spec.Type)
	}
	if spec.Type == Radar && len(spec.Data) < 3 {
		return fmt.Errorf("radar charts require at least three data rows")
	}
	seen := make(map[string]bool, len(spec.Series))
	for index, series := range spec.Series {
		if series.DataKey == "" {
			return fmt.Errorf("series %d is missing dataKey", index+1)
		}
		if err := validateInlineText(fmt.Sprintf("series %d dataKey", index+1), series.DataKey, maxDataKeyLength); err != nil {
			return err
		}
		if err := validateInlineText(fmt.Sprintf("series %q label", series.DataKey), series.Label, maxSeriesLabelLength); err != nil {
			return err
		}
		if seen[series.DataKey] {
			return fmt.Errorf("duplicate series dataKey %q", series.DataKey)
		}
		seen[series.DataKey] = true
		if spec.Type == Combo {
			if series.Mark != MarkBar && series.Mark != MarkLine {
				return fmt.Errorf("combo series %q mark must be bar or line", series.DataKey)
			}
		} else if series.Mark != "" {
			return fmt.Errorf("series mark is only valid for combo charts")
		}
		if _, err := colorHex(series.Color); err != nil {
			return fmt.Errorf("series %q: %w", series.DataKey, err)
		}
	}
	if err := validateAnnotations(spec, seen); err != nil {
		return err
	}
	return nil
}

func validateLayout(chartType Type, layout Layout) error {
	switch chartType {
	case Bar, Histogram:
		if layout != Grouped && layout != Stacked && layout != Normalized {
			return fmt.Errorf("%s layout must be grouped, stacked, or normalized", chartType)
		}
	case Area:
		if layout != Overlay && layout != Stacked && layout != Normalized {
			return fmt.Errorf("area layout must be overlay, stacked, or normalized")
		}
	default:
		if layout != "" {
			return fmt.Errorf("layout is not valid for %s charts", chartType)
		}
	}
	return nil
}

func validType(value Type) bool {
	for _, chartType := range Types() {
		if value == chartType {
			return true
		}
	}
	return false
}

func requiresNonNegative(chartType Type) bool {
	return chartType == Pie || chartType == Donut || chartType == Funnel || chartType == Radar || chartType == Heatmap
}

func displayValue(config *DisplaySpec, fallback bool) bool {
	if config == nil {
		return fallback
	}
	return config.Show
}

func numericValue(value any) (float64, bool, error) {
	if value == nil {
		return 0, true, nil
	}
	var number float64
	switch value := value.(type) {
	case float64:
		number = value
	case float32:
		number = float64(value)
	case int:
		number = float64(value)
	case int64:
		number = float64(value)
	case uint64:
		number = float64(value)
	case string:
		trimmed := strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
		if trimmed == "" {
			return 0, true, nil
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return 0, false, fmt.Errorf("%q is not numeric", value)
		}
		number = parsed
	default:
		if stringer, ok := value.(fmt.Stringer); ok {
			parsed, err := strconv.ParseFloat(stringer.String(), 64)
			if err == nil {
				number = parsed
				break
			}
		}
		return 0, false, fmt.Errorf("value must be numeric or null")
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false, fmt.Errorf("value must be finite")
	}
	if number == math.MaxFloat64 {
		return 0, false, fmt.Errorf("value is too large to render")
	}
	return number, false, nil
}

func isMissing(value float64) bool {
	return value == math.MaxFloat64
}

func scalarLabel(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func normalizeSeries(series []compiledSeries) error {
	if len(series) == 0 {
		return nil
	}
	for point := range series[0].values {
		total := 0.0
		hasValue := false
		for index := range series {
			value := series[index].values[point]
			if !isMissing(value) {
				hasValue = true
				total += value
			}
		}
		if math.IsInf(total, 0) || math.IsNaN(total) {
			return fmt.Errorf("normalized layout data row %d total is too large to render", point+1)
		}
		if !hasValue {
			continue
		}
		if total == 0 {
			return fmt.Errorf("normalized layout data row %d has no positive values", point+1)
		}
		for index := range series {
			if !isMissing(series[index].values[point]) {
				series[index].values[point] = series[index].values[point] / total * 100
			}
		}
	}
	return nil
}

func cloneSpec(spec Spec) Spec {
	clone := spec
	clone.Data = make([]Row, len(spec.Data))
	for index, row := range spec.Data {
		clone.Data[index] = make(Row, len(row))
		for key, value := range row {
			clone.Data[index][key] = value
		}
	}
	clone.Series = append([]SeriesSpec(nil), spec.Series...)
	clone.Legend = cloneDisplaySpec(spec.Legend)
	clone.Axes = cloneDisplaySpec(spec.Axes)
	clone.Labels = cloneDisplaySpec(spec.Labels)
	clone.Annotations = make([]Annotation, len(spec.Annotations))
	for index, annotation := range spec.Annotations {
		clone.Annotations[index] = annotation
		if annotation.DataIndex != nil {
			value := *annotation.DataIndex
			clone.Annotations[index].DataIndex = &value
		}
	}
	return clone
}

func cloneDisplaySpec(value *DisplaySpec) *DisplaySpec {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func (chart *Chart) compileHistogram() error {
	key := chart.spec.Series[0].DataKey
	values := make([]float64, 0, len(chart.spec.Data))
	for rowIndex, row := range chart.spec.Data {
		raw, ok := row[key]
		if !ok {
			return fmt.Errorf("data row %d is missing series dataKey %q", rowIndex+1, key)
		}
		value, missing, err := numericValue(raw)
		if err != nil {
			return fmt.Errorf("data row %d field %q: %w", rowIndex+1, key, err)
		}
		if !missing {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return fmt.Errorf("histogram series %q has no numeric values", key)
	}
	minimum, maximum := values[0], values[0]
	for _, value := range values[1:] {
		minimum = math.Min(minimum, value)
		maximum = math.Max(maximum, value)
	}
	bins := chart.spec.Bins
	if bins == 0 {
		bins = max(1, int(math.Ceil(math.Sqrt(float64(len(values))))))
	}
	if minimum == maximum {
		bins = 1
	}
	counts := make([]float64, bins)
	step := (maximum - minimum) / float64(bins)
	if step == 0 {
		step = 1
	}
	chart.labels = make([]string, bins)
	for index := range bins {
		start := minimum + float64(index)*step
		end := start + step
		chart.labels[index] = formatBinBoundary(start) + "–" + formatBinBoundary(end)
	}
	for _, value := range values {
		index := min(bins-1, int((value-minimum)/step))
		counts[index]++
	}
	chart.series = []compiledSeries{{spec: chart.spec.Series[0], values: counts}}
	return nil
}

func formatBinBoundary(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}

func humanize(value string) string {
	value = strings.TrimSpace(strings.NewReplacer("_", " ", "-", " ").Replace(value))
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

var defaultColors = []string{"#2563EB", "#60A5FA", "#34D399", "#F59E0B", "#F87171", "#A78BFA"}
