package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/geo"
)

var connectorTestRect = geo.RectByLTRB(4, 20, 10, 30)

func TestAroundOf_NotAround(t *testing.T) {
	h := geo.Horizontal
	for _, point := range []geo.DirectedPoint{
		// Around left edge
		directed(h, 2, 25), directed(h, 6, 25),
		// Around right edge
		directed(h, 8, 25), directed(h, 12, 25),
		// Around top edge
		directed(h, 6, 18), directed(h, 6, 22),
		// Around bottom edge
		directed(h, 6, 28), directed(h, 6, 32),
		// On left edge, outside top
		directed(h, 3, 18), directed(h, 4, 18), directed(h, 5, 18),
		// On left edge, outside bottom
		directed(h, 3, 32), directed(h, 4, 32), directed(h, 5, 32),
		// On right edge, outside top
		directed(h, 9, 18), directed(h, 10, 18), directed(h, 11, 18),
		// On right edge, outside bottom
		directed(h, 9, 32), directed(h, 10, 32), directed(h, 11, 32),
		// On top edge, outside left
		directed(h, 2, 19), directed(h, 2, 20), directed(h, 2, 21),
		// On top edge, outside right
		directed(h, 12, 19), directed(h, 12, 20), directed(h, 12, 21),
		// On bottom edge, outside left
		directed(h, 2, 29), directed(h, 2, 30), directed(h, 2, 31),
		// On bottom edge, outside right
		directed(h, 12, 29), directed(h, 12, 30), directed(h, 12, 31),
	} {
		_, ok := aroundOf(point, connectorTestRect)
		assert.False(t, ok, point)
	}
}

func TestAroundOf(t *testing.T) {
	const (
		h = geo.Horizontal
		v = geo.Vertical
	)
	for _, testCase := range []struct {
		name     string
		expected map[geo.DirectedPoint]around
	}{
		{"left", map[geo.DirectedPoint]around{
			// No conflict
			directed(h, 3, 25): aroundLeft, directed(h, 4, 25): aroundLeft, directed(h, 5, 25): aroundLeft,
			directed(v, 3, 25): aroundLeft, directed(v, 4, 25): aroundLeft, directed(v, 5, 25): aroundLeft,
			// Horizontal on top edge goes left, vertical goes top
			directed(h, 3, 20): aroundLeft, directed(h, 4, 20): aroundLeft, directed(h, 5, 20): aroundLeft,
			directed(v, 3, 20): aroundTop, directed(v, 4, 20): aroundTop, directed(v, 5, 20): aroundTop,
			// Horizontal on bottom edge goes left, vertical goes bottom
			directed(h, 3, 30): aroundLeft, directed(h, 4, 30): aroundLeft, directed(h, 5, 30): aroundLeft,
			directed(v, 3, 30): aroundBottom, directed(v, 4, 30): aroundBottom, directed(v, 5, 30): aroundBottom,
		}},
		{"right", map[geo.DirectedPoint]around{
			directed(h, 9, 25): aroundRight, directed(h, 10, 25): aroundRight, directed(h, 11, 25): aroundRight,
			directed(v, 9, 25): aroundRight, directed(v, 10, 25): aroundRight, directed(v, 11, 25): aroundRight,
			directed(h, 9, 20): aroundRight, directed(h, 10, 20): aroundRight, directed(h, 11, 20): aroundRight,
			directed(v, 9, 20): aroundTop, directed(v, 10, 20): aroundTop, directed(v, 11, 20): aroundTop,
			directed(h, 9, 30): aroundRight, directed(h, 10, 30): aroundRight, directed(h, 11, 30): aroundRight,
			directed(v, 9, 30): aroundBottom, directed(v, 10, 30): aroundBottom, directed(v, 11, 30): aroundBottom,
		}},
		{"top", map[geo.DirectedPoint]around{
			directed(h, 6, 19): aroundTop, directed(h, 6, 20): aroundTop, directed(h, 6, 21): aroundTop,
			directed(v, 6, 19): aroundTop, directed(v, 6, 20): aroundTop, directed(v, 6, 21): aroundTop,
			directed(v, 4, 19): aroundTop, directed(v, 4, 20): aroundTop, directed(v, 4, 21): aroundTop,
			directed(h, 4, 19): aroundLeft, directed(h, 4, 20): aroundLeft, directed(h, 4, 21): aroundLeft,
			directed(v, 10, 19): aroundTop, directed(v, 10, 20): aroundTop, directed(v, 10, 21): aroundTop,
			directed(h, 10, 19): aroundRight, directed(h, 10, 20): aroundRight, directed(h, 10, 21): aroundRight,
		}},
		{"bottom", map[geo.DirectedPoint]around{
			directed(h, 6, 29): aroundBottom, directed(h, 6, 30): aroundBottom, directed(h, 6, 31): aroundBottom,
			directed(v, 6, 29): aroundBottom, directed(v, 6, 30): aroundBottom, directed(v, 6, 31): aroundBottom,
			directed(v, 4, 29): aroundBottom, directed(v, 4, 30): aroundBottom, directed(v, 4, 31): aroundBottom,
			directed(h, 4, 29): aroundLeft, directed(h, 4, 30): aroundLeft, directed(h, 4, 31): aroundLeft,
			directed(v, 10, 29): aroundBottom, directed(v, 10, 30): aroundBottom, directed(v, 10, 31): aroundBottom,
			directed(h, 10, 29): aroundRight, directed(h, 10, 30): aroundRight, directed(h, 10, 31): aroundRight,
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			for point, expected := range testCase.expected {
				got, ok := aroundOf(point, connectorTestRect)
				assert.True(t, ok, point)
				assert.Equal(t, expected, got, point)
			}
		})
	}
}

func TestShapeConnector_Serializable(t *testing.T) {
	rectangle := NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	line := NewLine(directed(geo.Horizontal, 5, 2), directed(geo.Horizontal, 20, 2), "", "")

	target := NewShapeConnector()
	target.AddConnector(line, AnchorStart, rectangle)
	assert.True(t, target.HasConnector(line, AnchorStart))
	assert.False(t, target.HasConnector(line, AnchorEnd))

	restored := RestoreShapeConnector(target.Serializable())
	assert.Equal(t, target.Connectors(rectangle), restored.Connectors(rectangle))

	restored.RemoveShape(line)
	assert.False(t, restored.HasConnector(line, AnchorStart))
}
