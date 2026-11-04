package shape

import (
	"slices"

	"github.com/bluemir/nemokan/internal/geo"
)

// createJointPoints connects seedPoints with horizontal and vertical segments, following
// the direction of each seed point.
func createJointPoints(seedPoints []geo.DirectedPoint) []geo.Point {
	mainPoints := createAnchorPoints(seedPoints)
	if len(mainPoints) == 0 {
		return nil
	}
	directedJointPoints := []geo.DirectedPoint{mainPoints[0]}
	for _, endPoint := range mainPoints[1:] {
		startPoint := directedJointPoints[len(directedJointPoints)-1]
		switch {
		case isOnSameStraightLine(startPoint, endPoint):
		case startPoint.Direction == geo.Horizontal:
			directedJointPoints = append(directedJointPoints, geo.DirectedPoint{Direction: endPoint.Direction, Left: endPoint.Left, Top: startPoint.Top})
		default:
			directedJointPoints = append(directedJointPoints, geo.DirectedPoint{Direction: endPoint.Direction, Left: startPoint.Left, Top: endPoint.Top})
		}
		directedJointPoints = append(directedJointPoints, endPoint)
	}
	jointPoints := make([]geo.Point, len(directedJointPoints))
	for i, point := range directedJointPoints {
		jointPoints[i] = point.Point()
	}
	return reducePoints(jointPoints)
}

// createAnchorPoints adds a middle point between two points with the same direction that
// are not on a straight line.
func createAnchorPoints(seedPoints []geo.DirectedPoint) []geo.DirectedPoint {
	if len(seedPoints) == 0 {
		return nil
	}
	mainPoints := []geo.DirectedPoint{seedPoints[0]}
	for _, endPoint := range seedPoints[1:] {
		startPoint := mainPoints[len(mainPoints)-1]
		if startPoint.Direction == endPoint.Direction && !isOnSameStraightLine(startPoint, endPoint) {
			mainPoints = append(mainPoints, geo.DirectedPoint{
				Direction: startPoint.Direction.Normalized(),
				Left:      (startPoint.Left + endPoint.Left) / 2,
				Top:       (startPoint.Top + endPoint.Top) / 2,
			})
		}
		mainPoints = append(mainPoints, endPoint)
	}
	return mainPoints
}

// reducePoints merges points in the middle of a straight line.
func reducePoints(points []geo.Point) []geo.Point {
	return reducePointsOnce(reducePointsOnce(points))
}

func reducePointsOnce(points []geo.Point) []geo.Point {
	if len(points) == 0 {
		return points
	}
	var result []geo.Point
	for _, p3 := range points[1:] {
		if len(result) >= 2 && isOnSameLine(result[len(result)-2], result[len(result)-1], p3) {
			result[len(result)-1] = p3
		} else {
			result = append(result, p3)
		}
	}

	if len(result) >= 2 && isOnSameLine(points[0], result[0], result[1]) {
		result[0] = points[0]
	} else {
		result = slices.Insert(result, 0, points[0])
	}
	return result
}

// isOnStraightLineInOrder reports whether p2 is between p1 and p3 on a horizontal or
// vertical line.
func isOnStraightLineInOrder(p1, p2, p3 geo.Point) bool {
	return isEquals(p1.Left, p2.Left, p3.Left) && isMonotonic(p1.Top, p2.Top, p3.Top) ||
		isEquals(p1.Top, p2.Top, p3.Top) && isMonotonic(p1.Left, p2.Left, p3.Left)
}

// isOnSameLine reports whether the three points are on a horizontal or vertical line,
// in any order.
func isOnSameLine(p1, p2, p3 geo.Point) bool {
	return isEquals(p1.Left, p2.Left, p3.Left) || isEquals(p1.Top, p2.Top, p3.Top)
}

func isEquals(a, b, c int) bool {
	return a == b && b == c
}

func isMonotonic(a, b, c int) bool {
	return a <= b && b <= c || c <= b && b <= a
}

func isOnSameStraightLine(p1, p2 geo.DirectedPoint) bool {
	return p1.Left == p2.Left || p1.Top == p2.Top
}

func createEdges(jointPoints []geo.Point) []Edge {
	var edges []Edge
	for i := 0; i+1 < len(jointPoints); i++ {
		edges = append(edges, newEdge(jointPoints[i], jointPoints[i+1]))
	}
	return edges
}
