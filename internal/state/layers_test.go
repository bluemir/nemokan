package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func (s *testState) doubleClick(target geo.Point) {
	s.click(target, false)
	// The second mouse up is within 500ms of the first one, with no move between them.
	secondUp := s.event(target.Left, target.Top, false)
	s.OnMouseDown(secondUp)
	secondUp.TimeMillis -= 600
	s.OnMouseUp(secondUp)
	s.Board()
}

func (s *testState) selected() []shape.Shape {
	return s.Selection().SelectedShapes.Value()
}

// addGroupOfTwo adds two rectangles and groups them.
func (s *testState) addGroupOfTwo() (*shape.Group, shape.Shape, shape.Shape) {
	first := s.addRectangle(point(1, 1), point(4, 3))
	second := s.addRectangle(point(10, 1), point(13, 3))
	s.actions.SetOneTime(action.SelectAllShapes{})
	s.actions.SetOneTime(action.GroupShapes{})
	group := s.ShapeManager().Root().Items()[0].(*shape.Group)
	return group, first, second
}

func TestGroupShapes_ClickSelectsGroup(t *testing.T) {
	state := newTestState(t)
	group, first, second := state.addGroupOfTwo()
	assert.Equal(t, []shape.Shape{first, second}, group.Items())
	assert.Equal(t, []shape.Shape{group}, state.selected())

	state.click(point(30, 20), false)
	assert.Empty(t, state.selected())
	state.click(point(1, 1), false)
	assert.Equal(t, []shape.Shape{group}, state.selected())
	// The shapes in a selected group are highlighted.
	assert.Equal(t, "selected", highlightName(state, point(10, 1)))
}

func highlightName(state *testState, target geo.Point) string {
	view := state.Board()
	index := (target.Top-view.Bound.Top())*view.Bound.Width() + target.Left - view.Bound.Left()
	if view.Highlights[index] == 0 {
		return "none"
	}
	return "selected"
}

func TestGroupShapes_DoubleClickEntersGroup(t *testing.T) {
	state := newTestState(t)
	group, _, second := state.addGroupOfTwo()

	state.doubleClick(point(10, 1))
	assert.Equal(t, group.ID(), state.Layers().WorkingGroupID)
	assert.Equal(t, []shape.Shape{second}, state.selected())

	// Clicking outside the group goes back to the root.
	state.click(point(30, 20), false)
	assert.Equal(t, "", state.Layers().WorkingGroupID)
	state.click(point(10, 1), false)
	assert.Equal(t, []shape.Shape{group}, state.selected())
}

func TestGroupShapes_MoveGroup(t *testing.T) {
	state := newTestState(t)
	_, first, second := state.addGroupOfTwo()

	state.actions.SetOneTime(action.MoveShapes{OffsetRow: 1, OffsetColumn: 2})
	assert.Equal(t, geo.RectByLTRB(3, 2, 6, 4), first.Bound())
	assert.Equal(t, geo.RectByLTRB(12, 2, 15, 4), second.Bound())

	state.actions.SetOneTime(action.ChangeShapeBound{NewLeft: new(0)})
	assert.Equal(t, geo.RectByLTRB(0, 2, 3, 4), first.Bound())
	assert.Equal(t, geo.RectByLTRB(9, 2, 12, 4), second.Bound())
}

func TestUngroupShapes(t *testing.T) {
	state := newTestState(t)
	_, first, second := state.addGroupOfTwo()

	state.actions.SetOneTime(action.UngroupShapes{})
	assert.Equal(t, []shape.Shape{first, second}, state.ShapeManager().Root().Items())
	assert.Equal(t, []shape.Shape{first, second}, state.selected())
}

func TestSetShapeHidden(t *testing.T) {
	state := newTestState(t)
	first := state.addRectangle(point(1, 1), point(4, 3))
	second := state.addRectangle(point(10, 1), point(13, 3))
	state.actions.SetOneTime(action.SelectShape{ShapeID: first.ID()})

	state.actions.SetOneTime(action.SetShapeHidden{ShapeID: first.ID(), IsHidden: true})
	assert.True(t, first.IsHidden())
	assert.Empty(t, state.selected())
	assert.Equal(t, "    ", state.rows(geo.RectByLTRB(1, 1, 4, 1)))
	// A hidden shape is not found by the mouse, exported or selected by select all.
	state.click(point(1, 1), false)
	assert.Empty(t, state.selected())
	assert.Equal(t, "┌──┐\n│  │\n└──┘", state.ExportText())
	state.actions.SetOneTime(action.SelectAllShapes{})
	assert.Equal(t, []shape.Shape{second}, state.selected())
	state.actions.SetOneTime(action.SelectShape{ShapeID: first.ID()})
	assert.Equal(t, []shape.Shape{second}, state.selected())

	state.actions.SetOneTime(action.SetShapeHidden{ShapeID: first.ID(), IsHidden: false})
	assert.Equal(t, "┌──┐", state.rows(geo.RectByLTRB(1, 1, 4, 1)))
}

func TestSetShapeLocked(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(4, 3))
	state.actions.SetOneTime(action.SetShapeLocked{ShapeID: rectangle.ID(), IsLocked: true})
	state.actions.SetOneTime(action.DeselectShapes{})

	// The canvas cannot select a locked shape.
	state.click(point(1, 1), false)
	assert.Empty(t, state.selected())

	// The layers can, but the canvas does not move it.
	state.actions.SetOneTime(action.SelectShape{ShapeID: rectangle.ID()})
	assert.Equal(t, []shape.Shape{rectangle}, state.selected())
	assert.Empty(t, state.InteractionBounds()[0].Points())
	state.drag(point(1, 1), point(5, 5))
	state.actions.SetOneTime(action.MoveShapes{OffsetRow: 1})
	assert.Equal(t, geo.RectByLTRB(1, 1, 4, 3), rectangle.Bound())
}

func TestRenameShape(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(4, 3))
	state.actions.SetOneTime(action.RenameShape{ShapeID: rectangle.ID(), Name: "Header"})

	layers := state.Layers()
	assert.Equal(t, "Header", layers.Items[0].Name)
	assert.Equal(t, "rectangle", layers.Items[0].Type)
	assert.True(t, layers.Items[0].IsSelected)
}

func TestMoveShapeInLayers(t *testing.T) {
	state := newTestState(t)
	group, first, second := state.addGroupOfTwo()
	third := state.addRectangle(point(20, 1), point(23, 3))

	// Into the group, above first.
	state.actions.SetOneTime(action.MoveShapeInLayers{ShapeID: third.ID(), ParentID: group.ID(), BelowID: first.ID()})
	assert.Equal(t, []shape.Shape{first, third, second}, group.Items())
	assert.Equal(t, group.ID(), state.Layers().WorkingGroupID)

	// A group cannot move into itself.
	state.actions.SetOneTime(action.MoveShapeInLayers{ShapeID: group.ID(), ParentID: group.ID()})
	assert.Equal(t, []shape.Shape{group}, state.ShapeManager().Root().Items())

	// Out of the group to the bottom of the root. A group left with one item is ungrouped.
	state.actions.SetOneTime(action.MoveShapeInLayers{ShapeID: first.ID()})
	state.actions.SetOneTime(action.MoveShapeInLayers{ShapeID: third.ID()})
	assert.Equal(t, []shape.Shape{third, first, second}, state.ShapeManager().Root().Items())
	_, ok := state.ShapeManager().Shape(group.ID())
	assert.False(t, ok)
	assert.Equal(t, "", state.Layers().WorkingGroupID)
}
