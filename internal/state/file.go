package state

import (
	"slices"

	"github.com/bluemir/monosketch/internal/board"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// SelectedShapesText returns the selected shapes drawn as text, without the hidden ones. It
// returns false if no shape is selected or all of them are hidden.
func (m *Manager) SelectedShapesText() (string, bool) {
	selected := m.visibleLeafShapes(slices.DeleteFunc(m.selectedShapesInOrder(), m.isHiddenInGroups))
	if len(selected) == 0 {
		return "", false
	}
	return m.exportText(selected), true
}

// ExportText returns the shapes of exportingShapes drawn as text.
func (m *Manager) ExportText() string {
	shapes := m.exportingShapes()
	if len(shapes) == 0 {
		return ""
	}
	return m.exportText(shapes)
}

// ExportRows returns the same shapes as ExportText, one string for each row and one char for
// each cell, as BoardView.Rows. It is for drawing them as an image.
func (m *Manager) ExportRows() []string {
	shapes := m.exportingShapes()
	if len(shapes) == 0 {
		return nil
	}
	exportingBoard, window := m.exportBoard(shapes)
	rows := make([]string, 0, window.Height())
	chars := make([]rune, window.Width())
	for row := window.Top(); row <= window.Bottom(); row++ {
		for column := window.Left(); column <= window.Right(); column++ {
			chars[column-window.Left()] = viewChar(exportingBoard.Get(column, row), exportingBoard.Get(column+1, row))
		}
		rows = append(rows, string(chars))
	}
	return rows
}

// exportingShapes returns the shapes to export in the drawing order: the selected ones, or
// all of the document if none is selected. Groups are replaced by the shapes in them, and
// hidden shapes are left out (ADR-0013). It returns nil if there is nothing to export.
func (m *Manager) exportingShapes() []shape.Shape {
	if selected := m.selectedShapesInOrder(); len(selected) > 0 {
		return m.visibleLeafShapes(slices.DeleteFunc(selected, m.isHiddenInGroups))
	}
	return m.visibleLeafShapes(m.shapeManager.Root().Items())
}

// visibleLeafShapes returns shapes with each group replaced by the shapes in it, leaving out
// the hidden ones.
func (m *Manager) visibleLeafShapes(shapes []shape.Shape) []shape.Shape {
	var result []shape.Shape
	for _, target := range shapes {
		if target.IsHidden() {
			continue
		}
		if group, ok := target.(*shape.Group); ok {
			result = append(result, m.visibleLeafShapes(group.Items())...)
			continue
		}
		result = append(result, target)
	}
	return result
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
	exportingBoard, window := m.exportBoard(shapes)
	return exportingBoard.StringInBound(window)
}

// exportBoard draws shapes onto a new board, and returns it with the bound fitting them.
func (m *Manager) exportBoard(shapes []shape.Shape) (*board.Board, geo.Rect) {
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
	return exportingBoard, window
}

// drawShapesOnto draws shapes, which are not groups, onto target.
func (m *Manager) drawShapesOnto(target *board.Board, shapes []shape.Shape) {
	for _, item := range shapes {
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
