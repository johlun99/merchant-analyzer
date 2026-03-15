package views

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// placeOverlay composites fg onto bg at character position (x, y),
// preserving background content on all sides of the overlay.
func placeOverlay(x, y int, fg, bg string) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	for i, fgLine := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}

		bgLine := bgLines[row]
		fgWidth := ansi.StringWidth(fgLine)

		// Left side: keep background up to column x.
		left := ansi.Truncate(bgLine, x, "")
		// Pad with spaces if the background line is shorter than x.
		if leftWidth := ansi.StringWidth(left); leftWidth < x {
			left += strings.Repeat(" ", x-leftWidth)
		}

		// Right side: keep background after the overlay ends.
		right := ansi.TruncateLeft(bgLine, x+fgWidth, "")

		bgLines[row] = left + fgLine + right
	}

	return strings.Join(bgLines, "\n")
}
