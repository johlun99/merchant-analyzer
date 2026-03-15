package views

import (
	"strings"
	"testing"
)

func TestPlaceOverlay_PreservesAllSides(t *testing.T) {
	bg := makeGrid('B', 20, 5)
	fg := "FFF"
	got := placeOverlay(5, 1, fg, bg)
	lines := strings.Split(got, "\n")

	// Row 0 unchanged
	if lines[0] != strings.Repeat("B", 20) {
		t.Errorf("row 0: want all B's, got %q", lines[0])
	}
	// Row 1: 5 B's + FFF + 12 B's
	want := strings.Repeat("B", 5) + "FFF" + strings.Repeat("B", 12)
	if lines[1] != want {
		t.Errorf("row 1: want %q, got %q", want, lines[1])
	}
	// Rows 2-4 unchanged
	for i := 2; i < 5; i++ {
		if lines[i] != strings.Repeat("B", 20) {
			t.Errorf("row %d: want all B's, got %q", i, lines[i])
		}
	}
}

func TestPlaceOverlay_OriginPlacement(t *testing.T) {
	bg := makeGrid('B', 10, 3)
	fg := "FFF"
	got := placeOverlay(0, 0, fg, bg)
	lines := strings.Split(got, "\n")

	want := "FFF" + strings.Repeat("B", 7)
	if lines[0] != want {
		t.Errorf("row 0: want %q, got %q", want, lines[0])
	}
	// Other rows unchanged
	for i := 1; i < 3; i++ {
		if lines[i] != strings.Repeat("B", 10) {
			t.Errorf("row %d: want all B's, got %q", i, lines[i])
		}
	}
}

func TestPlaceOverlay_ExceedsHeight(t *testing.T) {
	bg := makeGrid('B', 10, 3)
	fg := "AAA\nBBB\nCCC\nDDD" // 4 lines, bg only has 3
	got := placeOverlay(0, 1, fg, bg)
	lines := strings.Split(got, "\n")

	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(lines))
	}
	// Row 0 unchanged
	if lines[0] != strings.Repeat("B", 10) {
		t.Errorf("row 0: want all B's, got %q", lines[0])
	}
	// Rows 1-2 get overlay lines
	if want := "AAA" + strings.Repeat("B", 7); lines[1] != want {
		t.Errorf("row 1: want %q, got %q", want, lines[1])
	}
	if want := "BBB" + strings.Repeat("B", 7); lines[2] != want {
		t.Errorf("row 2: want %q, got %q", want, lines[2])
	}
}

func TestPlaceOverlay_EmptyForeground(t *testing.T) {
	bg := makeGrid('B', 10, 3)
	got := placeOverlay(0, 0, "", bg)
	if got != bg {
		t.Errorf("want unchanged background, got %q", got)
	}
}

// makeGrid builds a width×height block of the given character.
func makeGrid(ch rune, width, height int) string {
	row := strings.Repeat(string(ch), width)
	rows := make([]string, height)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}
