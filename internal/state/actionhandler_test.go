package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func pointer[T any](value T) *T {
	return &value
}

func TestAction_SelectDeleteAndDeselect(t *testing.T) {
	state := newTestState(t)
	rectangle1 := state.addRectangle(point(1, 1), point(3, 3))
	rectangle2 := state.addRectangle(point(10, 1), point(12, 3))

	state.actions.SetOneTime(action.SelectAllShapes{})
	assert.ElementsMatch(t, []shape.Shape{rectangle1, rectangle2}, state.Selection().SelectedShapes.Value())

	state.actions.SetOneTime(action.DeselectShapes{})
	assert.Empty(t, state.Selection().SelectedShapes.Value())

	state.Selection().Add(rectangle1)
	state.actions.SetOneTime(action.DeleteSelectedShapes{})
	assert.Equal(t, []shape.Shape{rectangle2}, state.ShapeManager().Root().Items())
	assert.Empty(t, state.Selection().SelectedShapes.Value())
}

func TestAction_MoveAndChangeBound(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(3, 3))

	state.actions.SetOneTime(action.MoveShapes{OffsetRow: 1, OffsetColumn: -1})
	assert.Equal(t, geo.RectByLTWH(0, 2, 3, 3), rectangle.Bound())

	state.actions.SetOneTime(action.ChangeShapeBound{NewWidth: pointer(7)})
	assert.Equal(t, geo.RectByLTWH(0, 2, 7, 3), rectangle.Bound())
}

func TestAction_ChangeRectangleExtra(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(3, 3)).(*shape.Text)

	state.actions.SetOneTime(action.ChangeShapeFillExtra{IsEnabled: pointer(true), NewFillStyleID: pointer("F2")})
	state.actions.SetOneTime(action.ChangeShapeBorderExtra{NewBorderStyleID: pointer("S2")})
	state.actions.SetOneTime(action.ChangeShapeBorderDashPatternExtra{Gap: pointer(2)})
	state.actions.SetOneTime(action.ChangeShapeBorderCornerExtra{IsRoundedCorner: true})

	extra := rectangle.Extra().BoundExtra
	assert.True(t, extra.IsFillEnabled)
	assert.Equal(t, "F2", extra.UserSelectedFillStyle.ID)
	assert.Equal(t, "S2", extra.UserSelectedBorderStyle.ID)
	assert.Equal(t, shape.StraightStrokeDashPattern{Dash: 1, Gap: 2, Offset: 0}, extra.DashPattern)
	assert.Equal(t, shape.CornerRounded, extra.Corner)
}

func TestAction_ChangeLineExtra(t *testing.T) {
	state := newTestState(t)
	state.actions.SetRetainable(action.AddLine)
	state.drag(point(1, 1), point(8, 1))
	line := state.ShapeManager().Root().Items()[0].(*shape.Line)

	state.actions.SetOneTime(action.ChangeLineStrokeExtra{NewStrokeStyleID: pointer("S3")})
	state.actions.SetOneTime(action.ChangeLineStrokeDashPatternExtra{Dash: pointer(3)})
	state.actions.SetOneTime(action.ChangeLineStrokeCornerExtra{IsRoundedCorner: true})
	state.actions.SetOneTime(action.ChangeLineStartAnchorExtra{IsEnabled: pointer(true), NewHeadID: pointer("A2")})
	state.actions.SetOneTime(action.ChangeLineEndAnchorExtra{IsEnabled: pointer(true), NewHeadID: pointer("A3")})

	extra := line.Extra()
	assert.Equal(t, "S3", extra.UserSelectedStrokeStyle.ID)
	assert.Equal(t, 3, extra.DashPattern.Dash)
	assert.True(t, extra.IsRoundedCorner)
	assert.Equal(t, "A2", extra.StartAnchor().ID)
	assert.Equal(t, "A3", extra.EndAnchor().ID)
}

func TestAction_TextAlignmentWithoutSelectionChangesDefault(t *testing.T) {
	defaultTextAlign := shape.DefaultTextAlign()
	t.Cleanup(func() {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{TextHorizontalAlign: &defaultTextAlign.Horizontal, TextVerticalAlign: &defaultTextAlign.Vertical})
	})

	state := newTestState(t)
	state.actions.SetOneTime(action.TextAlignment{NewHorizontalAlign: pointer(shape.AlignRight)})
	assert.Equal(t, shape.AlignRight, shape.DefaultTextAlign().Horizontal)
}

func TestAction_ReorderShape(t *testing.T) {
	state := newTestState(t)
	rectangle1 := state.addRectangle(point(1, 1), point(3, 3))
	rectangle2 := state.addRectangle(point(10, 1), point(12, 3))

	state.actions.SetOneTime(action.ReorderShape{OrderType: shape.OrderBack})
	assert.Equal(t, []shape.Shape{rectangle2, rectangle1}, state.ShapeManager().Root().Items())
}

func TestAction_EditSelectedText(t *testing.T) {
	state := newTestState(t)
	state.actions.SetRetainable(action.AddText)
	state.drag(point(1, 1), point(8, 4))
	state.UpdateEditingText("hi")
	state.FinishTextEditing()
	text := state.ShapeManager().Root().Items()[0].(*shape.Text)

	state.actions.SetOneTime(action.EditSelectedShapes{})
	assert.True(t, text.IsTextEditing())
	assert.Equal(t, "hi", state.TextEditing().Text)

	state.UpdateEditingText("hello")
	state.FinishTextEditing()
	assert.Equal(t, "hello", text.Text())
	assert.False(t, text.IsTextEditing())
	assert.Nil(t, state.EditingMode.Value().SkippedVersion)
}

func TestAction_CopyAndPaste(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))
	state.click(point(30, 20), false)
	state.actions.SetRetainable(action.AddLine)
	state.drag(point(5, 2), point(12, 2))
	state.Selection().Add(rectangle)

	state.actions.SetOneTime(action.Copy{})
	text, ok := state.TakeClipboardText()
	assert.True(t, ok)
	_, ok = state.TakeClipboardText()
	assert.False(t, ok)

	state.Paste(text)
	items := state.ShapeManager().Root().Items()
	assert.Len(t, items, 4)
	pastedLine := items[2].(*shape.Line)
	pastedRectangle := items[3]
	assert.Equal(t, []shape.Shape{pastedLine, pastedRectangle}, state.Selection().SelectedShapes.Value())
	assert.NotEqual(t, rectangle.ID(), pastedRectangle.ID())
	// The window is 40x30 cells, so the pasted shapes start at (8, 6).
	assert.Equal(t, point(8, 6), pastedRectangle.Bound().Position)
	assert.True(t, state.ShapeManager().ShapeConnector().HasConnector(pastedLine, shape.AnchorStart))
}

func TestAction_PasteText(t *testing.T) {
	state := newTestState(t)
	state.Paste("ab c")

	text := state.ShapeManager().Root().Items()[0].(*shape.Text)
	assert.Equal(t, geo.RectByLTWH(8, 6, 4, 1), text.Bound())
	assert.Equal(t, "ab c", state.rows(geo.RectByLTWH(8, 6, 4, 1)))
}

func TestAction_DuplicateAndCut(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))

	state.actions.SetOneTime(action.Duplicate{})
	items := state.ShapeManager().Root().Items()
	assert.Len(t, items, 2)
	assert.Equal(t, geo.RectByLTRB(2, 2, 6, 4), items[1].Bound())

	state.actions.SetOneTime(action.Cut{})
	assert.Equal(t, []shape.Shape{rectangle}, state.ShapeManager().Root().Items())
	_, ok := state.TakeClipboardText()
	assert.True(t, ok)
}
