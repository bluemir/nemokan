package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/action"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func TestShapeTool_NothingSelected(t *testing.T) {
	state := newTestState(t)

	assert.Equal(t, ShapeTool{}, state.ShapeTool())
}

func TestShapeTool_SelectedRectangle(t *testing.T) {
	state := newTestState(t)
	state.addRectangle(point(1, 1), point(3, 3))
	state.actions.SetOneTime(action.ChangeShapeFillExtra{IsEnabled: pointer(true), NewFillStyleID: pointer("F2")})
	state.changes = map[Change]int{}
	state.actions.SetOneTime(action.ChangeShapeBorderCornerExtra{IsRoundedCorner: true})

	tool := state.ShapeTool()
	bound := geo.RectByLTWH(1, 1, 3, 3)
	assert.Equal(t, &bound, tool.Bound)
	assert.True(t, tool.IsResizeable)
	assert.Equal(t, &RectangleAppearance{
		Fill:            StyleSelection{IsEnabled: true, SelectedID: "F2"},
		Border:          StyleSelection{IsEnabled: true, SelectedID: "S1"},
		DashPattern:     shape.SolidDashPattern,
		IsRoundedCorner: pointer(true),
	}, tool.Rectangle)
	assert.Nil(t, tool.Line)
	// A rectangle is a text which cannot be edited, so it has no text alignment.
	assert.Nil(t, tool.TextAlign)
	assert.Positive(t, state.changes[ChangeShapeTool])
}

func TestShapeTool_BorderWithoutRoundedCorner(t *testing.T) {
	state := newTestState(t)
	state.addRectangle(point(1, 1), point(3, 3))

	state.actions.SetOneTime(action.ChangeShapeBorderExtra{NewBorderStyleID: pointer("S2")})
	assert.Nil(t, state.ShapeTool().Rectangle.IsRoundedCorner)

	state.actions.SetOneTime(action.ChangeShapeBorderExtra{NewBorderStyleID: pointer("S1"), IsEnabled: pointer(false)})
	assert.Nil(t, state.ShapeTool().Rectangle.IsRoundedCorner)
}

func TestShapeTool_DefaultsWhileAdding(t *testing.T) {
	state := newTestState(t)

	state.actions.SetRetainable(action.AddText)
	tool := state.ShapeTool()
	assert.Nil(t, tool.Bound)
	assert.NotNil(t, tool.Rectangle)
	assert.Nil(t, tool.Line)
	assert.Equal(t, shape.DefaultTextAlign(), *tool.TextAlign)

	state.actions.SetRetainable(action.AddLine)
	tool = state.ShapeTool()
	assert.Nil(t, tool.Rectangle)
	assert.Equal(t, StyleSelection{IsEnabled: true, SelectedID: "S1"}, tool.Line.Stroke)
	assert.Nil(t, tool.TextAlign)
}
