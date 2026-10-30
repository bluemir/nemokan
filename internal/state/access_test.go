package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func TestAccessView_SelectsAndCopiesOnly(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))
	state.actions.SetOneTime(action.DeselectShapes{})
	version := state.ShapeManager().Version.Value()

	state.SetAccess(AccessView)

	// Adding shapes is not allowed.
	state.actions.SetRetainable(action.AddRectangle)
	assert.Equal(t, action.RetainableIdle, state.actions.Retainable.Value())
	state.drag(point(10, 10), point(15, 15))
	assert.Len(t, state.ShapeManager().Root().Items(), 1)

	// Selecting is allowed, without the handles for changing shapes.
	state.click(point(1, 1), false)
	assert.Equal(t, []shape.Shape{rectangle}, state.Selection().SelectedShapes.Value())
	assert.Empty(t, state.InteractionBounds())
	assert.Equal(t, ShapeTool{}, state.ShapeTool())

	// Dragging the selected shape does not move it.
	state.drag(point(2, 2), point(8, 8))
	assert.Equal(t, geo.RectByLTRB(1, 1, 5, 3), rectangle.Bound())

	for _, oneTime := range []action.OneTime{
		action.MoveShapes{OffsetRow: 1, OffsetColumn: 1},
		action.ChangeShapeBound{NewWidth: pointer(10)},
		action.ChangeShapeFillExtra{IsEnabled: pointer(true)},
		action.ReorderShape{OrderType: shape.OrderFront},
		action.Duplicate{},
		action.Cut{},
		action.DeleteSelectedShapes{},
		action.EditSelectedShapes{},
		action.Undo{},
	} {
		state.Selection().Clear()
		state.Selection().Add(rectangle)
		state.actions.SetOneTime(oneTime)
	}
	state.Paste("pasted text")
	assert.Equal(t, version, state.ShapeManager().Version.Value())
	assert.Nil(t, state.TextEditing())

	// Copying is allowed.
	state.actions.SetOneTime(action.SelectAllShapes{})
	state.actions.SetOneTime(action.CopyText{})
	text, ok := state.TakeClipboardText()
	assert.True(t, ok)
	assert.Contains(t, text, "┌───┐")
}

func TestAccessEdit_ShowsHandlesAgain(t *testing.T) {
	state := newTestState(t)
	state.addRectangle(point(1, 1), point(5, 3))

	state.SetAccess(AccessView)
	assert.Empty(t, state.InteractionBounds())

	state.SetAccess(AccessEdit)
	assert.Len(t, state.InteractionBounds(), 1)
}
