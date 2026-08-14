package chartmux

import "fmt"

var demoNames = []string{
	"line",
	"grouped-bar",
	"stacked-bar",
	"normalized-bar",
	"annotated-bar",
	"horizontal-bar",
	"area",
	"stacked-area",
	"combo",
	"scatter",
	"histogram",
	"pie",
	"donut",
	"heatmap",
	"radar",
	"funnel",
}

func DemoNames() []string {
	return append([]string(nil), demoNames...)
}

func Demo(name string) (Spec, error) {
	switch name {
	case "line":
		return visitorsSpec(Line, "Line Chart", ""), nil
	case "grouped-bar", "bar":
		return visitorsSpec(Bar, "Grouped Bar Chart", Grouped), nil
	case "stacked-bar":
		return visitorsSpec(Bar, "Stacked Bar Chart", Stacked), nil
	case "normalized-bar":
		return visitorsSpec(Bar, "100% Stacked Bar Chart", Normalized), nil
	case "annotated-bar":
		spec := visitorsSpec(Bar, "Illustrative Operating Performance", Grouped)
		spec.Description = "Monthly visitors by platform | indexed operating trend"
		spec.Footer = "Source: Company information; illustrative analysis"
		latest := len(spec.Data) - 1
		spec.Annotations = []Annotation{
			{Text: "Mobile contribution expanded meaningfully into the latest period.", Position: AnnotationTop, DataIndex: &latest, Series: "mobile", Color: defaultColors[1]},
			{Text: "Figures shown for presentation purposes; totals may not sum due to rounding.", Position: AnnotationBottom},
		}
		return spec, nil
	case "horizontal-bar":
		spec := visitorsSpec(Bar, "Horizontal Grouped Bar Chart", Grouped)
		spec.Orientation = Horizontal
		return spec, nil
	case "area":
		return visitorsSpec(Area, "Area Chart", Overlay), nil
	case "stacked-area":
		return visitorsSpec(Area, "Stacked Area Chart", Stacked), nil
	case "combo":
		spec := visitorsSpec(Combo, "Bar and Line Combo Chart", "")
		spec.Series[0].Mark = MarkBar
		spec.Series[1].Mark = MarkLine
		return spec, nil
	case "scatter":
		return Spec{
			Version: SpecVersion,
			Type:    Scatter,
			Title:   "Session Duration and Page Views",
			XAxis:   AxisSpec{DataKey: "minutes", Kind: "number"},
			Series:  []SeriesSpec{{DataKey: "pages", Label: "Pages", Color: defaultColors[0]}},
			Data: []Row{
				{"minutes": 1, "pages": 2}, {"minutes": 2, "pages": 4}, {"minutes": 3, "pages": 3},
				{"minutes": 5, "pages": 7}, {"minutes": 7, "pages": 8}, {"minutes": 9, "pages": 12},
			},
		}, nil
	case "histogram":
		return Spec{
			Version: SpecVersion,
			Type:    Histogram,
			Title:   "Order Value Distribution",
			Bins:    6,
			Series:  []SeriesSpec{{DataKey: "value", Label: "Orders", Color: defaultColors[0]}},
			Data: []Row{
				{"value": 18}, {"value": 22}, {"value": 25}, {"value": 31}, {"value": 32}, {"value": 34},
				{"value": 41}, {"value": 44}, {"value": 47}, {"value": 51}, {"value": 68}, {"value": 83},
			},
		}, nil
	case "pie", "donut":
		chartType := Pie
		if name == "donut" {
			chartType = Donut
		}
		return Spec{
			Version: SpecVersion,
			Type:    chartType,
			Title:   humanize(name) + " Chart — Browsers",
			XAxis:   AxisSpec{DataKey: "browser"},
			Series:  []SeriesSpec{{DataKey: "visitors", Label: "Visitors", Color: defaultColors[0]}},
			Data: []Row{
				{"browser": "Chrome", "visitors": 275}, {"browser": "Safari", "visitors": 200},
				{"browser": "Firefox", "visitors": 187}, {"browser": "Edge", "visitors": 173},
				{"browser": "Other", "visitors": 90},
			},
		}, nil
	case "heatmap":
		return Spec{
			Version: SpecVersion,
			Type:    Heatmap,
			Title:   "Weekly Activity Heatmap",
			XAxis:   AxisSpec{DataKey: "day"},
			Series: []SeriesSpec{
				{DataKey: "morning", Label: "Morning", Color: defaultColors[0]},
				{DataKey: "afternoon", Label: "Afternoon", Color: defaultColors[1]},
				{DataKey: "evening", Label: "Evening", Color: defaultColors[2]},
			},
			Data: []Row{
				{"day": "Mon", "morning": 22, "afternoon": 48, "evening": 35},
				{"day": "Tue", "morning": 31, "afternoon": 56, "evening": 42},
				{"day": "Wed", "morning": 28, "afternoon": 63, "evening": 51},
				{"day": "Thu", "morning": 35, "afternoon": 58, "evening": 47},
				{"day": "Fri", "morning": 41, "afternoon": 72, "evening": 65},
			},
		}, nil
	case "radar":
		return Spec{
			Version: SpecVersion,
			Type:    Radar,
			Title:   "Channel Performance",
			Max:     100,
			XAxis:   AxisSpec{DataKey: "metric"},
			Series: []SeriesSpec{
				{DataKey: "desktop", Label: "Desktop", Color: defaultColors[0]},
				{DataKey: "mobile", Label: "Mobile", Color: defaultColors[1]},
			},
			Data: []Row{
				{"metric": "Reach", "desktop": 82, "mobile": 91},
				{"metric": "Engagement", "desktop": 76, "mobile": 84},
				{"metric": "Conversion", "desktop": 88, "mobile": 69},
				{"metric": "Retention", "desktop": 80, "mobile": 72},
				{"metric": "Revenue", "desktop": 93, "mobile": 74},
			},
		}, nil
	case "funnel":
		return Spec{
			Version: SpecVersion,
			Type:    Funnel,
			Title:   "Acquisition Funnel",
			XAxis:   AxisSpec{DataKey: "stage"},
			Series:  []SeriesSpec{{DataKey: "users", Label: "Users", Color: defaultColors[0]}},
			Labels:  &DisplaySpec{Show: true},
			Data: []Row{
				{"stage": "Visitors", "users": 2400}, {"stage": "Signups", "users": 1320},
				{"stage": "Trials", "users": 760}, {"stage": "Customers", "users": 410},
			},
		}, nil
	default:
		return Spec{}, fmt.Errorf("unknown demo %q", name)
	}
}

func visitorsSpec(chartType Type, title string, layout Layout) Spec {
	spec := Spec{
		Version:     SpecVersion,
		Type:        chartType,
		Title:       title,
		Description: "Visitors by month",
		Footer:      "January – June 2024",
		XAxis:       AxisSpec{DataKey: "month", Kind: "category"},
		Layout:      layout,
		Series: []SeriesSpec{
			{DataKey: "desktop", Label: "Desktop", Color: defaultColors[0]},
			{DataKey: "mobile", Label: "Mobile", Color: defaultColors[1]},
		},
		Data: []Row{
			{"month": "Jan", "desktop": 186, "mobile": 80},
			{"month": "Feb", "desktop": 305, "mobile": 200},
			{"month": "Mar", "desktop": 237, "mobile": 120},
			{"month": "Apr", "desktop": 73, "mobile": 190},
			{"month": "May", "desktop": 209, "mobile": 130},
			{"month": "Jun", "desktop": 214, "mobile": 140},
		},
	}
	if chartType == Line || chartType == Area {
		spec.Curve = Smooth
	}
	return spec
}
