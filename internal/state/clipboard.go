package state

import (
	"github.com/bluemir/monosketch/internal/clipboard"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// TakeClipboardText returns the text to write to the clipboard, once. JS reads it after
// ChangeClipboard.
func (m *Manager) TakeClipboardText() (string, bool) {
	text := m.clipboardText
	m.clipboardText = nil
	if text == nil {
		return "", false
	}
	return *text, true
}

func (m *Manager) setClipboardText(text string) {
	m.clipboardText = &text
	m.notify(ChangeClipboard)
}

// Paste is called by JS with the pasted text.
func (m *Manager) Paste(text string) {
	if m.access == AccessView {
		return
	}
	object, ok := clipboard.Parse(text)
	if !ok || len(object.Shapes) == 0 {
		return
	}
	m.selection.Clear()
	bound := m.WindowBound()
	m.insertShapes(geo.Point{Left: bound.Left() + bound.Width()/5, Top: bound.Top() + bound.Height()/5}, object)
}

func (m *Manager) copySelectedShapes() {
	text, err := m.clipboardObject().Text()
	if err != nil {
		return
	}
	m.setClipboardText(text)
}

func (m *Manager) removeSelectedShapes() {
	for _, selected := range m.selection.SelectedShapes.Value() {
		m.removeShape(selected)
	}
	m.selection.Clear()
}

func (m *Manager) duplicateSelectedShapes() {
	selectedShapes := m.selection.SelectedShapes.Value()
	if len(selectedShapes) == 0 {
		return
	}
	object := m.clipboardObject()
	topLeft := selectedShapes[0].Bound().Position
	for _, selected := range selectedShapes {
		topLeft.Left = min(topLeft.Left, selected.Bound().Left())
		topLeft.Top = min(topLeft.Top, selected.Bound().Top())
	}
	m.selection.Clear()
	m.insertShapes(topLeft.Plus(geo.Point{Left: 1, Top: 1}), object)
}

// insertShapes adds the shapes in object with their top-left at position, and selects them.
func (m *Manager) insertShapes(position geo.Point, object clipboard.Object) {
	parentID := m.workingParentGroup.ID()
	originalIDToShape := map[string]shape.Shape{}
	shapes := make([]shape.Shape, 0, len(object.Shapes))
	for _, serializable := range object.Shapes {
		restored := shape.RestoreShape(serializable, parentID)
		originalIDToShape[serializable.OriginalID()] = restored
		shapes = append(shapes, restored)
	}

	topLeft := shapes[0].Bound().Position
	for _, restored := range shapes {
		topLeft.Left = min(topLeft.Left, restored.Bound().Left())
		topLeft.Top = min(topLeft.Top, restored.Bound().Top())
	}
	offset := topLeft.Minus(position)
	for _, restored := range shapes {
		bound := restored.Bound()
		restored.SetBound(geo.Rect{Position: bound.Position.Minus(offset), Size: bound.Size})
		m.addShape(restored)
		m.addSelectedShape(restored)
	}

	for _, connector := range object.Connectors {
		line, ok := originalIDToShape[connector.LineID].(*shape.Line)
		if !ok {
			continue
		}
		target, ok := originalIDToShape[connector.TargetID]
		if !ok {
			continue
		}
		m.shapeManager.ShapeConnector().AddConnector(line, connector.Anchor, target)
	}
}

func (m *Manager) clipboardObject() clipboard.Object {
	selectedShapes := m.selection.SelectedShapes.Value()
	object := clipboard.Object{Shapes: make(shape.SerializableShapes, 0, len(selectedShapes))}
	for _, selected := range selectedShapes {
		object.Shapes = append(object.Shapes, selected.Serializable(false))
		for _, connector := range m.shapeManager.ShapeConnector().Connectors(selected) {
			object.Connectors = append(object.Connectors, connector.Serializable(selected.ID()))
		}
	}
	return object
}
