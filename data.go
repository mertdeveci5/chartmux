package chartmux

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const MaxInputBytes = 8 << 20

type Dataset struct {
	Columns []string
	Rows    []Row
}

func ParseSpec(reader io.Reader) (Spec, error) {
	content, err := readInput(reader)
	if err != nil {
		return Spec{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var spec Spec
	if err := decodeOne(decoder, &spec); err != nil {
		return Spec{}, fmt.Errorf("parse chart spec: %w", err)
	}
	if spec.Version == 0 {
		return Spec{}, fmt.Errorf("chart spec is missing version")
	}
	return spec, nil
}

func ParseDataset(reader io.Reader) (Dataset, error) {
	content, err := readInput(reader)
	if err != nil {
		return Dataset{}, err
	}
	if content[0] == '[' {
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.UseNumber()
		var rows []Row
		if err := decodeOne(decoder, &rows); err != nil {
			return Dataset{}, fmt.Errorf("parse JSON dataset: %w", err)
		}
		return datasetFromRows(rows)
	}
	return parseDelimited(content)
}

func SpecFromDataset(dataset Dataset, chartType Type, xKey string, seriesKeys []string) (Spec, error) {
	if !validType(chartType) {
		return Spec{}, fmt.Errorf("unsupported chart type %q", chartType)
	}
	if len(dataset.Rows) == 0 || len(dataset.Columns) == 0 {
		return Spec{}, fmt.Errorf("dataset is empty")
	}
	if chartType != Histogram {
		if xKey == "" {
			xKey = inferXKey(dataset, chartType)
		}
		if !hasColumn(dataset.Columns, xKey) {
			return Spec{}, fmt.Errorf("unknown x column %q; choose from %s", xKey, strings.Join(dataset.Columns, ", "))
		}
	}
	if len(seriesKeys) == 0 {
		seriesKeys = inferSeries(dataset, xKey)
	}
	if len(seriesKeys) == 0 {
		return Spec{}, fmt.Errorf("could not infer a numeric series; pass --series")
	}
	series := make([]SeriesSpec, len(seriesKeys))
	seen := make(map[string]bool, len(seriesKeys))
	for index, key := range seriesKeys {
		if !hasColumn(dataset.Columns, key) {
			return Spec{}, fmt.Errorf("unknown series column %q; choose from %s", key, strings.Join(dataset.Columns, ", "))
		}
		if key == xKey {
			return Spec{}, fmt.Errorf("series column %q is already used by the x axis", key)
		}
		if seen[key] {
			return Spec{}, fmt.Errorf("duplicate series column %q", key)
		}
		seen[key] = true
		series[index] = SeriesSpec{DataKey: key, Label: humanize(key), Color: defaultColors[index%len(defaultColors)]}
	}
	return Spec{
		Version: SpecVersion,
		Type:    chartType,
		Title:   humanize(string(chartType)) + " Chart",
		Data:    dataset.Rows,
		XAxis:   AxisSpec{DataKey: xKey},
		Series:  series,
	}, nil
}

func readInput(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, MaxInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	if len(content) > MaxInputBytes {
		return nil, fmt.Errorf("input exceeds %d MB", MaxInputBytes>>20)
	}
	content = bytes.TrimSpace(content)
	if len(content) == 0 {
		return nil, fmt.Errorf("input is empty")
	}
	return content, nil
}

func decodeOne(decoder *json.Decoder, target any) error {
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("unexpected content after JSON value")
	}
	return nil
}

func datasetFromRows(rows []Row) (Dataset, error) {
	if len(rows) == 0 {
		return Dataset{}, fmt.Errorf("JSON dataset has no rows")
	}
	set := make(map[string]bool)
	for rowIndex, row := range rows {
		if len(row) == 0 {
			return Dataset{}, fmt.Errorf("JSON dataset row %d is empty", rowIndex+1)
		}
		for key, value := range row {
			if _, _, err := scalarValue(value); err != nil {
				return Dataset{}, fmt.Errorf("JSON dataset row %d field %q: %w", rowIndex+1, key, err)
			}
			set[key] = true
		}
	}
	columns := make([]string, 0, len(set))
	for key := range set {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	return Dataset{Columns: columns, Rows: rows}, nil
}

func parseDelimited(content []byte) (Dataset, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.Comma = detectDelimiter(content)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return Dataset{}, fmt.Errorf("parse delimited input: %w", err)
	}
	if len(records) < 2 {
		return Dataset{}, fmt.Errorf("input needs a header and at least one data row")
	}
	columns := records[0]
	seen := make(map[string]bool, len(columns))
	for index, column := range columns {
		column = strings.TrimSpace(column)
		if column == "" {
			return Dataset{}, fmt.Errorf("column %d has no name", index+1)
		}
		if seen[column] {
			return Dataset{}, fmt.Errorf("duplicate column %q", column)
		}
		seen[column] = true
		columns[index] = column
	}
	rows := make([]Row, 0, len(records)-1)
	for rowIndex, record := range records[1:] {
		if len(record) != len(columns) {
			return Dataset{}, fmt.Errorf("row %d has %d fields; expected %d", rowIndex+2, len(record), len(columns))
		}
		row := make(Row, len(columns))
		for columnIndex, column := range columns {
			value := strings.TrimSpace(record[columnIndex])
			if value == "" {
				row[column] = nil
			} else {
				row[column] = value
			}
		}
		rows = append(rows, row)
	}
	return Dataset{Columns: columns, Rows: rows}, nil
}

func detectDelimiter(content []byte) rune {
	firstLine := string(content)
	if index := strings.IndexByte(firstLine, '\n'); index >= 0 {
		firstLine = firstLine[:index]
	}
	candidates := []rune{',', '\t', ';'}
	best := candidates[0]
	count := 0
	for _, candidate := range candidates {
		if candidateCount := strings.Count(firstLine, string(candidate)); candidateCount > count {
			best = candidate
			count = candidateCount
		}
	}
	return best
}

func inferXKey(dataset Dataset, chartType Type) string {
	for _, column := range dataset.Columns {
		if chartType == Scatter {
			if columnIsNumeric(dataset.Rows, column) {
				return column
			}
			continue
		}
		if !columnIsNumeric(dataset.Rows, column) {
			return column
		}
	}
	return dataset.Columns[0]
}

func inferSeries(dataset Dataset, xKey string) []string {
	var keys []string
	for _, column := range dataset.Columns {
		if column != xKey && columnIsNumeric(dataset.Rows, column) {
			keys = append(keys, column)
		}
	}
	return keys
}

func columnIsNumeric(rows []Row, column string) bool {
	found := false
	for _, row := range rows {
		value, ok := row[column]
		if !ok {
			return false
		}
		_, missing, err := numericValue(value)
		if err != nil {
			return false
		}
		found = found || !missing
	}
	return found
}

func scalarValue(value any) (string, bool, error) {
	if value == nil {
		return "", true, nil
	}
	switch value := value.(type) {
	case string:
		return value, false, nil
	case json.Number:
		return value.String(), false, nil
	case float64, bool:
		return fmt.Sprint(value), false, nil
	default:
		return "", false, fmt.Errorf("value must be a string, number, boolean, or null")
	}
}

func hasColumn(columns []string, target string) bool {
	for _, column := range columns {
		if column == target {
			return true
		}
	}
	return false
}
