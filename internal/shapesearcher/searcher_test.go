package shapesearcher

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/bitmapmanager"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func point(left, top int) geo.Point {
	return geo.Point{Left: left, Top: top}
}

func newTestSearcher(shapes ...shape.Shape) *Searcher {
	shapeManager := shape.NewManager()
	searcher := New(shapeManager, bitmapmanager.New())
	for _, target := range shapes {
		shapeManager.Execute(shape.AddShape{Shape: target})
		searcher.Register(target)
	}
	return searcher
}

func TestToZoneAddress(t *testing.T) {
	assert.Equal(t, zoneAddress{row: 0, column: 0}, toZoneAddress(0, 15))
	assert.Equal(t, zoneAddress{row: 1, column: 0}, toZoneAddress(16, 0))
	assert.Equal(t, zoneAddress{row: -1, column: -1}, toZoneAddress(-1, -16))
	assert.Equal(t, zoneAddress{row: -2, column: 0}, toZoneAddress(-17, 0))
}

func TestSearcher_Shapes(t *testing.T) {
	// A border-only rectangle has pixels on its border only.
	bottom := shape.NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	top := shape.NewRectangle(geo.RectByLTWH(4, 4, 30, 30), "", "")
	searcher := newTestSearcher(bottom, top)

	assert.Equal(t, []shape.Shape{bottom}, searcher.Shapes(point(0, 0)))
	assert.Empty(t, searcher.Shapes(point(2, 2)))
	assert.Equal(t, []shape.Shape{bottom, top}, searcher.Shapes(point(4, 4)))
	assert.Equal(t, []shape.Shape{top}, searcher.Shapes(point(33, 20)))
}

func TestSearcher_ShapesInZone(t *testing.T) {
	near := shape.NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	far := shape.NewRectangle(geo.RectByLTWH(40, 40, 5, 5), "", "")
	searcher := newTestSearcher(near, far)

	assert.Equal(t, []shape.Shape{near}, searcher.ShapesInZone(geo.RectByLTWH(2, 2, 10, 10)))
	assert.Equal(t, []shape.Shape{near, far}, searcher.ShapesInZone(geo.RectByLTWH(0, 0, 50, 50)))
}

func TestSearcher_Clear(t *testing.T) {
	rectangle := shape.NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	searcher := newTestSearcher(rectangle)

	searcher.Clear(geo.RectByLTWH(0, 0, 1, 1))
	assert.Empty(t, searcher.Shapes(point(0, 0)))

	searcher.Register(rectangle)
	assert.Equal(t, []shape.Shape{rectangle}, searcher.Shapes(point(0, 0)))
}

func TestSearcher_EdgeDirection(t *testing.T) {
	rectangle := shape.NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	searcher := newTestSearcher(rectangle)

	for _, testCase := range []struct {
		name       string
		point      geo.Point
		expected   geo.Direction
		isExpected bool
	}{
		{"left edge", point(0, 2), geo.Vertical, true},
		{"right edge", point(4, 2), geo.Vertical, true},
		{"top edge", point(2, 0), geo.Horizontal, true},
		{"bottom edge", point(2, 4), geo.Horizontal, true},
		{"vertex", point(0, 0), geo.Horizontal, false},
		{"inside", point(2, 2), geo.Horizontal, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			direction, ok := searcher.EdgeDirection(testCase.point)
			assert.Equal(t, testCase.isExpected, ok)
			if ok {
				assert.Equal(t, testCase.expected, direction)
			}
		})
	}
}
