package bitmapmanager

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func point(left, top int) geo.Point {
	return geo.Point{Left: left, Top: top}
}

func singleAnchorChar(char rune) *shape.AnchorChar {
	return &shape.AnchorChar{ID: "id", DisplayName: "name", Left: char, Right: char, Top: char, Bottom: char}
}

func testLineExtra() shape.LineExtra {
	return shape.LineExtra{
		IsStrokeEnabled:         true,
		UserSelectedStrokeStyle: shape.PredefinedStrokeStyles()[0],
		IsStartAnchorEnabled:    true,
		UserSelectedStartAnchor: singleAnchorChar('0'),
		IsEndAnchorEnabled:      true,
		UserSelectedEndAnchor:   singleAnchorChar('1'),
		DashPattern:             shape.SolidDashPattern,
	}
}

func lines(rows ...string) string {
	return strings.Join(rows, "\n")
}

func TestLineBitmap(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		points   []geo.Point
		expected string
	}{
		{"horizontal", []geo.Point{point(2, 0), point(6, 0)}, "0───1"},
		{"horizontal reversed", []geo.Point{point(6, 0), point(2, 0)}, "1───0"},
		{"horizontal monotonic", []geo.Point{point(0, 0), point(2, 0), point(4, 0)}, "0───1"},
		{"horizontal monotonic reversed", []geo.Point{point(4, 0), point(2, 0), point(0, 0)}, "1───0"},
		{"horizontal non monotonic", []geo.Point{point(0, 0), point(4, 0), point(2, 0)}, "0─1──"},
		{"horizontal non monotonic reversed", []geo.Point{point(4, 0), point(0, 0), point(2, 0)}, "──1─0"},
		{"vertical", []geo.Point{point(0, 2), point(0, 6)}, lines("0", "│", "│", "│", "1")},
		{"vertical reversed", []geo.Point{point(0, 6), point(0, 2)}, lines("1", "│", "│", "│", "0")},
		{"vertical monotonic", []geo.Point{point(0, 0), point(0, 2), point(0, 4)}, lines("0", "│", "│", "│", "1")},
		{"vertical monotonic reversed", []geo.Point{point(0, 4), point(0, 2), point(0, 0)}, lines("1", "│", "│", "│", "0")},
		{"vertical non monotonic", []geo.Point{point(0, 0), point(0, 4), point(0, 2)}, lines("0", "│", "1", "│", "│")},
		{"vertical non monotonic reversed", []geo.Point{point(0, 4), point(0, 0), point(0, 2)}, lines("│", "│", "1", "│", "0")},
		{"upper left", []geo.Point{point(0, 4), point(4, 4), point(4, 0)}, lines("    1", "    │", "    │", "    │", "0───┘")},
		{"upper left reversed", []geo.Point{point(4, 0), point(4, 4), point(0, 4)}, lines("    0", "    │", "    │", "    │", "1───┘")},
		{"lower left", []geo.Point{point(0, 0), point(4, 0), point(4, 4)}, lines("0───┐", "    │", "    │", "    │", "    1")},
		{"lower left reversed", []geo.Point{point(4, 4), point(4, 0), point(0, 0)}, lines("1───┐", "    │", "    │", "    │", "    0")},
		{"upper right", []geo.Point{point(4, 4), point(0, 4), point(0, 0)}, lines("1    ", "│    ", "│    ", "│    ", "└───0")},
		{"upper right reversed", []geo.Point{point(0, 0), point(0, 4), point(4, 4)}, lines("0    ", "│    ", "│    ", "│    ", "└───1")},
		{"lower right", []geo.Point{point(4, 0), point(0, 0), point(0, 4)}, lines("┌───0", "│    ", "│    ", "│    ", "1    ")},
		{"lower right reversed", []geo.Point{point(0, 4), point(0, 0), point(4, 0)}, lines("┌───1", "│    ", "│    ", "│    ", "0    ")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, lineBitmap(testCase.points, testLineExtra()).String())
		})
	}
}

func TestLineBitmap_AnchorChar(t *testing.T) {
	extra := testLineExtra()
	extra.UserSelectedStartAnchor = &shape.AnchorChar{ID: "id", DisplayName: "name", Left: 'L', Right: 'R', Top: 'T', Bottom: 'B'}
	extra.UserSelectedEndAnchor = &shape.AnchorChar{ID: "id", DisplayName: "name", Left: 'l', Right: 'r', Top: 't', Bottom: 'b'}

	for _, testCase := range []struct {
		name     string
		points   []geo.Point
		expected string
	}{
		{"left to right", []geo.Point{point(0, 0), point(4, 0)}, "L───r"},
		{"right to left", []geo.Point{point(4, 0), point(0, 0)}, "l───R"},
		{"top to bottom", []geo.Point{point(0, 0), point(0, 4)}, lines("T", "│", "│", "│", "b")},
		{"bottom to top", []geo.Point{point(0, 4), point(0, 0)}, lines("t", "│", "│", "│", "B")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, lineBitmap(testCase.points, extra).String())
		})
	}
}
