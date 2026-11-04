package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/action"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func (s *testState) addWidget(kind shape.WidgetKind, from, to geo.Point) *shape.Widget {
	s.actions.SetRetainable(action.AddWidget(kind))
	if from == to {
		s.click(from, false)
	} else {
		s.drag(from, to)
	}
	items := s.ShapeManager().Root().Items()
	return items[len(items)-1].(*shape.Widget)
}

func TestAddWidget(t *testing.T) {
	state := newTestState(t)

	// A click puts the default size.
	button := state.addWidget("button", point(1, 1), point(1, 1))
	assert.Equal(t, geo.RectByLTWH(1, 1, 6, 1), button.Bound())
	assert.Equal(t, "[ OK ]", state.rows(geo.RectByLTWH(1, 1, 6, 1)))
	assert.Equal(t, []shape.Shape{button}, state.selected())
	assert.Equal(t, action.RetainableIdle, state.actions.Retainable.Value())

	// A drag sets the width of a button, but not its height.
	wide := state.addWidget("button", point(1, 5), point(10, 8))
	assert.Equal(t, geo.RectByLTWH(1, 5, 10, 1), wide.Bound())

	// A card takes the dragged size.
	card := state.addWidget("card", point(20, 1), point(29, 5))
	assert.Equal(t, geo.RectByLTRB(20, 1, 29, 5), card.Bound())
	assert.NotNil(t, state.ShapeTool().Rectangle)
	assert.Equal(t, shape.WidgetKind("card"), state.ShapeTool().Widget.Kind)
}

func TestWidget_DoubleClickTogglesMark(t *testing.T) {
	state := newTestState(t)
	checkbox := state.addWidget("checkbox", point(1, 1), point(1, 1))

	state.doubleClick(point(2, 1))
	assert.True(t, checkbox.Content().(*shape.CheckboxContent).IsChecked)
	assert.Equal(t, "[x] Checkbox", state.rows(geo.RectByLTWH(1, 1, 12, 1)))
}

func TestWidget_DoubleClickEditsField(t *testing.T) {
	state := newTestState(t)
	table := state.addWidget("table", point(0, 0), point(30, 4))

	// The second column name.
	state.doubleClick(point(13, 1))
	editing := state.TextEditing()
	assert.NotNil(t, editing)
	assert.Equal(t, "Email", editing.Text)
	assert.True(t, editing.IsSingleLine)

	state.UpdateEditingText("Mail")
	state.FinishTextEditing()
	assert.Equal(t, []string{"Name", "Mail", "Role"}, table.Content().(*shape.TableContent).Columns)
	assert.False(t, state.EditingMode.Value().IsEditing)
}

func TestWidget_ChangeContentFromShapeTool(t *testing.T) {
	state := newTestState(t)
	checkbox := state.addWidget("checkbox", point(1, 1), point(1, 1))

	state.actions.SetOneTime(action.ChangeWidgetContent{Content: json.RawMessage(`{"label": "Remember me", "isChecked": true}`)})
	assert.Equal(t, &shape.CheckboxContent{Label: "Remember me", IsChecked: true}, checkbox.Content())
	// The size follows the longer label.
	assert.Equal(t, 15, checkbox.Bound().Width())

	// A broken content is ignored.
	state.actions.SetOneTime(action.ChangeWidgetContent{Content: json.RawMessage(`{"label": 1}`)})
	assert.Equal(t, "Remember me", checkbox.Content().(*shape.CheckboxContent).Label)
}

func TestWidget_SaveAndOpen(t *testing.T) {
	source := newTestState(t)
	source.addWidget("tabs", point(1, 1), point(1, 1))
	source.actions.SetOneTime(action.ChangeWidgetContent{Content: json.RawMessage(`{"tabs": ["가", "B"], "active": 1}`)})

	data, err := json.Marshal(source.File("doc", geo.Point{}))
	assert.NoError(t, err)
	file, err := shape.ParseFile(data)
	assert.NoError(t, err)

	target := newTestState(t)
	target.OpenFile(file)
	widget := target.ShapeManager().Root().Items()[0].(*shape.Widget)
	assert.Equal(t, &shape.TabsContent{Tabs: []string{"가", "B"}, Active: 1}, widget.Content())
	assert.Equal(t, source.rows(geo.RectByLTWH(1, 1, 12, 2)), target.rows(geo.RectByLTWH(1, 1, 12, 2)))

	// An unknown kind cannot be opened.
	_, err = shape.ParseFile([]byte(`{"version": 1, "root": {"shapes": [{"type": "widget", "kind": "unknown", "content": {}}]}}`))
	assert.Error(t, err)
}
