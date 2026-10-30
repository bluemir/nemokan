package selection

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func TestManager_Selection(t *testing.T) {
	manager := New()
	shape1 := shape.NewRectangle(geo.Rect{}, "", "")
	shape2 := shape.NewRectangle(geo.Rect{}, "", "")

	manager.Add(shape1)
	manager.Add(shape2)
	manager.Add(shape1)
	assert.Equal(t, []shape.Shape{shape1, shape2}, manager.SelectedShapes.Value())

	manager.Toggle(shape1)
	assert.Equal(t, []shape.Shape{shape2}, manager.SelectedShapes.Value())
	manager.Toggle(shape1)
	assert.Equal(t, []shape.Shape{shape2, shape1}, manager.SelectedShapes.Value())

	manager.Clear()
	assert.Empty(t, manager.SelectedShapes.Value())
}

func TestManager_Selection_DoesNotChangeNotifiedSlice(t *testing.T) {
	manager := New()
	shape1 := shape.NewRectangle(geo.Rect{}, "", "")
	shape2 := shape.NewRectangle(geo.Rect{}, "", "")
	manager.Add(shape1)
	manager.Add(shape2)

	before := manager.SelectedShapes.Value()
	manager.Toggle(shape1)
	assert.Equal(t, []shape.Shape{shape1, shape2}, before)
}

func TestManager_FocusType(t *testing.T) {
	manager := New()
	focused := shape.NewRectangle(geo.Rect{}, "", "")
	other := shape.NewRectangle(geo.Rect{}, "", "")

	manager.SetFocusingShape(focused, FocusLineConnecting)
	focusType, ok := manager.FocusType(focused)
	assert.True(t, ok)
	assert.Equal(t, FocusLineConnecting, focusType)

	_, ok = manager.FocusType(other)
	assert.False(t, ok)

	manager.SetFocusingShape(nil, FocusSelectModeHover)
	assert.Nil(t, manager.FocusingShape.Value())
	_, ok = manager.FocusType(focused)
	assert.False(t, ok)
}
