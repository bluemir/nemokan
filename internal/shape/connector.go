package shape

import (
	"fmt"

	"github.com/bluemir/monosketch/internal/geo"
)

// LineConnector attaches an end of a line to a box.
// Ratio is the position relative to the size of the box, and Offset is the absolute offset
// from the box.
type LineConnector struct {
	LineID string
	Anchor Anchor
	Ratio  geo.PointF
	Offset geo.Point
}

func (c LineConnector) ID() string {
	return connectorID(c.LineID, c.Anchor)
}

func connectorID(lineID string, anchor Anchor) string {
	return fmt.Sprintf("%s@@%d", lineID, anchor)
}

// PointInNewBound returns where the connected end should be when the box moves to boxBound.
func (c LineConnector) PointInNewBound(direction geo.Direction, boxBound geo.Rect) geo.DirectedPoint {
	return geo.DirectedPoint{
		Direction: direction,
		Left:      int(float64(boxBound.Left()) + float64(boxBound.Width()-1)*c.Ratio.Left + float64(c.Offset.Left)),
		Top:       int(float64(boxBound.Top()) + float64(boxBound.Height()-1)*c.Ratio.Top + float64(c.Offset.Top)),
	}
}

// ShapeConnector manages which line ends are attached to which shapes.
type ShapeConnector struct {
	lineConnectors *twoWayQuickMap[LineConnector, shapeID]
}

func NewShapeConnector() *ShapeConnector {
	return &ShapeConnector{lineConnectors: newTwoWayQuickMap[LineConnector, shapeID]()}
}

func RestoreShapeConnector(connectors []SerializableLineConnector) *ShapeConnector {
	result := NewShapeConnector()
	for _, connector := range connectors {
		lineConnector := LineConnector{LineID: connector.LineID, Anchor: connector.Anchor, Ratio: connector.Ratio, Offset: connector.Offset}
		result.lineConnectors.set(lineConnector, shapeID(connector.TargetID))
	}
	return result
}

// AddConnector attaches the anchor of line to shape. Nothing happens if the anchor is not
// around shape.
func (c *ShapeConnector) AddConnector(line *Line, anchor Anchor, shape Shape) {
	anchorPoint := line.StartPoint()
	if anchor == AnchorEnd {
		anchorPoint = line.EndPoint()
	}
	boxBound := shape.Bound()
	around, ok := aroundOf(anchorPoint, boxBound)
	if !ok {
		return
	}

	// TODO: Based on the position of anchor point and shape's bound, the direction of the
	//  anchor must be updated to optimise the initial directions of the line
	c.lineConnectors.set(LineConnector{
		LineID: line.ID(),
		Anchor: anchor,
		Ratio:  connectorRatio(around, anchorPoint, boxBound),
		Offset: connectorOffset(around, anchorPoint, boxBound),
	}, shapeID(shape.ID()))
}

func (c *ShapeConnector) Connectors(shape Shape) []LineConnector {
	return c.lineConnectors.keys(shape.ID())
}

func (c *ShapeConnector) RemoveConnector(line *Line, anchor Anchor) {
	c.lineConnectors.removeKey(connectorID(line.ID(), anchor))
}

func (c *ShapeConnector) HasConnector(line *Line, anchor Anchor) bool {
	_, ok := c.lineConnectors.get(connectorID(line.ID(), anchor))
	return ok
}

// RemoveShape removes the connectors to shape, and the connectors of shape if it is a line.
func (c *ShapeConnector) RemoveShape(shape Shape) {
	c.lineConnectors.removeValue(shape.ID())
	if line, ok := shape.(*Line); ok {
		c.RemoveConnector(line, AnchorStart)
		c.RemoveConnector(line, AnchorEnd)
	}
}

func (c *ShapeConnector) Serializable() []SerializableLineConnector {
	var result []SerializableLineConnector
	c.lineConnectors.pairs(func(connector LineConnector, target shapeID) {
		result = append(result, connector.Serializable(target.ID()))
	})
	return result
}

func (c LineConnector) Serializable(targetID string) SerializableLineConnector {
	return SerializableLineConnector{LineID: c.LineID, Anchor: c.Anchor, TargetID: targetID, Ratio: c.Ratio, Offset: c.Offset}
}

// maxConnectDistance is how far from the box border a line end can connect.
const maxConnectDistance = 1

type around int

const (
	aroundLeft around = iota
	aroundTop
	aroundRight
	aroundBottom
)

// ConnectableShape returns the last candidate that point can connect to.
func ConnectableShape(point geo.Point, candidates []Shape) (Shape, bool) {
	for i := len(candidates) - 1; i >= 0; i-- {
		if canConnect(point, candidates[i]) {
			return candidates[i], true
		}
	}
	return nil, false
}

// canConnect reports whether point is around a rectangle or a text.
func canConnect(point geo.Point, shape Shape) bool {
	switch shape.(type) {
	case *Rectangle, *Text:
		sides := detectAround(point, shape.Bound())
		return sides.left || sides.right || sides.top || sides.bottom
	}
	return false
}

// aroundOf decides the side of boxBound the anchor connects to. A corner goes to the side
// that matches the direction of the anchor.
func aroundOf(anchorPoint geo.DirectedPoint, boxBound geo.Rect) (around, bool) {
	sides := detectAround(anchorPoint.Point(), boxBound)
	isHorizontal := anchorPoint.Direction == geo.Horizontal

	switch {
	case sides.left:
		switch {
		case sides.top && !isHorizontal:
			return aroundTop, true
		case sides.bottom && !isHorizontal:
			return aroundBottom, true
		}
		return aroundLeft, true
	case sides.top:
		if sides.right && isHorizontal {
			return aroundRight, true
		}
		return aroundTop, true
	case sides.right:
		if sides.bottom && !isHorizontal {
			return aroundBottom, true
		}
		return aroundRight, true
	case sides.bottom:
		return aroundBottom, true
	}
	return 0, false
}

// sidesAround tells which sides of a box a point is near.
type sidesAround struct {
	left   bool
	right  bool
	top    bool
	bottom bool
}

func detectAround(point geo.Point, boxBound geo.Rect) sidesAround {
	return sidesAround{
		left:   isAround(point.Left, boxBound.Left(), boxBound.Left()) && isAround(point.Top, boxBound.Top(), boxBound.Bottom()),
		right:  isAround(point.Left, boxBound.Right(), boxBound.Right()) && isAround(point.Top, boxBound.Top(), boxBound.Bottom()),
		top:    isAround(point.Top, boxBound.Top(), boxBound.Top()) && isAround(point.Left, boxBound.Left(), boxBound.Right()),
		bottom: isAround(point.Top, boxBound.Bottom(), boxBound.Bottom()) && isAround(point.Left, boxBound.Left(), boxBound.Right()),
	}
}

func isAround(value, lower, upper int) bool {
	return lower-maxConnectDistance <= value && value <= upper+maxConnectDistance
}

func connectorRatio(side around, anchorPoint geo.DirectedPoint, boxBound geo.Rect) geo.PointF {
	leftRatio := float64(anchorPoint.Left-boxBound.Left()) / float64(max(boxBound.Width()-1, 1))
	topRatio := float64(anchorPoint.Top-boxBound.Top()) / float64(max(boxBound.Height()-1, 1))
	switch side {
	case aroundLeft:
		return geo.PointF{Left: 0, Top: clamp(topRatio, 0, 1)}
	case aroundTop:
		return geo.PointF{Left: clamp(leftRatio, 0, 1), Top: 0}
	case aroundRight:
		return geo.PointF{Left: 1, Top: clamp(topRatio, 0, 1)}
	default:
		return geo.PointF{Left: clamp(leftRatio, 0, 1), Top: 1}
	}
}

func connectorOffset(side around, anchorPoint geo.DirectedPoint, boxBound geo.Rect) geo.Point {
	switch side {
	case aroundLeft:
		return geo.Point{Left: anchorPoint.Left - boxBound.Left(), Top: offsetToRange(anchorPoint.Top, boxBound.Top(), boxBound.Bottom())}
	case aroundTop:
		return geo.Point{Left: offsetToRange(anchorPoint.Left, boxBound.Left(), boxBound.Right()), Top: anchorPoint.Top - boxBound.Top()}
	case aroundRight:
		return geo.Point{Left: anchorPoint.Left - boxBound.Right(), Top: offsetToRange(anchorPoint.Top, boxBound.Top(), boxBound.Bottom())}
	default:
		return geo.Point{Left: offsetToRange(anchorPoint.Left, boxBound.Left(), boxBound.Right()), Top: anchorPoint.Top - boxBound.Bottom()}
	}
}

// offsetToRange returns how far value is outside [lower, upper], or 0 if inside.
func offsetToRange(value, lower, upper int) int {
	switch {
	case value < lower:
		return value - lower
	case value > upper:
		return value - upper
	}
	return 0
}

func clamp(value, low, high float64) float64 {
	return min(max(value, low), high)
}
