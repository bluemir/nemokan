package state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

const cellSizePx = 10

type testState struct {
	*Manager
	t           *testing.T
	actions     *action.Manager
	changes     map[Change]int
	timeMillis  float64
	oneTimeSeen []action.OneTime
}

func newTestState(t *testing.T) *testState {
	actions := action.NewManager()
	state := &testState{t: t, actions: actions, changes: map[Change]int{}}
	state.Manager = New(shape.NewManager(), actions, func(change Change) { state.changes[change]++ })
	state.SetDrawingInfo(DrawingInfo{
		CellSizePx:   geo.SizeF{Width: cellSizePx, Height: cellSizePx},
		CanvasSizePx: geo.Size{Width: 400, Height: 300},
	})
	actions.OneTime.Observe(func(oneTime action.OneTime) {
		if _, ok := oneTime.(action.Idle); !ok {
			state.oneTimeSeen = append(state.oneTimeSeen, oneTime)
		}
	})
	return state
}

// event is a mouse event at the center of a board cell.
func (s *testState) event(left, top int, isWithShiftKey bool) MouseEvent {
	s.timeMillis += 1000
	return MouseEvent{
		OffsetPx:       geo.Point{Left: left*cellSizePx + cellSizePx/2, Top: top*cellSizePx + cellSizePx/2},
		IsWithShiftKey: isWithShiftKey,
		TimeMillis:     s.timeMillis,
	}
}

func (s *testState) drag(from, to geo.Point) {
	s.OnMouseMove(s.event(from.Left, from.Top, false))
	s.OnMouseDown(s.event(from.Left, from.Top, false))
	s.OnMouseMove(s.event(to.Left, to.Top, false))
	s.OnMouseUp(s.event(to.Left, to.Top, false))
	s.Board()
}

func (s *testState) click(point geo.Point, isWithShiftKey bool) {
	s.OnMouseMove(s.event(point.Left, point.Top, isWithShiftKey))
	s.OnMouseDown(s.event(point.Left, point.Top, isWithShiftKey))
	s.OnMouseUp(s.event(point.Left, point.Top, isWithShiftKey))
	s.Board()
}

func (s *testState) rows(bound geo.Rect) string {
	view := s.Board()
	var rows []string
	for row := bound.Top(); row <= bound.Bottom(); row++ {
		chars := []rune(view.Rows[row-view.Bound.Top()])
		chars = chars[bound.Left()-view.Bound.Left() : bound.Right()-view.Bound.Left()+1]
		rows = append(rows, strings.ReplaceAll(string(chars), string(WideCharTailInView), ""))
	}
	return strings.Join(rows, "\n")
}

func point(left, top int) geo.Point {
	return geo.Point{Left: left, Top: top}
}

func (s *testState) addRectangle(from, to geo.Point) shape.Shape {
	s.actions.SetRetainable(action.AddRectangle)
	s.drag(from, to)
	items := s.ShapeManager().Root().Items()
	return items[len(items)-1]
}

func TestAddRectangle(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))

	assert.Equal(t, geo.RectByLTRB(1, 1, 5, 3), rectangle.Bound())
	assert.Equal(t, []shape.Shape{rectangle}, state.Selection().SelectedShapes.Value())
	assert.Equal(t, action.RetainableIdle, state.actions.Retainable.Value())
	// The original leaves the editing mode on here.
	assert.False(t, state.EditingMode.Value().IsEditing)
	assert.Equal(t, ""+
		"┌───┐\n"+
		"│   │\n"+
		"└───┘", state.rows(geo.RectByLTRB(1, 1, 5, 3)))
	assert.Len(t, state.InteractionBounds(), 1)
	assert.Positive(t, state.changes[ChangeBoard])
}

func TestClickSelectsShape(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))

	state.click(point(20, 20), false)
	assert.Empty(t, state.Selection().SelectedShapes.Value())

	state.click(point(1, 2), false)
	assert.Equal(t, []shape.Shape{rectangle}, state.Selection().SelectedShapes.Value())
}

func TestDragSelectsShapesInZone(t *testing.T) {
	state := newTestState(t)
	rectangle1 := state.addRectangle(point(1, 1), point(3, 3))
	rectangle2 := state.addRectangle(point(10, 1), point(12, 3))
	state.click(point(30, 20), false)

	state.drag(point(0, 0), point(20, 5))
	assert.ElementsMatch(t, []shape.Shape{rectangle1, rectangle2}, state.Selection().SelectedShapes.Value())
	_, ok := state.SelectionBound()
	assert.False(t, ok)
}

func TestMoveSelectedShape(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))

	state.drag(point(3, 2), point(13, 6))
	assert.Equal(t, geo.RectByLTRB(11, 5, 15, 7), rectangle.Bound())
}

func TestScaleSelectedShape(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))

	// The bottom right interaction point is at (6.25, 4.25) cells.
	pointPx := geo.Point{Left: 62, Top: 42}
	state.OnMouseMove(MouseEvent{OffsetPx: pointPx})
	state.OnMouseDown(MouseEvent{OffsetPx: pointPx})
	state.OnMouseMove(MouseEvent{OffsetPx: geo.Point{Left: 102, Top: 82}})
	state.OnMouseUp(MouseEvent{OffsetPx: geo.Point{Left: 102, Top: 82}})

	assert.Equal(t, geo.RectByLTRB(1, 1, 9, 7), rectangle.Bound())
}

func TestAddLineConnectsToRectangle(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))
	// Mouse down on a selected shape moves it, so deselect first.
	state.click(point(20, 20), false)

	state.actions.SetRetainable(action.AddLine)
	state.drag(point(5, 2), point(12, 2))

	items := state.ShapeManager().Root().Items()
	line := items[len(items)-1].(*shape.Line)
	assert.True(t, state.ShapeManager().ShapeConnector().HasConnector(line, shape.AnchorStart))
	assert.Len(t, state.ShapeManager().ShapeConnector().Connectors(rectangle), 1)

	// The line follows the rectangle.
	state.click(point(1, 2), false)
	state.drag(point(1, 2), point(1, 12))
	assert.Equal(t, 5, line.StartPoint().Left)
	assert.Equal(t, 12, line.StartPoint().Top)
}

func TestAddFreeText(t *testing.T) {
	state := newTestState(t)
	state.actions.SetRetainable(action.AddText)
	state.click(point(2, 2), false)

	editing := state.TextEditing()
	if !assert.NotNil(t, editing) {
		return
	}
	assert.Equal(t, RectF{Left: 20, Top: 20, Width: 380, Height: 40}, editing.BoundPx)
	assert.True(t, state.EditingMode.Value().IsEditing)

	state.UpdateEditingText("hello\nhi")
	state.FinishTextEditing()

	text := state.ShapeManager().Root().Items()[0].(*shape.Text)
	assert.Equal(t, geo.RectByLTWH(2, 2, 5, 2), text.Bound())
	assert.False(t, text.IsTextEditing())
	assert.False(t, state.EditingMode.Value().IsEditing)
	assert.Equal(t, []shape.Shape{text}, state.Selection().SelectedShapes.Value())
	assert.Equal(t, "hello\nhi   ", state.rows(geo.RectByLTWH(2, 2, 5, 2)))
}

func TestAddFreeText_WideChars(t *testing.T) {
	state := newTestState(t)
	state.actions.SetRetainable(action.AddText)
	state.click(point(2, 2), false)
	state.UpdateEditingText("한글a")
	state.FinishTextEditing()

	text := state.ShapeManager().Root().Items()[0].(*shape.Text)
	assert.Equal(t, geo.RectByLTWH(2, 2, 5, 1), text.Bound())
	view := state.Board()
	row := []rune(view.Rows[2-view.Bound.Top()])
	assert.Equal(t, "한\x00글\x00a", string(row[2:7]))
}

func TestBoard_WideCharWithCoveredTailIsHidden(t *testing.T) {
	state := newTestState(t)
	state.Paste("한")
	text := state.ShapeManager().Root().Items()[0]
	// A one cell wide rectangle covers the tail.
	state.ShapeManager().Execute(shape.AddShape{Shape: shape.NewRectangle(geo.RectByLTWH(text.Bound().Left()+1, text.Bound().Top(), 1, 1), "", "")})

	view := state.Board()
	row := []rune(view.Rows[text.Bound().Top()-view.Bound.Top()])
	column := text.Bound().Left() - view.Bound.Left()
	assert.Equal(t, "□", string(row[column+1]))
	assert.Equal(t, " ", string(row[column]))
}

func TestAddFreeText_EmptyIsRemoved(t *testing.T) {
	state := newTestState(t)
	state.actions.SetRetainable(action.AddText)
	state.click(point(2, 2), false)
	state.FinishTextEditing()

	assert.Equal(t, 0, state.ShapeManager().Root().ItemCount())
	assert.NotNil(t, state.EditingMode.Value().SkippedVersion)
}

func TestDoubleClickEditsSelectedShape(t *testing.T) {
	state := newTestState(t)
	rectangle := state.addRectangle(point(1, 1), point(5, 3))

	state.click(point(1, 2), false)
	secondUp := state.event(1, 2, false)
	state.OnMouseDown(secondUp)
	// Within 500ms of the first mouse up.
	secondUp.TimeMillis -= 600
	state.OnMouseUp(secondUp)

	assert.Equal(t, []action.OneTime{action.EditSelectedShape{Shape: rectangle}}, state.oneTimeSeen)
}

func TestHoverShapeIsHighlighted(t *testing.T) {
	state := newTestState(t)
	state.addRectangle(point(1, 1), point(5, 3))
	state.click(point(20, 20), false)

	state.OnMouseMove(state.event(1, 2, false))
	focusing := state.Selection().FocusingShape.Value()
	assert.NotNil(t, focusing)
}
