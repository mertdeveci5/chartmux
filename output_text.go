package chartmux

import (
	"fmt"
	"html"
	"strings"

	"github.com/go-analyze/charts"
)

const (
	graphicalTextMargin = 24
	minimumPlotHeight   = 96
)

type graphicalTextLine struct {
	text   string
	style  charts.FontStyle
	height int
}

type graphicalTextLayoutResult struct {
	top          []graphicalTextLine
	bottom       []graphicalTextLine
	topHeight    int
	bottomHeight int
}

func (layout graphicalTextLayoutResult) plotBox(width, height int) charts.Box {
	return charts.NewBox(0, layout.topHeight, width, height-layout.bottomHeight)
}

func (chart *Chart) graphicalTextLayout(painter *charts.Painter, theme charts.ColorPalette, width, height int) (graphicalTextLayoutResult, error) {
	availableWidth := max(1, width-graphicalTextMargin*2)
	titleStyle := charts.NewFontStyleWithSize(18).
		WithColor(theme.GetTitleTextColor()).
		WithFont(charts.GetFont(charts.FontFamilyNotoSansBold))
	descriptionStyle := charts.NewFontStyleWithSize(11).WithColor(theme.GetLabelTextColor())
	annotationStyle := charts.NewFontStyleWithSize(10.5).WithColor(theme.GetLabelTextColor())
	footerStyle := charts.NewFontStyleWithSize(9.5).WithColor(theme.GetXAxisTextColor())

	var layout graphicalTextLayoutResult
	layout.top = appendGraphicalBlock(layout.top, painter, chart.spec.Title, availableWidth, titleStyle)
	layout.top = appendGraphicalBlock(layout.top, painter, chart.spec.Description, availableWidth, descriptionStyle)
	for _, annotation := range chart.spec.Annotations {
		if annotationPosition(annotation) != AnnotationTop {
			continue
		}
		style, err := graphicalAnnotationStyle(annotation, annotationStyle)
		if err != nil {
			return graphicalTextLayoutResult{}, err
		}
		layout.top = appendGraphicalBlock(layout.top, painter, "◆ "+chart.annotationText(annotation), availableWidth, style)
	}
	for _, annotation := range chart.spec.Annotations {
		if annotationPosition(annotation) != AnnotationBottom {
			continue
		}
		style, err := graphicalAnnotationStyle(annotation, annotationStyle)
		if err != nil {
			return graphicalTextLayoutResult{}, err
		}
		layout.bottom = appendGraphicalBlock(layout.bottom, painter, "◆ "+chart.annotationText(annotation), availableWidth, style)
	}
	layout.bottom = appendGraphicalBlock(layout.bottom, painter, chart.spec.Footer, availableWidth, footerStyle)

	if len(layout.top) > 0 {
		layout.topHeight = graphicalLinesHeight(layout.top) + 24
	}
	if len(layout.bottom) > 0 {
		layout.bottomHeight = graphicalLinesHeight(layout.bottom) + 24
	}
	if height-layout.topHeight-layout.bottomHeight < minimumPlotHeight {
		return graphicalTextLayoutResult{}, fmt.Errorf("image is too short for chart text and annotations; increase the image height")
	}
	return layout, nil
}

func graphicalAnnotationStyle(annotation Annotation, fallback charts.FontStyle) (charts.FontStyle, error) {
	if annotation.Color == "" {
		return fallback, nil
	}
	color, err := colorHex(annotation.Color)
	if err != nil {
		return charts.FontStyle{}, err
	}
	return fallback.WithColor(color), nil
}

func appendGraphicalBlock(lines []graphicalTextLine, painter *charts.Painter, text string, width int, style charts.FontStyle) []graphicalTextLine {
	if strings.TrimSpace(text) == "" {
		return lines
	}
	if len(lines) > 0 {
		lines = append(lines, graphicalTextLine{height: 5})
	}
	lineHeight := max(1, painter.MeasureText("Ag", 0, style).Height()+2)
	for _, line := range wrapPainterText(painter, text, width, style) {
		lines = append(lines, graphicalTextLine{text: line, style: style, height: lineHeight})
	}
	return lines
}

func graphicalLinesHeight(lines []graphicalTextLine) int {
	height := 0
	for _, line := range lines {
		height += line.height
	}
	return height
}

func wrapPainterText(painter *charts.Painter, text string, width int, style charts.FontStyle) []string {
	var result []string
	for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		var line string
		for _, word := range strings.Fields(paragraph) {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if painter.MeasureText(candidate, 0, style).Width() <= width {
				line = candidate
				continue
			}
			if line != "" {
				result = append(result, line)
				line = ""
			}
			chunks := breakPainterWord(painter, word, width, style)
			if len(chunks) > 1 {
				result = append(result, chunks[:len(chunks)-1]...)
			}
			line = chunks[len(chunks)-1]
		}
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func breakPainterWord(painter *charts.Painter, word string, width int, style charts.FontStyle) []string {
	var chunks []string
	var current string
	for _, char := range word {
		candidate := current + string(char)
		if current != "" && painter.MeasureText(candidate, 0, style).Width() > width {
			chunks = append(chunks, current)
			current = string(char)
			continue
		}
		current = candidate
	}
	if current != "" || len(chunks) == 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func (chart *Chart) drawGraphicalText(painter *charts.Painter, layout graphicalTextLayoutResult, theme charts.ColorPalette, svg bool, frameWidth, frameHeight, originTop int) {
	drawGraphicalLines(painter, layout.top, graphicalTextMargin, 12-originTop, svg)
	if len(layout.bottom) == 0 {
		return
	}
	start := frameHeight - layout.bottomHeight
	painter.LineStroke(
		[]charts.Point{{X: graphicalTextMargin, Y: start + 6 - originTop}, {X: frameWidth - graphicalTextMargin, Y: start + 6 - originTop}},
		theme.GetAxisSplitLineColor(),
		0.75,
	)
	drawGraphicalLines(painter, layout.bottom, graphicalTextMargin, start+12-originTop, svg)
}

func drawGraphicalLines(painter *charts.Painter, lines []graphicalTextLine, x, y int, svg bool) {
	for _, line := range lines {
		y += line.height
		if line.text == "" {
			continue
		}
		text := line.text
		if svg {
			text = html.EscapeString(text)
		}
		painter.Text(text, x, y-2, 0, line.style)
	}
}
