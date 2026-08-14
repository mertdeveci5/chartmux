package chartmux

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	terminalLayerGrid       = 10
	terminalLayerFill       = 20
	terminalLayerMark       = 40
	terminalLayerLine       = 50
	terminalLayerCrosshair  = 60
	terminalLayerAxis       = 70
	terminalLayerPoint      = 80
	terminalLayerAnnotation = 90
	terminalLayerLabel      = 100
)

const (
	terminalGridColor = "#52525B"
	terminalAxisColor = "#71717A"
	terminalTextColor = "#A1A1AA"
)

type terminalPaintStyle struct {
	color    string
	priority int
	bold     bool
	faint    bool
	reverse  bool
}

type terminalFrameCell struct {
	char         rune
	style        terminalPaintStyle
	continuation bool
}

type terminalFrame struct {
	width  int
	height int
	cells  [][]terminalFrameCell
}

type terminalPoint struct {
	x int
	y int
}

type terminalRect struct {
	left   int
	top    int
	right  int
	bottom int
}

func (rect terminalRect) width() int {
	return max(0, rect.right-rect.left+1)
}

func (rect terminalRect) height() int {
	return max(0, rect.bottom-rect.top+1)
}

func newTerminalFrame(width, height int) *terminalFrame {
	frame := &terminalFrame{width: max(1, width), height: max(1, height)}
	frame.cells = make([][]terminalFrameCell, frame.height)
	for row := range frame.cells {
		frame.cells[row] = make([]terminalFrameCell, frame.width)
		for column := range frame.cells[row] {
			frame.cells[row][column].char = ' '
		}
	}
	return frame
}

func (frame *terminalFrame) paint(x, y int, char rune, style terminalPaintStyle) bool {
	if char == 0 || x < 0 || x >= frame.width || y < 0 || y >= frame.height {
		return false
	}
	current := frame.cells[y][x]
	if current.style.priority > style.priority {
		return false
	}
	if current.continuation {
		if current.style.priority >= style.priority {
			return false
		}
		// A higher layer may replace the second cell of a wide rune, but the
		// original leading cell must be cleared or the terminal would still
		// render the old rune across both columns.
		if x > 0 {
			frame.cells[y][x-1] = terminalFrameCell{char: ' ', style: style}
		}
	}
	currentWidth := ansi.StringWidth(string(current.char))
	for offset := 1; offset < currentWidth && x+offset < frame.width; offset++ {
		if frame.cells[y][x+offset].continuation {
			frame.cells[y][x+offset] = terminalFrameCell{char: ' ', style: style}
		}
	}
	frame.cells[y][x] = terminalFrameCell{char: char, style: style}
	return true
}

func (frame *terminalFrame) paintText(x, y int, value string, style terminalPaintStyle) {
	if y < 0 || y >= frame.height || x >= frame.width {
		return
	}
	column := x
	for _, char := range terminalSafeText(value) {
		charWidth := ansi.StringWidth(string(char))
		if charWidth <= 0 {
			continue
		}
		if column < 0 {
			column += charWidth
			continue
		}
		if column+charWidth > frame.width {
			break
		}
		blocked := false
		for offset := 1; offset < charWidth; offset++ {
			if frame.cells[y][column+offset].style.priority > style.priority {
				blocked = true
				break
			}
		}
		if blocked || !frame.paint(column, y, char, style) {
			column += charWidth
			continue
		}
		for offset := 1; offset < charWidth; offset++ {
			if frame.cells[y][column+offset].style.priority <= style.priority {
				frame.cells[y][column+offset] = terminalFrameCell{
					char:         ' ',
					style:        style,
					continuation: true,
				}
			}
		}
		column += charWidth
	}
}

func (frame *terminalFrame) paintAlignedText(x, y, width int, value string, align int, style terminalPaintStyle) {
	if width <= 0 {
		return
	}
	value = ansi.Truncate(terminalSafeText(value), width, "…")
	valueWidth := ansi.StringWidth(value)
	offset := 0
	switch {
	case align < 0:
		offset = 0
	case align > 0:
		offset = max(0, width-valueWidth)
	default:
		offset = max(0, (width-valueWidth)/2)
	}
	frame.paintText(x+offset, y, value, style)
}

func (frame *terminalFrame) paintHorizontal(x0, x1, y int, char rune, style terminalPaintStyle) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	for x := x0; x <= x1; x++ {
		frame.paint(x, y, char, style)
	}
}

func (frame *terminalFrame) paintVertical(x, y0, y1 int, char rune, style terminalPaintStyle) {
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		frame.paint(x, y, char, style)
	}
}

func (frame *terminalFrame) paintLine(start, end terminalPoint, char rune, style terminalPaintStyle) {
	x0, y0 := start.x, start.y
	x1, y1 := end.x, end.y
	dx := int(math.Abs(float64(x1 - x0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -int(math.Abs(float64(y1 - y0)))
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		frame.paint(x0, y0, char, style)
		if x0 == x1 && y0 == y1 {
			break
		}
		doubled := 2 * err
		if doubled >= dy {
			err += dy
			x0 += sx
		}
		if doubled <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (frame *terminalFrame) render() string {
	rows := make([]string, frame.height)
	for y, cells := range frame.cells {
		last := -1
		for x, cell := range cells {
			if cell.char != ' ' || cell.continuation {
				last = x
			}
		}
		if last < 0 {
			continue
		}
		var row strings.Builder
		for x := 0; x <= last; {
			if cells[x].continuation {
				x++
				continue
			}
			style := cells[x].style
			var run strings.Builder
			for x <= last {
				cell := cells[x]
				if cell.continuation {
					x++
					continue
				}
				if cell.style != style {
					break
				}
				run.WriteRune(cell.char)
				charWidth := max(1, ansi.StringWidth(string(cell.char)))
				x += charWidth
			}
			row.WriteString(renderTerminalRun(run.String(), style))
		}
		rows[y] = row.String()
	}
	return strings.Join(rows, "\n")
}

func renderTerminalRun(value string, style terminalPaintStyle) string {
	if value == "" || (style.color == "" && !style.bold && !style.faint && !style.reverse) {
		return value
	}
	renderer := lipgloss.NewStyle().
		Bold(style.bold).
		Faint(style.faint).
		Reverse(style.reverse)
	if style.color != "" {
		renderer = renderer.Foreground(terminalColor(style.color))
	}
	return renderer.Render(value)
}

type terminalTick struct {
	value float64
	label string
	y     int
}

type terminalCartesianScene struct {
	frame       *terminalFrame
	plot        terminalRect
	axisX       int
	baselineY   int
	labelY      int
	minimum     float64
	maximum     float64
	ticks       []terminalTick
	showAxes    bool
	percentAxis bool
}

func newTerminalCartesianScene(width, height int, minimum, maximum float64, showAxes, percent bool) (*terminalCartesianScene, error) {
	if minimum == maximum {
		minimum--
		maximum++
	}
	frame := newTerminalFrame(width, height)
	scene := &terminalCartesianScene{
		frame:       frame,
		plot:        terminalRect{left: 0, top: 0, right: width - 1, bottom: height - 1},
		minimum:     minimum,
		maximum:     maximum,
		showAxes:    showAxes,
		percentAxis: percent,
	}
	if !showAxes {
		return scene, nil
	}
	if height < 5 {
		return nil, fmt.Errorf("terminal is too short for cartesian axes; increase the chart height")
	}
	tickTarget := min(5, max(3, (height-2)/3+1))
	niceMin, niceMax, tickValues := terminalNiceScale(minimum, maximum, tickTarget)
	scene.minimum = niceMin
	scene.maximum = niceMax
	axisWidth := 1
	for _, value := range tickValues {
		axisWidth = max(axisWidth, ansi.StringWidth(scene.formatTick(value)))
	}
	scene.axisX = axisWidth + 1
	scene.baselineY = height - 2
	scene.labelY = height - 1
	scene.plot = terminalRect{left: scene.axisX + 1, top: 0, right: width - 1, bottom: scene.baselineY}
	if scene.plot.width() < 8 {
		return nil, fmt.Errorf("terminal is too narrow for cartesian axes; increase the chart width")
	}
	if scene.plot.height() < 4 {
		return nil, fmt.Errorf("terminal is too short for the chart plot; increase the chart height")
	}
	scene.paintAxes(tickValues, axisWidth)
	return scene, nil
}

func (scene *terminalCartesianScene) formatTick(value float64) string {
	if scene.percentAxis {
		return fmt.Sprintf("%.0f%%", value)
	}
	return formatValue(value)
}

func (scene *terminalCartesianScene) paintAxes(values []float64, axisWidth int) {
	gridStyle := terminalPaintStyle{color: terminalGridColor, priority: terminalLayerGrid, faint: true}
	axisStyle := terminalPaintStyle{color: terminalAxisColor, priority: terminalLayerAxis, faint: true}
	labelStyle := terminalPaintStyle{color: terminalTextColor, priority: terminalLayerLabel, faint: true}
	scene.frame.paintVertical(scene.axisX, scene.plot.top, scene.baselineY, '│', axisStyle)
	for _, value := range values {
		y := scene.y(value)
		label := scene.formatTick(value)
		scene.frame.paintAlignedText(0, y, axisWidth, label, 1, labelStyle)
		if y == scene.baselineY {
			scene.frame.paint(scene.axisX, y, '└', axisStyle)
			scene.frame.paintHorizontal(scene.plot.left, scene.plot.right, y, '─', axisStyle)
			continue
		}
		scene.frame.paint(scene.axisX, y, '├', axisStyle)
		for x := scene.plot.left; x <= scene.plot.right; x += 3 {
			scene.frame.paint(x, y, '┄', gridStyle)
		}
		scene.ticks = append(scene.ticks, terminalTick{value: value, label: label, y: y})
	}
	if scene.frame.cells[scene.baselineY][scene.axisX].char != '└' {
		scene.frame.paint(scene.axisX, scene.baselineY, '└', axisStyle)
		scene.frame.paintHorizontal(scene.plot.left, scene.plot.right, scene.baselineY, '─', axisStyle)
	}
}

func (scene *terminalCartesianScene) y(value float64) int {
	ratio := (value - scene.minimum) / (scene.maximum - scene.minimum)
	ratio = math.Max(0, math.Min(1, ratio))
	return scene.plot.bottom - int(math.Round(ratio*float64(max(1, scene.plot.height()-1))))
}

func (scene *terminalCartesianScene) numericX(value, minimum, maximum float64) int {
	if minimum == maximum {
		return scene.plot.left + scene.plot.width()/2
	}
	ratio := (value - minimum) / (maximum - minimum)
	ratio = math.Max(0, math.Min(1, ratio))
	return scene.plot.left + int(math.Round(ratio*float64(max(1, scene.plot.width()-1))))
}

func (scene *terminalCartesianScene) pointX(index, count int) int {
	if count <= 1 {
		return scene.plot.left + scene.plot.width()/2
	}
	return scene.plot.left + int(math.Round(float64(index)/float64(count-1)*float64(scene.plot.width()-1)))
}

func (scene *terminalCartesianScene) band(index, count int) terminalRect {
	if count <= 0 {
		return terminalRect{left: scene.plot.left, right: scene.plot.right}
	}
	left := scene.plot.left + int(math.Floor(float64(index)*float64(scene.plot.width())/float64(count)))
	right := scene.plot.left + int(math.Floor(float64(index+1)*float64(scene.plot.width())/float64(count))) - 1
	if index == count-1 {
		right = scene.plot.right
	}
	return terminalRect{left: left, top: scene.plot.top, right: max(left, right), bottom: scene.plot.bottom}
}

func (scene *terminalCartesianScene) paintCategoryLabels(labels []string, banded bool) {
	if !scene.showAxes || len(labels) == 0 {
		return
	}
	axisStyle := terminalPaintStyle{color: terminalAxisColor, priority: terminalLayerAxis, faint: true}
	labelStyle := terminalPaintStyle{color: terminalTextColor, priority: terminalLayerLabel, faint: true}
	lastEnd := -1
	for index, raw := range labels {
		x := scene.pointX(index, len(labels))
		available := max(1, scene.plot.width()/len(labels))
		if banded {
			band := scene.band(index, len(labels))
			x = band.left + band.width()/2
			available = max(1, band.width()-1)
		}
		label := ansi.Truncate(terminalSafeText(raw), available, "…")
		labelWidth := ansi.StringWidth(label)
		start := max(0, min(scene.frame.width-labelWidth, x-labelWidth/2))
		if start <= lastEnd {
			continue
		}
		scene.frame.paint(x, scene.baselineY, '┬', axisStyle)
		scene.frame.paintText(start, scene.labelY, label, labelStyle)
		lastEnd = start + labelWidth
	}
}

func (scene *terminalCartesianScene) paintNumericLabels(minimum, maximum float64) {
	if !scene.showAxes {
		return
	}
	axisStyle := terminalPaintStyle{color: terminalAxisColor, priority: terminalLayerAxis, faint: true}
	labelStyle := terminalPaintStyle{color: terminalTextColor, priority: terminalLayerLabel, faint: true}
	const count = 4
	lastEnd := -1
	for index := 0; index < count; index++ {
		ratio := float64(index) / float64(count-1)
		value := minimum + (maximum-minimum)*ratio
		x := scene.numericX(value, minimum, maximum)
		label := formatValue(value)
		labelWidth := ansi.StringWidth(label)
		start := max(0, min(scene.frame.width-labelWidth, x-labelWidth/2))
		if start <= lastEnd {
			continue
		}
		scene.frame.paint(x, scene.baselineY, '┬', axisStyle)
		scene.frame.paintText(start, scene.labelY, label, labelStyle)
		lastEnd = start + labelWidth
	}
}

func terminalNiceScale(minimum, maximum float64, target int) (float64, float64, []float64) {
	if minimum == maximum {
		minimum--
		maximum++
	}
	target = max(2, target)
	step := terminalNiceNumber((maximum-minimum)/float64(target-1), true)
	if step <= 0 || math.IsInf(step, 0) || math.IsNaN(step) {
		step = 1
	}
	niceMin := math.Floor(minimum/step) * step
	niceMax := math.Ceil(maximum/step) * step
	values := make([]float64, 0, target+2)
	for value, count := niceMin, 0; value <= niceMax+step*0.5 && count < 12; value, count = value+step, count+1 {
		if math.Abs(value) < step*1e-9 {
			value = 0
		}
		values = append(values, value)
	}
	return niceMin, niceMax, values
}

func terminalNiceNumber(value float64, round bool) float64 {
	if value <= 0 {
		return 1
	}
	exponent := math.Floor(math.Log10(value))
	fraction := value / math.Pow(10, exponent)
	niceFraction := 1.0
	if round {
		switch {
		case fraction < 1.5:
			niceFraction = 1
		case fraction < 2.25:
			niceFraction = 2
		case fraction < 3.5:
			niceFraction = 2.5
		case fraction < 9:
			niceFraction = 5
		default:
			niceFraction = 10
		}
	} else {
		switch {
		case fraction <= 1:
			niceFraction = 1
		case fraction <= 2:
			niceFraction = 2
		case fraction <= 2.5:
			niceFraction = 2.5
		case fraction <= 5:
			niceFraction = 5
		default:
			niceFraction = 10
		}
	}
	return niceFraction * math.Pow(10, exponent)
}

func terminalBarGlyph(series, x, y int) rune {
	_ = x
	_ = y
	return terminalBarPattern(series)
}

func terminalAreaGlyph(series, x, y int) rune {
	_ = x
	_ = y
	return terminalAreaPattern(series)
}
