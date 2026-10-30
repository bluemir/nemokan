package state

import (
	"slices"

	"github.com/bluemir/monosketch/internal/board"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// SelectedShapesText returns the selected shapes drawn as text. It returns false if no shape
// is selected.
func (m *Manager) SelectedShapesText() (string, bool) {
	selected := m.selectedShapesInOrder()
	if len(selected) == 0 {
		return "", false
	}
	return m.exportText(selected), true
}

// ExportText returns the selected shapes drawn as text, or all shapes of the working group if
// none is selected.
func (m *Manager) ExportText() string {
	if selected := m.selectedShapesInOrder(); len(selected) > 0 {
		return m.exportText(selected)
	}
	if m.workingParentGroup.ItemCount() == 0 {
		return ""
	}
	return m.exportText([]shape.Shape{m.workingParentGroup})
}

// selectedShapesInOrder returns the selected shapes in the drawing order.
func (m *Manager) selectedShapesInOrder() []shape.Shape {
	selected := m.selection.SelectedShapes.Value()
	var shapes []shape.Shape
	for _, item := range m.workingParentGroup.Items() {
		if slices.Contains(selected, item) {
			shapes = append(shapes, item)
		}
	}
	return shapes
}

// exportText draws shapes onto a new board fitting them, without highlights.
func (m *Manager) exportText(shapes []shape.Shape) string {
	window := shapes[0].Bound()
	for _, target := range shapes {
		bound := target.Bound()
		window = geo.RectByLTRB(
			min(window.Left(), bound.Left()), min(window.Top(), bound.Top()),
			max(window.Right(), bound.Right()), max(window.Bottom(), bound.Bottom()),
		)
	}
	exportingBoard := board.New()
	exportingBoard.ClearAndSetWindow(window)
	m.drawShapesOnto(exportingBoard, shapes)
	return exportingBoard.StringInBound(window)
}

func (m *Manager) drawShapesOnto(target *board.Board, shapes []shape.Shape) {
	for _, item := range shapes {
		if group, ok := item.(*shape.Group); ok {
			m.drawShapesOnto(target, group.Items())
			continue
		}
		if drawn := m.bitmapManager.Bitmap(item); drawn != nil {
			target.Fill(item.Bound().Position, drawn, board.HighlightNone)
		}
	}
}

// File returns the current document as a file. JS keeps the name and the offset. See ADR-0010.
func (m *Manager) File(name string, offset geo.Point) shape.File {
	root := m.shapeManager.Root()
	return shape.NewFile(name, offset, root.Serializable(true).(*shape.SerializableGroup), m.shapeManager.ShapeConnector().Serializable())
}

// OpenFile replaces the shapes with file, keeping the document id of file.
func (m *Manager) OpenFile(file shape.File) {
	m.Restore("", &Backup{Root: file.Root, Connectors: file.Connectors})
}

// OpenFileAsCopy replaces the shapes with file as a new document, for keeping both when a
// document with the same id exists.
func (m *Manager) OpenFileAsCopy(file shape.File) {
	root := *file.Root
	root.IsIDTemporary = true
	m.Restore("", &Backup{Root: &root, Connectors: file.Connectors})
}
