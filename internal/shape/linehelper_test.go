package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/geo"
)

func point(left, top int) geo.Point {
	return geo.Point{Left: left, Top: top}
}

func directed(direction geo.Direction, left, top int) geo.DirectedPoint {
	return geo.DirectedPoint{Direction: direction, Left: left, Top: top}
}

func TestCreateJointPoints(t *testing.T) {
	const (
		h = geo.Horizontal
		v = geo.Vertical
	)
	for _, testCase := range []struct {
		name     string
		start    geo.DirectedPoint
		end      geo.DirectedPoint
		expected []geo.Point
	}{
		{"same horizontal line HH", directed(h, 0, 0), directed(h, 10, 0), []geo.Point{point(0, 0), point(10, 0)}},
		{"same horizontal line HV", directed(h, 0, 0), directed(v, 10, 0), []geo.Point{point(0, 0), point(10, 0)}},
		{"same horizontal line VV", directed(v, 0, 0), directed(v, 10, 0), []geo.Point{point(0, 0), point(10, 0)}},
		{"same horizontal line VH", directed(v, 0, 0), directed(h, 10, 0), []geo.Point{point(0, 0), point(10, 0)}},
		{"same vertical line HH", directed(h, 0, 0), directed(h, 0, 10), []geo.Point{point(0, 0), point(0, 10)}},
		{"same vertical line HV", directed(h, 0, 0), directed(v, 0, 10), []geo.Point{point(0, 0), point(0, 10)}},
		{"same vertical line VV", directed(v, 0, 0), directed(v, 0, 10), []geo.Point{point(0, 0), point(0, 10)}},
		{"same vertical line VH", directed(v, 0, 0), directed(h, 0, 10), []geo.Point{point(0, 0), point(0, 10)}},
		{"different lines HH", directed(h, 0, 0), directed(h, 10, 10), []geo.Point{point(0, 0), point(5, 0), point(5, 10), point(10, 10)}},
		{"different lines HV", directed(h, 0, 0), directed(v, 10, 10), []geo.Point{point(0, 0), point(10, 0), point(10, 10)}},
		{"different lines VV", directed(v, 0, 0), directed(v, 10, 10), []geo.Point{point(0, 0), point(0, 5), point(10, 5), point(10, 10)}},
		{"different lines VH", directed(v, 0, 0), directed(h, 10, 10), []geo.Point{point(0, 0), point(0, 10), point(10, 10)}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, createJointPoints([]geo.DirectedPoint{testCase.start, testCase.end}))
		})
	}
}

func TestReducePoints_NotReduced(t *testing.T) {
	for _, points := range [][]geo.Point{
		{},
		{point(0, 0)},
		{point(0, 0), point(0, 0)},
		{point(0, 0), point(10, 0), point(10, 10)},
	} {
		assert.Equal(t, points, reducePoints(points))
	}
}

func TestReducePoints_Reduced(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		points   []geo.Point
		expected []geo.Point
	}{
		{"identical", []geo.Point{point(0, 0), point(0, 0), point(0, 0)}, []geo.Point{point(0, 0), point(0, 0)}},
		{"same first", []geo.Point{point(0, 0), point(0, 0), point(0, 0), point(0, 0), point(10, 0)}, []geo.Point{point(0, 0), point(10, 0)}},
		{"same middle", []geo.Point{point(0, 0), point(0, 10), point(0, 10), point(0, 10), point(0, 10), point(10, 10)}, []geo.Point{point(0, 0), point(0, 10), point(10, 10)}},
		{"same last", []geo.Point{point(0, 0), point(10, 0), point(10, 0), point(10, 0), point(10, 0)}, []geo.Point{point(0, 0), point(10, 0)}},
		{"monotonic", []geo.Point{point(0, 0), point(5, 0), point(10, 0)}, []geo.Point{point(0, 0), point(10, 0)}},
		{"monotonic 2", []geo.Point{point(0, 0), point(2, 0), point(4, 0), point(6, 0), point(8, 0), point(10, 0)}, []geo.Point{point(0, 0), point(10, 0)}},
		{"non monotonic", []geo.Point{point(0, 0), point(0, 5), point(0, 5), point(0, 0), point(10, 0)}, []geo.Point{point(0, 0), point(10, 0)}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, reducePoints(testCase.points))
		})
	}
}

func TestIsOnStraightLineInOrder(t *testing.T) {
	for _, points := range [][3]geo.Point{
		{point(0, 0), point(2, 0), point(1, 0)},
		{point(2, 0), point(0, 0), point(1, 0)},
		{point(0, 0), point(0, 2), point(0, 1)},
		{point(0, 2), point(0, 0), point(0, 1)},
	} {
		assert.False(t, isOnStraightLineInOrder(points[0], points[1], points[2]), points)
	}

	for _, points := range [][3]geo.Point{
		// Horizontal
		{point(0, 0), point(1, 0), point(2, 0)},
		{point(2, 0), point(1, 0), point(0, 0)},
		{point(0, 0), point(0, 0), point(1, 0)},
		{point(0, 0), point(1, 0), point(1, 0)},
		{point(1, 0), point(1, 0), point(0, 0)},
		{point(1, 0), point(1, 0), point(1, 0)},
		// Vertical
		{point(0, 0), point(0, 1), point(0, 2)},
		{point(0, 2), point(0, 1), point(0, 0)},
		{point(0, 0), point(0, 0), point(0, 1)},
		{point(0, 0), point(0, 1), point(0, 1)},
		{point(0, 1), point(0, 1), point(0, 0)},
		{point(0, 1), point(0, 1), point(0, 1)},
	} {
		assert.True(t, isOnStraightLineInOrder(points[0], points[1], points[2]), points)
	}
}

func TestIsOnSameLine(t *testing.T) {
	for _, points := range [][3]geo.Point{
		{point(0, 0), point(2, 0), point(1, 0)},
		{point(2, 0), point(0, 0), point(1, 0)},
		{point(0, 0), point(0, 2), point(0, 1)},
		{point(0, 2), point(0, 0), point(0, 1)},
	} {
		assert.True(t, isOnSameLine(points[0], points[1], points[2]), points)
	}
}

func TestCreateEdges(t *testing.T) {
	edges := createEdges([]geo.Point{point(0, 0), point(1, 0), point(1, 1)})
	assert.Len(t, edges, 2)

	assert.Equal(t, point(0, 0), edges[0].StartPoint)
	assert.Equal(t, point(1, 0), edges[0].EndPoint)
	assert.Equal(t, point(1, 0), edges[1].StartPoint)
	assert.Equal(t, point(1, 1), edges[1].EndPoint)
}

func TestDashPattern_IsGap(t *testing.T) {
	for _, testCase := range []struct {
		pattern  StraightStrokeDashPattern
		expected []bool
	}{
		{StraightStrokeDashPattern{Dash: 1, Gap: 0, Offset: 0}, []bool{false, false, false, false}},
		{StraightStrokeDashPattern{Dash: 1, Gap: 1, Offset: 0}, []bool{false, true, false, true, false, true}},
		{StraightStrokeDashPattern{Dash: 1, Gap: 1, Offset: 1}, []bool{true, false, true, false, true, false}},
		{StraightStrokeDashPattern{Dash: 1, Gap: 1, Offset: -1}, []bool{true, false, true, false, true, false}},
		{StraightStrokeDashPattern{Dash: 2, Gap: 1, Offset: 0}, []bool{false, false, true, false, false, true}},
		{StraightStrokeDashPattern{Dash: 2, Gap: 1, Offset: 1}, []bool{false, true, false, false, true, false}},
	} {
		for index, expected := range testCase.expected {
			assert.Equal(t, expected, testCase.pattern.IsGap(index), "%+v at %d", testCase.pattern, index)
		}
	}
}
