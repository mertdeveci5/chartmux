package chartmux

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxAnnotations       = 12
	maxAnnotationLength  = 500
	maxTitleLength       = 240
	maxBodyTextLength    = 2000
	maxDataKeyLength     = 120
	maxSeriesLabelLength = 120
	maxAxisLabelLength   = 240
)

type AnnotationPosition string

const (
	AnnotationTop    AnnotationPosition = "top"
	AnnotationBottom AnnotationPosition = "bottom"
)

// Annotation adds a collision-safe narrative note above or below the plot.
// DataIndex and Series add deterministic point or series context without
// placing free-form text over the data marks themselves.
type Annotation struct {
	Text      string             `json:"text"`
	Position  AnnotationPosition `json:"position,omitempty"`
	DataIndex *int               `json:"dataIndex,omitempty"`
	Series    string             `json:"series,omitempty"`
	Color     string             `json:"color,omitempty"`
}

func validateTextFields(spec Spec) error {
	fields := []struct {
		name  string
		value string
		limit int
	}{
		{name: "title", value: spec.Title, limit: maxTitleLength},
		{name: "description", value: spec.Description, limit: maxBodyTextLength},
		{name: "footer", value: spec.Footer, limit: maxBodyTextLength},
	}
	for _, field := range fields {
		if utf8.RuneCountInString(field.value) > field.limit {
			return fmt.Errorf("%s supports at most %d characters", field.name, field.limit)
		}
		if hasUnsafeControl(field.value) {
			return fmt.Errorf("%s contains an unsupported control character", field.name)
		}
	}
	return nil
}

func validateAnnotations(spec Spec, seriesKeys map[string]bool) error {
	if len(spec.Annotations) > maxAnnotations {
		return fmt.Errorf("chart spec supports at most %d annotations", maxAnnotations)
	}
	for index, annotation := range spec.Annotations {
		name := fmt.Sprintf("annotation %d", index+1)
		if strings.TrimSpace(annotation.Text) == "" {
			return fmt.Errorf("%s text must not be empty", name)
		}
		if utf8.RuneCountInString(annotation.Text) > maxAnnotationLength {
			return fmt.Errorf("%s supports at most %d characters", name, maxAnnotationLength)
		}
		if hasUnsafeControl(annotation.Text) {
			return fmt.Errorf("%s contains an unsupported control character", name)
		}
		if annotation.Position != "" && annotation.Position != AnnotationTop && annotation.Position != AnnotationBottom {
			return fmt.Errorf("%s position must be top or bottom", name)
		}
		if annotation.DataIndex != nil {
			if spec.Type == Histogram {
				return fmt.Errorf("%s dataIndex is not valid for histogram bins", name)
			}
			if *annotation.DataIndex < 0 || *annotation.DataIndex >= len(spec.Data) {
				return fmt.Errorf("%s dataIndex must be between 0 and %d", name, len(spec.Data)-1)
			}
		}
		if annotation.Series != "" && !seriesKeys[annotation.Series] {
			return fmt.Errorf("%s references unknown series %q", name, annotation.Series)
		}
		if annotation.Color != "" {
			if _, err := colorHex(annotation.Color); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}

func hasUnsafeControl(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) && char != '\n' && char != '\t' && char != '\r' {
			return true
		}
	}
	return false
}

func validateInlineText(name, value string, limit int) error {
	if utf8.RuneCountInString(value) > limit {
		return fmt.Errorf("%s supports at most %d characters", name, limit)
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return fmt.Errorf("%s contains an unsupported control character", name)
		}
	}
	return nil
}

func (chart *Chart) annotationText(annotation Annotation) string {
	context := make([]string, 0, 2)
	if annotation.DataIndex != nil && *annotation.DataIndex >= 0 && *annotation.DataIndex < len(chart.labels) {
		if label := strings.TrimSpace(chart.labels[*annotation.DataIndex]); label != "" {
			context = append(context, label)
		}
	}
	if annotation.Series != "" {
		for _, series := range chart.series {
			if series.spec.DataKey == annotation.Series {
				context = append(context, series.spec.Label)
				break
			}
		}
	}
	text := strings.TrimSpace(annotation.Text)
	if len(context) == 0 {
		return text
	}
	return strings.Join(context, " · ") + " — " + text
}

func annotationPosition(annotation Annotation) AnnotationPosition {
	if annotation.Position == "" {
		return AnnotationBottom
	}
	return annotation.Position
}
