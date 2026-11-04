package interactionbound

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/cursor"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func TestScalableBound(t *testing.T) {
	bound := NewScalableBound("shape", geo.RectByLTWH(1, 2, 3, 4))
	assert.Equal(t, 0.75, bound.Left)
	assert.Equal(t, 1.75, bound.Top)
	assert.Equal(t, 4.25, bound.Right)
	assert.Equal(t, 6.25, bound.Bottom)

	points := bound.Points()
	assert.Len(t, points, 8)
	assert.Equal(t, geo.PointF{Left: 2.5, Top: 1.75}, points[1].Position())
	assert.Equal(t, cursor.ResizeNS, points[1].Cursor())
	assert.Equal(t, "shape", points[1].ShapeID())
}

func TestScalePoint_NewShapeBound(t *testing.T) {
	current := geo.RectByLTRB(10, 10, 20, 20)
	newPoint := geo.PointF{Left: 4.5, Top: 30.5}
	for _, testCase := range []struct {
		handle   ScaleHandle
		expected geo.Rect
	}{
		{TopLeft, geo.RectByLTRB(5, 31, 20, 20)},
		{TopMiddle, geo.RectByLTRB(10, 31, 20, 20)},
		{TopRight, geo.RectByLTRB(10, 31, 3, 20)},
		{MiddleLeft, geo.RectByLTRB(5, 10, 20, 20)},
		{MiddleRight, geo.RectByLTRB(10, 10, 3, 20)},
		{BottomLeft, geo.RectByLTRB(5, 10, 20, 29)},
		{BottomMiddle, geo.RectByLTRB(10, 10, 20, 29)},
		{BottomRight, geo.RectByLTRB(10, 10, 3, 29)},
	} {
		point := &ScalePoint{Handle: testCase.handle}
		assert.Equal(t, testCase.expected, point.NewShapeBound(current, newPoint), testCase.handle)
	}
}

func TestLineBound(t *testing.T) {
	line := shape.NewLine(
		geo.DirectedPoint{Direction: geo.Horizontal, Left: 0, Top: 0},
		geo.DirectedPoint{Direction: geo.Vertical, Left: 4, Top: 4},
		"", "",
	)
	points := NewLineBound(line.ID(), line.Edges()).Points()
	assert.Len(t, points, 4)

	start := points[0].(*LineAnchorPoint)
	assert.Equal(t, shape.AnchorStart, start.Anchor)
	assert.Equal(t, geo.PointF{Left: 0, Top: 0.5}, start.Position())

	end := points[1].(*LineAnchorPoint)
	assert.Equal(t, shape.AnchorEnd, end.Anchor)
	assert.Equal(t, geo.PointF{Left: 4.5, Top: 5}, end.Position())

	horizontalEdge := points[2].(*LineEdgePoint)
	assert.Equal(t, line.Edges()[0].ID, horizontalEdge.EdgeID)
	assert.Equal(t, geo.PointF{Left: 2.5, Top: 0.5}, horizontalEdge.Position())
	assert.Equal(t, cursor.ResizeRow, horizontalEdge.Cursor())
	assert.Equal(t, cursor.ResizeCol, points[3].Cursor())
}
