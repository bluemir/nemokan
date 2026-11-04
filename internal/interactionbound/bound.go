// Package interactionbound defines the points around a selected shape the user drags to scale
// or move it.
package interactionbound

import (
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

// Bound is the interaction bound of a shape: *ScalableBound or *LineBound.
type Bound interface {
	Points() []Point
	isBound()
}

// ScalableBound surrounds a shape with a static bound, such as a rectangle or a text.
type ScalableBound struct {
	Left   float64
	Top    float64
	Right  float64
	Bottom float64
	points []Point
}

func NewScalableBound(shapeID string, shapeBound geo.Rect) *ScalableBound {
	left := float64(shapeBound.Left()) - 0.25
	top := float64(shapeBound.Top()) - 0.25
	right := float64(shapeBound.Right()) + 1.0 + 0.25
	bottom := float64(shapeBound.Bottom()) + 1.0 + 0.25
	horizontalMiddle := (left + right) / 2
	verticalMiddle := (top + bottom) / 2

	scalePoint := func(handle ScaleHandle, left, top float64) Point {
		return &ScalePoint{pointBase: pointBase{shapeID: shapeID, position: geo.PointF{Left: left, Top: top}}, Handle: handle}
	}
	return &ScalableBound{
		Left:   left,
		Top:    top,
		Right:  right,
		Bottom: bottom,
		points: []Point{
			scalePoint(TopLeft, left, top),
			scalePoint(TopMiddle, horizontalMiddle, top),
			scalePoint(TopRight, right, top),
			scalePoint(MiddleLeft, left, verticalMiddle),
			scalePoint(MiddleRight, right, verticalMiddle),
			scalePoint(BottomLeft, left, bottom),
			scalePoint(BottomMiddle, horizontalMiddle, bottom),
			scalePoint(BottomRight, right, bottom),
		},
	}
}

// NewOutlineBound only shows the bound of a shape which cannot be scaled, such as a group.
func NewOutlineBound(shapeBound geo.Rect) *ScalableBound {
	bound := NewScalableBound("", shapeBound)
	bound.points = nil
	return bound
}

func (b *ScalableBound) Points() []Point { return b.points }
func (*ScalableBound) isBound()          {}

// LineBound has the two anchors of a line and the middle of each edge.
type LineBound struct {
	points []Point
}

func NewLineBound(shapeID string, edges []shape.Edge) *LineBound {
	var reducedEdges []shape.Edge
	for _, edge := range edges {
		if edge.StartPoint != edge.EndPoint {
			reducedEdges = append(reducedEdges, edge)
		}
	}
	if len(reducedEdges) == 0 {
		reducedEdges = edges[:1]
	}

	points := []Point{
		lineAnchorPoint(shapeID, shape.AnchorStart, reducedEdges[0]),
		lineAnchorPoint(shapeID, shape.AnchorEnd, reducedEdges[len(reducedEdges)-1]),
	}
	for _, edge := range reducedEdges {
		if edge.StartPoint == edge.EndPoint {
			continue
		}
		points = append(points, &LineEdgePoint{
			pointBase:    pointBase{shapeID: shapeID, position: geo.PointF{Left: edge.MiddleLeft() + 0.5, Top: edge.MiddleTop() + 0.5}},
			EdgeID:       edge.ID,
			isHorizontal: edge.IsHorizontal(),
		})
	}
	return &LineBound{points: points}
}

func lineAnchorPoint(shapeID string, anchor shape.Anchor, edge shape.Edge) *LineAnchorPoint {
	point, anotherPoint := edge.StartPoint, edge.EndPoint
	if anchor == shape.AnchorEnd {
		point, anotherPoint = edge.EndPoint, edge.StartPoint
	}

	var horizontalOffset float64
	switch {
	case !edge.IsHorizontal():
		horizontalOffset = 0.5
	case point == anotherPoint && anchor == shape.AnchorStart:
		horizontalOffset = 0
	case point == anotherPoint:
		horizontalOffset = 1
	case point.Left < anotherPoint.Left:
		horizontalOffset = 0
	default:
		horizontalOffset = 1
	}

	var verticalOffset float64
	switch {
	case edge.IsHorizontal():
		verticalOffset = 0.5
	case point.Top <= anotherPoint.Top:
		verticalOffset = 0
	default:
		verticalOffset = 1
	}

	return &LineAnchorPoint{
		pointBase: pointBase{
			shapeID:  shapeID,
			position: geo.PointF{Left: float64(point.Left) + horizontalOffset, Top: float64(point.Top) + verticalOffset},
		},
		Anchor: anchor,
	}
}

func (b *LineBound) Points() []Point { return b.points }
func (*LineBound) isBound()          {}
