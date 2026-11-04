package state

import (
	"encoding/json"

	"github.com/bluemir/nemokan/internal/cursor"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

// Widgets. See ADR-0014.

// addWidgetCommand adds a widget: a click puts it in its default size, and a drag sets its
// size as the kind allows.
type addWidgetCommand struct {
	kind         shape.WidgetKind
	workingShape *shape.Widget
}

func (*addWidgetCommand) mouseCursor() (cursor.Cursor, bool) { return cursor.Crosshair, true }

func (c *addWidgetCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDown:
		content, ok := shape.NewWidgetContent(c.kind)
		if !ok {
			return commandDone
		}
		c.workingShape = shape.NewWidget(pointer.Point, content, "", m.workingParentGroup.ID())
		m.addShape(c.workingShape)
		m.selection.Clear()
		return commandWorking
	case MouseDrag:
		c.resize(m, pointer.MouseDownPoint, pointer.Point)
		return commandWorking
	case MouseUp:
		if pointer.Point != pointer.MouseDownPoint {
			c.resize(m, pointer.MouseDownPoint, pointer.Point)
		}
		m.addSelectedShape(c.workingShape)
		return commandDone
	default:
		return commandUnknown
	}
}

func (c *addWidgetCommand) resize(m *Manager, point1, point2 geo.Point) {
	if c.workingShape == nil {
		return
	}
	m.shapeManager.Execute(shape.ChangeBound{Target: c.workingShape, NewBound: rectByPoints(point1, point2)})
}

// editWidgetAt edits the field of target at point: a toggle is turned on or off, and a text
// is edited in the text editor.
func (m *Manager) editWidgetAt(target *shape.Widget, point geo.Point) {
	run, ok := target.FieldAt(point.Minus(target.Bound().Position))
	if !ok {
		return
	}
	if run.IsToggle() {
		if content, ok := shape.ToggledWidgetContent(target); ok {
			m.shapeManager.Execute(shape.ChangeWidgetContent{Target: target, NewContent: content})
		}
		return
	}
	m.editWidgetField(target, run)
}

// editFirstWidgetField edits the first text of target, for the Enter key.
func (m *Manager) editFirstWidgetField(target *shape.Widget) {
	for _, run := range target.Layout().Runs {
		if run.Field != nil && !run.IsToggle() {
			m.editWidgetField(target, run)
			return
		}
	}
}

func (m *Manager) editWidgetField(target *shape.Widget, run shape.WidgetRun) {
	field := *run.Field
	oldContent := target.Content()
	position := target.Bound().Position.Plus(geo.Point{Left: run.Column, Top: run.Row})
	m.enterEditingMode()
	m.showTextEditor(&TextEditing{
		Text:         run.Value,
		BoundPx:      m.boundPx(geo.Rect{Position: position, Size: geo.Size{Width: max(run.Width, minFieldEditorWidth), Height: 1}}),
		IsSingleLine: true,
		onChange: func(text string) {
			m.shapeManager.Execute(shape.ChangeWidgetContent{Target: target, NewContent: shape.WidgetContentWithText(target, field, text)})
		},
		onFinish: func(string) {
			if shape.SameWidgetContent(oldContent, target.Content()) {
				m.exitEditingModeSkippingHistory()
				return
			}
			m.exitEditingMode()
		},
	})
}

// minFieldEditorWidth keeps the text editor of a short field wide enough to type in.
const minFieldEditorWidth = 8

// WidgetTool is the widget section of the shape tool panel. Content is the JSON of the
// content, which JS shows as a form for the kind.
type WidgetTool struct {
	Kind    shape.WidgetKind `json:"kind"`
	Content json.RawMessage  `json:"content"`
}

func newWidgetTool(target *shape.Widget) *WidgetTool {
	content, err := json.Marshal(target.Content())
	if err != nil {
		panic(err)
	}
	return &WidgetTool{Kind: target.Content().Kind(), Content: content}
}

// setSelectedWidgetContent replaces the content of the selected widget with the JSON from the
// shape tool panel.
func (m *Manager) setSelectedWidgetContent(data json.RawMessage) {
	selected, _ := m.singleSelectedShape()
	target, ok := selected.(*shape.Widget)
	if !ok {
		return
	}
	content, err := shape.ParseWidgetContent(target.Content().Kind(), data)
	if err != nil {
		return
	}
	m.shapeManager.Execute(shape.ChangeWidgetContent{Target: target, NewContent: content})
	m.refreshInteractionBounds()
}
