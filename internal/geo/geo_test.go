package geo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverlappedRect(t *testing.T) {
	rect1 := RectByLTWH(0, 0, 6, 4)
	rect2 := RectByLTWH(2, 2, 4, 4)

	overlap, ok := rect1.OverlappedRect(rect2)
	assert.True(t, ok)
	assert.Equal(t, RectByLTWH(2, 2, 4, 2), overlap)

	overlap, ok = rect2.OverlappedRect(rect1)
	assert.True(t, ok)
	assert.Equal(t, RectByLTWH(2, 2, 4, 2), overlap)
}

func TestOverlappedRect_NotOverlapped(t *testing.T) {
	rect1 := RectByLTWH(0, 0, 1, 1)
	rect2 := RectByLTWH(2, 2, 1, 1)

	_, ok := rect1.OverlappedRect(rect2)
	assert.False(t, ok)
}

func TestJSON(t *testing.T) {
	data, err := json.Marshal(struct {
		Rect  Rect          `json:"rect"`
		Point DirectedPoint `json:"point"`
	}{
		Rect:  RectByLTWH(1, 2, 3, 4),
		Point: DirectedPoint{Direction: Vertical, Left: 5, Top: 6},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"rect":{"left":1,"top":2,"width":3,"height":4},"point":{"direction":"vertical","left":5,"top":6}}`, string(data))

	var decoded struct {
		Rect  Rect          `json:"rect"`
		Point DirectedPoint `json:"point"`
	}
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, RectByLTWH(1, 2, 3, 4), decoded.Rect)
	assert.Equal(t, DirectedPoint{Direction: Vertical, Left: 5, Top: 6}, decoded.Point)
}
