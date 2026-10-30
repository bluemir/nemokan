package interactionbound

import (
	"math"

	"github.com/bluemir/monosketch/internal/cursor"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// Point is a point the user drags to change a shape: *ScalePoint, *LineAnchorPoint or
// *LineEdgePoint. Position is the center of the point, in board units.
type Point interface {
	ShapeID() string
	Position() geo.PointF
	Cursor() cursor.Cursor
	isPoint()
}

type pointBase struct {
	shapeID  string
	position geo.PointF
}

func (p pointBase) ShapeID() string      { return p.shapeID }
func (p pointBase) Position() geo.PointF { return p.position }
func (pointBase) isPoint()               {}

// ScaleHandle is which part of the bound a ScalePoint moves.
type ScaleHandle int

const (
	TopLeft ScaleHandle = iota
	TopMiddle
	TopRight
	MiddleLeft
	MiddleRight
	BottomLeft
	BottomMiddle
	BottomRight
)

type ScalePoint struct {
	pointBase
	Handle ScaleHandle
}

func (p *ScalePoint) Cursor() cursor.Cursor {
	switch p.Handle {
	case TopLeft, BottomRight:
		return cursor.ResizeNWSE
	case TopRight, BottomLeft:
		return cursor.ResizeNESW
	case TopMiddle, BottomMiddle:
		return cursor.ResizeNS
	default:
		return cursor.ResizeEW
	}
}

// NewShapeBound returns currentBound with the handled edges moved to newPoint.
func (p *ScalePoint) NewShapeBound(currentBound geo.Rect, newPoint geo.PointF) geo.Rect {
	left, top, right, bottom := currentBound.Left(), currentBound.Top(), currentBound.Right(), currentBound.Bottom()
	switch p.Handle {
	case TopLeft, MiddleLeft, BottomLeft:
		left = adjustUpperBound(newPoint.Left)
	case TopRight, MiddleRight, BottomRight:
		right = adjustLowerBound(newPoint.Left)
	}
	switch p.Handle {
	case TopLeft, TopMiddle, TopRight:
		top = adjustUpperBound(newPoint.Top)
	case BottomLeft, BottomMiddle, BottomRight:
		bottom = adjustLowerBound(newPoint.Top)
	}
	return geo.RectByLTRB(left, top, right, bottom)
}

// adjustUpperBound keeps the mouse cursor on the interaction point for left and top edges.
func adjustUpperBound(value float64) int {
	return int(math.Floor(value + 0.8))
}

// adjustLowerBound keeps the mouse cursor on the interaction point for right and bottom edges.
func adjustLowerBound(value float64) int {
	return int(math.Floor(value - 0.8))
}

type LineAnchorPoint struct {
	pointBase
	Anchor shape.Anchor
}

func (*LineAnchorPoint) Cursor() cursor.Cursor { return cursor.Move }

type LineEdgePoint struct {
	pointBase
	EdgeID       int
	isHorizontal bool
}

func (p *LineEdgePoint) Cursor() cursor.Cursor {
	if p.isHorizontal {
		return cursor.ResizeRow
	}
	return cursor.ResizeCol
}
