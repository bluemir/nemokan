package state

import (
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// hoverShapeCache keeps the hover shape of each point, to avoid searching a point again
// while the mouse moves. A nil shape means nothing at the point.
type hoverShapeCache struct {
	pointToShape map[geo.Point]shape.Shape
	search       func(geo.Point) shape.Shape
}

// newLineConnectHoverCache finds the shape a line end connects to.
func newLineConnectHoverCache(m *Manager) *hoverShapeCache {
	return &hoverShapeCache{
		pointToShape: map[geo.Point]shape.Shape{},
		search: func(point geo.Point) shape.Shape {
			found, ok := shape.ConnectableShape(point, m.searcher.Shapes(point))
			if !ok {
				return nil
			}
			return found
		},
	}
}

// newHoverCache finds the top shape at a point.
func newHoverCache(m *Manager) *hoverShapeCache {
	return &hoverShapeCache{
		pointToShape: map[geo.Point]shape.Shape{},
		search: func(point geo.Point) shape.Shape {
			shapes := m.searcher.Shapes(point)
			if len(shapes) == 0 {
				return nil
			}
			return shapes[len(shapes)-1]
		},
	}
}

func (c *hoverShapeCache) hoverShape(point geo.Point) shape.Shape {
	if found, ok := c.pointToShape[point]; ok {
		return found
	}
	found := c.search(point)
	c.pointToShape[point] = found
	return found
}

func (c *hoverShapeCache) reset() {
	clear(c.pointToShape)
}
