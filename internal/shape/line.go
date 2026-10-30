package shape

import (
	"fmt"
	"slices"

	"github.com/bluemir/monosketch/internal/geo"
)

// Line connects two directed end points with straight edges.
//
// At first the edges follow the directions of the two end points. For example,
//
//	Start (0, 0, Horizontal), End (5, 3, Vertical)   Start (0, 0, Horizontal), End (5, 3, Horizontal)
//	x----+                                           x-+
//	     |                                             |
//	     |                                             |
//	     x                                             +--x
//
// Once an edge is moved, the line no longer depends on the directions.
//
// TODO: Extract move anchor point and move edge code to use case
type Line struct {
	base
	startPoint  geo.DirectedPoint
	endPoint    geo.DirectedPoint
	jointPoints []geo.Point
	edges       []Edge
	extra       LineExtra
	// confirmedJointPoints are set once an edge is moved.
	confirmedJointPoints []geo.Point
}

type Anchor int

const (
	AnchorStart Anchor = iota
	AnchorEnd
)

func (a Anchor) MarshalText() ([]byte, error) {
	if a == AnchorEnd {
		return []byte("end"), nil
	}
	return []byte("start"), nil
}

func (a *Anchor) UnmarshalText(text []byte) error {
	switch string(text) {
	case "start":
		*a = AnchorStart
	case "end":
		*a = AnchorEnd
	default:
		return fmt.Errorf("unknown anchor: %q", text)
	}
	return nil
}

type AnchorPointUpdate struct {
	Anchor Anchor
	Point  geo.DirectedPoint
}

// NewLine generates an id if id is empty.
func NewLine(startPoint, endPoint geo.DirectedPoint, id, parentID string) *Line {
	jointPoints := createJointPoints([]geo.DirectedPoint{startPoint, endPoint})
	return &Line{
		base:        newBase(id, parentID),
		startPoint:  startPoint,
		endPoint:    endPoint,
		jointPoints: jointPoints,
		edges:       createEdges(jointPoints),
		extra:       defaultLineExtra,
	}
}

func restoreLine(serializable *SerializableLine, parentID string) *Line {
	line := NewLine(serializable.StartPoint, serializable.EndPoint, serializable.actualID(), parentID)
	line.jointPoints = serializable.JointPoints
	if serializable.WasMovingEdge {
		line.confirmedJointPoints = serializable.JointPoints
	}
	line.edges = createEdges(line.jointPoints)
	line.extra = serializable.Extra.restore()
	line.versionCode = serializable.VersionCode
	return line
}

func (l *Line) StartPoint() geo.DirectedPoint { return l.startPoint }
func (l *Line) EndPoint() geo.DirectedPoint   { return l.endPoint }
func (l *Line) Edges() []Edge                 { return l.edges }
func (l *Line) Extra() LineExtra              { return l.extra }

func (l *Line) ReducedJointPoints() []geo.Point {
	return reducePoints(l.jointPoints)
}

func (l *Line) Bound() geo.Rect {
	points := l.ReducedJointPoints()
	left, right := points[0].Left, points[0].Left
	top, bottom := points[0].Top, points[0].Top
	for _, point := range points {
		left = min(left, point.Left)
		right = max(right, point.Left)
		top = min(top, point.Top)
		bottom = max(bottom, point.Top)
	}
	return geo.RectByLTRB(left, top, right, bottom)
}

func (l *Line) Serializable(isIDIncluded bool) SerializableShape {
	return &SerializableLine{
		serializableBase: serializableBase{ID: l.id, IsIDTemporary: !isIDIncluded, VersionCode: l.versionCode},
		StartPoint:       l.startPoint,
		EndPoint:         l.endPoint,
		JointPoints:      l.jointPoints,
		Extra:            serializableLineExtra(l.extra),
		WasMovingEdge:    l.WasMovingEdge(),
	}
}

// SetBound moves the line so that its joint points start at the top left of newBound.
func (l *Line) SetBound(newBound geo.Rect) {
	left, top := l.jointPoints[0].Left, l.jointPoints[0].Top
	for _, point := range l.jointPoints {
		left = min(left, point.Left)
		top = min(top, point.Top)
	}
	offset := geo.Point{Left: newBound.Left() - left, Top: newBound.Top() - top}
	if offset == (geo.Point{}) {
		return
	}
	l.update(func() bool {
		l.startPoint = l.startPoint.Plus(offset)
		l.endPoint = l.endPoint.Plus(offset)
		l.jointPoints = movePoints(l.jointPoints, offset)
		l.confirmedJointPoints = movePoints(l.confirmedJointPoints, offset)
		l.edges = createEdges(l.jointPoints)
		return true
	})
}

func movePoints(points []geo.Point, offset geo.Point) []geo.Point {
	if points == nil {
		return nil
	}
	moved := make([]geo.Point, len(points))
	for i, point := range points {
		moved[i] = point.Plus(offset)
	}
	return moved
}

func (l *Line) SetExtra(newExtra Extra) {
	extra, ok := newExtra.(LineExtra)
	if !ok {
		panic(fmt.Sprintf("new extra is not a LineExtra (%T)", newExtra))
	}
	if extra == l.extra {
		return
	}
	l.update(func() bool {
		l.extra = extra
		return true
	})
}

// MoveAnchorPoint moves the start or end point.
//
// If the edges have never been moved, or there are only 2 joint points, the edges are
// created again from the two end points and their directions.
//
// If the edges have been moved and justMoveAnchor is true, only the anchor and its adjacent
// point are moved:
//
//	+-----+         +------------+
//	      |   ->                 |
//	      o                      |
//	                             o
//
// Otherwise the anchor is moved, and a new joint point is added if the new position is not
// on the line of the edge it was on:
//
//	+-------o  ->  +---------------x  (same line)
//	+-------o  ->  +----------+       (different line)
//	                          |
//	                          x
//
// isReduceRequired merges the points on a straight line, for when the move is confirmed.
func (l *Line) MoveAnchorPoint(anchorPointUpdate AnchorPointUpdate, isReduceRequired, justMoveAnchor bool) {
	l.update(func() bool {
		switch anchorPointUpdate.Anchor {
		case AnchorStart:
			l.startPoint = anchorPointUpdate.Point
		case AnchorEnd:
			l.endPoint = anchorPointUpdate.Point
		}

		isEdgeUpdated := len(l.confirmedJointPoints) > 0
		var newJointPoints []geo.Point
		switch {
		case !isEdgeUpdated || len(l.confirmedJointPoints) == 2:
			newJointPoints = createJointPoints([]geo.DirectedPoint{l.startPoint, l.endPoint})

		case justMoveAnchor && len(l.confirmedJointPoints) > 2:
			newJointPoints = slices.Clone(l.confirmedJointPoints)
			anchorIndex, affectedIndex := 0, 1
			if anchorPointUpdate.Anchor == AnchorEnd {
				anchorIndex, affectedIndex = len(newJointPoints)-1, len(newJointPoints)-2
			}
			newPoint := anchorPointUpdate.Point.Point()
			if newJointPoints[anchorIndex].Left == newJointPoints[affectedIndex].Left {
				newJointPoints[affectedIndex].Left = newPoint.Left
			} else {
				newJointPoints[affectedIndex].Top = newPoint.Top
			}
			newJointPoints[anchorIndex] = newPoint

		default:
			newJointPoint, ok := newAnchorJointPoint(l.confirmedJointPoints, anchorPointUpdate)
			newJointPoints = slices.Clone(l.confirmedJointPoints)
			anchorIndex, newJointPointIndex := 0, 1
			if anchorPointUpdate.Anchor == AnchorEnd {
				anchorIndex, newJointPointIndex = len(newJointPoints)-1, len(newJointPoints)-1
			}
			newJointPoints[anchorIndex] = anchorPointUpdate.Point.Point()
			if ok {
				newJointPoints = slices.Insert(newJointPoints, newJointPointIndex, newJointPoint)
			}
		}

		isUpdated := !slices.Equal(newJointPoints, l.jointPoints)
		if isReduceRequired {
			l.jointPoints = reducePoints(newJointPoints)
		} else {
			l.jointPoints = newJointPoints
		}
		if isReduceRequired && isEdgeUpdated {
			l.confirmedJointPoints = l.jointPoints
		}
		l.edges = createEdges(l.jointPoints)
		return isUpdated
	})
}

// newAnchorJointPoint returns the joint point to add when the anchor moves off the line of
// its edge. It returns false if the new position is on the same line.
func newAnchorJointPoint(points []geo.Point, anchorPointUpdate AnchorPointUpdate) (geo.Point, bool) {
	anchorIndex, previousIndex := 0, 1
	if anchorPointUpdate.Anchor == AnchorEnd {
		anchorIndex, previousIndex = len(points)-1, len(points)-2
	}
	anchorPoint := points[anchorIndex]
	previousPoint := points[previousIndex]
	updatePoint := anchorPointUpdate.Point.Point()

	if isOnSameLine(anchorPoint, previousPoint, updatePoint) {
		return geo.Point{}, false
	}
	if anchorPoint.Top == previousPoint.Top {
		return geo.Point{Left: updatePoint.Left, Top: anchorPoint.Top}, true
	}
	return geo.Point{Left: anchorPoint.Left, Top: updatePoint.Top}, true
}

// MoveEdge moves the edge with edgeID so that its line contains point. The two anchor
// points do not move. If the edge is the first or the last one, a new edge is added.
//
//	x---o---x  ->  x       x
//	               |       |
//	               +---o---+
//
// After an edge is moved, the line no longer depends on the directions of its end points.
//
// isReduceRequired merges adjacent edges on the same line with the same direction:
//
//	1-----2-----3  ->  1-----------3
//	1-----3-----2  ->  1-----3-----2
func (l *Line) MoveEdge(edgeID int, point geo.Point, isReduceRequired bool) {
	l.update(func() bool {
		edgeIndex := slices.IndexFunc(l.edges, func(edge Edge) bool { return edge.ID == edgeID })
		if edgeIndex < 0 {
			return false
		}

		edge := l.edges[edgeIndex]
		movedEdge := edge.translate(point)
		if !isReduceRequired && edge == movedEdge {
			return false
		}

		lastEdgeIndex := len(l.edges) - 1
		newJointPoints := slices.Clone(l.jointPoints)
		switch {
		case edgeIndex == 0 && edgeIndex == lastEdgeIndex:
			newJointPoints = slices.Insert(newJointPoints, 1, movedEdge.StartPoint, movedEdge.EndPoint)
		case edgeIndex == 0:
			newJointPoints = slices.Insert(newJointPoints, 1, movedEdge.StartPoint)
			newJointPoints[2] = movedEdge.EndPoint
		case edgeIndex == lastEdgeIndex:
			startPointIndex := len(newJointPoints) - 2
			newJointPoints[startPointIndex] = movedEdge.StartPoint
			newJointPoints = slices.Insert(newJointPoints, startPointIndex+1, movedEdge.EndPoint)
		default:
			// Edges are created from joint points in order, so the edge starts at the joint
			// point with the same index.
			newJointPoints[edgeIndex] = movedEdge.StartPoint
			newJointPoints[edgeIndex+1] = movedEdge.EndPoint
		}

		isUpdated := !slices.Equal(l.jointPoints, newJointPoints)
		if isReduceRequired {
			l.jointPoints = reducePoints(newJointPoints)
		} else {
			l.jointPoints = newJointPoints
		}
		l.confirmedJointPoints = l.jointPoints

		newEdges := createEdges(l.jointPoints)
		if !isReduceRequired {
			newEdgeIndex := edgeIndex
			if edgeIndex == 0 {
				newEdgeIndex = 1
			}
			// Keep the id of the edge being moved.
			newEdges[newEdgeIndex].ID = edge.ID
		}
		l.edges = newEdges
		return isUpdated
	})
}

func (l *Line) Direction(anchor Anchor) geo.Direction {
	if anchor == AnchorEnd {
		return l.endPoint.Direction
	}
	return l.startPoint.Direction
}

func (l *Line) WasMovingEdge() bool {
	return len(l.confirmedJointPoints) > 0
}

func (l *Line) Contains(point geo.Point) bool {
	return slices.ContainsFunc(l.edges, func(edge Edge) bool { return edge.contains(point) })
}

func (l *Line) IsVertex(geo.Point) bool {
	// TODO: Correct this to any of its joint points
	return false
}

func (l *Line) IsOverlapped(rect geo.Rect) bool {
	return slices.ContainsFunc(l.edges, func(edge Edge) bool {
		return geo.RectByLTRB(edge.StartPoint.Left, edge.StartPoint.Top, edge.EndPoint.Left, edge.EndPoint.Top).IsOverlapped(rect)
	})
}

type Edge struct {
	ID         int
	StartPoint geo.Point
	EndPoint   geo.Point
}

var lastEdgeID int

func newEdge(startPoint, endPoint geo.Point) Edge {
	lastEdgeID++
	return Edge{ID: lastEdgeID, StartPoint: startPoint, EndPoint: endPoint}
}

func (e Edge) IsHorizontal() bool {
	return e.StartPoint.Top == e.EndPoint.Top
}

func (e Edge) MiddleLeft() float64 { return float64(e.StartPoint.Left+e.EndPoint.Left) / 2 }
func (e Edge) MiddleTop() float64  { return float64(e.StartPoint.Top+e.EndPoint.Top) / 2 }

// translate moves the edge so that its line contains point.
func (e Edge) translate(point geo.Point) Edge {
	if e.IsHorizontal() {
		e.StartPoint.Top = point.Top
		e.EndPoint.Top = point.Top
	} else {
		e.StartPoint.Left = point.Left
		e.EndPoint.Left = point.Left
	}
	return e
}

func (e Edge) contains(point geo.Point) bool {
	return isOnStraightLineInOrder(e.StartPoint, point, e.EndPoint)
}
