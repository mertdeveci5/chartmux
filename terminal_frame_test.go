package chartmux

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTerminalFrameKeepsHigherLayersVisible(t *testing.T) {
	frame := newTerminalFrame(8, 1)
	frame.paint(2, 0, '✦', terminalPaintStyle{priority: terminalLayerAnnotation})
	frame.paint(2, 0, '░', terminalPaintStyle{priority: terminalLayerFill})
	frame.paint(2, 0, '·', terminalPaintStyle{priority: terminalLayerGrid})

	if output := ansi.Strip(frame.render()); !strings.Contains(output, "✦") || strings.ContainsAny(output, "░·") {
		t.Fatalf("lower layers overwrote an annotation: %q", output)
	}
}

func TestTerminalFrameProtectsWideTextFromLowerLayers(t *testing.T) {
	frame := newTerminalFrame(6, 1)
	frame.paintText(0, 0, "財A", terminalPaintStyle{priority: terminalLayerLabel})
	frame.paint(1, 0, '✦', terminalPaintStyle{priority: terminalLayerAnnotation})

	output := ansi.Strip(frame.render())
	if output != "財A" || ansi.StringWidth(output) != 3 {
		t.Fatalf("wide label was corrupted by a lower layer: %q (%d cells)", output, ansi.StringWidth(output))
	}
}

func TestTerminalFrameReplacesWideTextWithoutLeavingContinuationCells(t *testing.T) {
	frame := newTerminalFrame(6, 1)
	frame.paintText(0, 0, "財A", terminalPaintStyle{priority: terminalLayerLabel})
	frame.paint(1, 0, '!', terminalPaintStyle{priority: terminalLayerLabel + 1})

	output := ansi.Strip(frame.render())
	if output != " !A" || ansi.StringWidth(output) != 3 {
		t.Fatalf("wide label replacement left corrupt cells: %q (%d cells)", output, ansi.StringWidth(output))
	}
}

func TestTerminalFrameDoesNotLeaveContinuationForRejectedWideText(t *testing.T) {
	frame := newTerminalFrame(4, 1)
	frame.paint(0, 0, '✦', terminalPaintStyle{priority: terminalLayerAnnotation})
	frame.paintText(0, 0, "財", terminalPaintStyle{priority: terminalLayerFill})

	output := ansi.Strip(frame.render())
	if output != "✦" || ansi.StringWidth(output) != 1 {
		t.Fatalf("rejected wide text left a continuation cell: %q (%d cells)", output, ansi.StringWidth(output))
	}
}
