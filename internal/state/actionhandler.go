package state

import (
	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// handleOneTimeAction is the Go port of the original OneTimeActionHandler.
func (m *Manager) handleOneTimeAction(oneTime action.OneTime) {
	if !m.isAllowed(oneTime) {
		return
	}
	switch oneTime := oneTime.(type) {
	case action.SelectAllShapes:
		m.selectAllShapes()
	case action.DeselectShapes:
		m.selection.Clear()
	case action.DeleteSelectedShapes:
		m.deleteSelectedShapes()
	case action.EditSelectedShapes:
		if selected, ok := m.singleSelectedShape(); ok {
			m.editShape(selected)
		}
	case action.EditSelectedShape:
		m.editShape(oneTime.Shape)
	case action.TextAlignment:
		m.setTextAlignment(oneTime)
	case action.MoveShapes:
		m.moveSelectedShapes(geo.Point{Left: oneTime.OffsetColumn, Top: oneTime.OffsetRow})
	case action.ChangeShapeBound:
		m.setSelectedShapeBound(oneTime)
	case action.ChangeShapeFillExtra:
		m.setSelectedShapeFillExtra(oneTime)
	case action.ChangeShapeBorderExtra:
		m.setSelectedShapeBorderExtra(oneTime)
	case action.ChangeShapeBorderDashPatternExtra:
		m.setSelectedShapeBorderDashPattern(oneTime.Dash, oneTime.Gap, oneTime.Offset)
	case action.ChangeShapeBorderCornerExtra:
		m.setSelectedShapeBorderCorner(oneTime.IsRoundedCorner)
	case action.ChangeLineStrokeExtra:
		m.setSelectedLineStrokeExtra(oneTime)
	case action.ChangeLineStrokeDashPatternExtra:
		m.setSelectedLineDashPattern(oneTime.Dash, oneTime.Gap, oneTime.Offset)
	case action.ChangeLineStrokeCornerExtra:
		m.setSelectedLineCorner(oneTime.IsRoundedCorner)
	case action.ChangeLineStartAnchorExtra:
		m.setSelectedLineStartAnchor(oneTime.IsEnabled, oneTime.NewHeadID)
	case action.ChangeLineEndAnchorExtra:
		m.setSelectedLineEndAnchor(oneTime.IsEnabled, oneTime.NewHeadID)
	case action.ReorderShape:
		if selected, ok := m.singleSelectedShape(); ok {
			m.shapeManager.Execute(shape.ChangeOrder{Shape: selected, OrderType: oneTime.OrderType})
		}
	case action.Copy:
		m.copySelectedShapes()
	case action.Cut:
		m.copySelectedShapes()
		m.removeSelectedShapes()
	case action.Duplicate:
		m.duplicateSelectedShapes()
	case action.CopyText:
		if text, ok := m.SelectedShapesText(); ok {
			m.setClipboardText(text)
		}
	case action.Undo:
		m.undo()
	case action.Redo:
		m.redo()
	}
}

func (m *Manager) singleSelectedShape() (shape.Shape, bool) {
	selected := m.selection.SelectedShapes.Value()
	if len(selected) != 1 {
		return nil, false
	}
	return selected[0], true
}

func (m *Manager) singleSelectedLine() (*shape.Line, bool) {
	selected, ok := m.singleSelectedShape()
	if !ok {
		return nil, false
	}
	line, ok := selected.(*shape.Line)
	return line, ok
}

func (m *Manager) deleteSelectedShapes() {
	for _, selected := range m.selection.SelectedShapes.Value() {
		m.removeShape(selected)
		m.shapeManager.ShapeConnector().RemoveShape(selected)
	}
	m.selection.Clear()
}

func (m *Manager) editShape(target shape.Shape) {
	text, ok := target.(*shape.Text)
	if !ok {
		return
	}
	m.enterEditingMode()
	oldText := text.Text()
	m.shapeManager.Execute(shape.MakeTextEditable{Target: text})
	m.shapeManager.Execute(shape.UpdateTextEditingMode{Target: text, IsEditing: true})
	m.startTextEditing(text, false, func(finalText string) {
		m.shapeManager.Execute(shape.UpdateTextEditingMode{Target: text, IsEditing: false})
		if finalText != oldText {
			m.exitEditingMode()
		} else {
			m.exitEditingModeSkippingHistory()
		}
	})
}

func (m *Manager) setTextAlignment(alignment action.TextAlignment) {
	selected, _ := m.singleSelectedShape()
	text, ok := selected.(*shape.Text)
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{
			TextHorizontalAlign: alignment.NewHorizontalAlign,
			TextVerticalAlign:   alignment.NewVerticalAlign,
		})
		return
	}
	extra := text.Extra()
	extra.TextAlign.Horizontal = valueOr(alignment.NewHorizontalAlign, extra.TextAlign.Horizontal)
	extra.TextAlign.Vertical = valueOr(alignment.NewVerticalAlign, extra.TextAlign.Vertical)
	m.shapeManager.Execute(shape.ChangeExtra{Target: text, NewExtra: extra})
}

func (m *Manager) moveSelectedShapes(offset geo.Point) {
	m.moveShapes(m.selection.SelectedShapes.Value(), true, func(target shape.Shape) (geo.Point, bool) {
		return target.Bound().Position.Plus(offset), true
	})
	m.refreshInteractionBounds()
}

func (m *Manager) setSelectedShapeBound(change action.ChangeShapeBound) {
	selected, ok := m.singleSelectedShape()
	if !ok {
		return
	}
	currentBound := selected.Bound()
	newBound := geo.RectByLTWH(
		valueOr(change.NewLeft, currentBound.Left()),
		valueOr(change.NewTop, currentBound.Top()),
		valueOr(change.NewWidth, currentBound.Width()),
		valueOr(change.NewHeight, currentBound.Height()),
	)
	m.updateConnectors(selected, newBound, true)
	m.shapeManager.Execute(shape.ChangeBound{Target: selected, NewBound: newBound})
	m.refreshInteractionBounds()
}

// changeRectangleExtra changes the rectangle extra of target, which is the bound of a text.
// It returns false if target has no rectangle extra.
func (m *Manager) changeRectangleExtra(target shape.Shape, change func(extra *shape.RectangleExtra)) bool {
	switch target := target.(type) {
	case *shape.Rectangle:
		extra := target.Extra()
		change(&extra)
		m.shapeManager.Execute(shape.ChangeExtra{Target: target, NewExtra: extra})
	case *shape.Text:
		extra := target.Extra()
		change(&extra.BoundExtra)
		m.shapeManager.Execute(shape.ChangeExtra{Target: target, NewExtra: extra})
	default:
		return false
	}
	return true
}

// Without one selected shape, the style changes become the default for new shapes.

func (m *Manager) setSelectedShapeFillExtra(change action.ChangeShapeFillExtra) {
	selected, ok := m.singleSelectedShape()
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsFillEnabled: change.IsEnabled, FillStyleID: change.NewFillStyleID})
		return
	}
	m.changeRectangleExtra(selected, func(extra *shape.RectangleExtra) {
		extra.IsFillEnabled = valueOr(change.IsEnabled, extra.IsFillEnabled)
		extra.UserSelectedFillStyle = shape.RectangleFillStyleOr(valueOr(change.NewFillStyleID, ""), extra.UserSelectedFillStyle)
	})
}

func (m *Manager) setSelectedShapeBorderExtra(change action.ChangeShapeBorderExtra) {
	selected, ok := m.singleSelectedShape()
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsBorderEnabled: change.IsEnabled, BorderStyleID: change.NewBorderStyleID})
		return
	}
	m.changeRectangleExtra(selected, func(extra *shape.RectangleExtra) {
		extra.IsBorderEnabled = valueOr(change.IsEnabled, extra.IsBorderEnabled)
		extra.UserSelectedBorderStyle = shape.StrokeStyleOr(valueOr(change.NewBorderStyleID, ""), extra.UserSelectedBorderStyle)
	})
}

func (m *Manager) setSelectedShapeBorderDashPattern(dash, gap, offset *int) {
	selected, ok := m.singleSelectedShape()
	if !ok {
		pattern := changeDashPattern(shape.DefaultRectangleExtra().DashPattern, dash, gap, offset)
		shape.SetDefaultValues(shape.DefaultValuesUpdate{BorderDashPattern: &pattern})
		return
	}
	m.changeRectangleExtra(selected, func(extra *shape.RectangleExtra) {
		extra.DashPattern = changeDashPattern(extra.DashPattern, dash, gap, offset)
	})
}

func (m *Manager) setSelectedShapeBorderCorner(isRoundedCorner bool) {
	corner := shape.CornerNotRounded
	if isRoundedCorner {
		corner = shape.CornerRounded
	}
	selected, _ := m.singleSelectedShape()
	isChanged := m.changeRectangleExtra(selected, func(extra *shape.RectangleExtra) { extra.Corner = corner })
	if !isChanged {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsBorderRoundedCorner: &isRoundedCorner})
	}
}

func (m *Manager) setSelectedLineStrokeExtra(change action.ChangeLineStrokeExtra) {
	line, ok := m.singleSelectedLine()
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsLineStrokeEnabled: change.IsEnabled, LineStrokeStyleID: change.NewStrokeStyleID})
		return
	}
	extra := line.Extra()
	extra.IsStrokeEnabled = valueOr(change.IsEnabled, extra.IsStrokeEnabled)
	extra.UserSelectedStrokeStyle = shape.StrokeStyleOr(valueOr(change.NewStrokeStyleID, ""), extra.UserSelectedStrokeStyle)
	m.shapeManager.Execute(shape.ChangeExtra{Target: line, NewExtra: extra})
}

func (m *Manager) setSelectedLineDashPattern(dash, gap, offset *int) {
	line, ok := m.singleSelectedLine()
	if !ok {
		pattern := changeDashPattern(shape.DefaultLineExtra().DashPattern, dash, gap, offset)
		shape.SetDefaultValues(shape.DefaultValuesUpdate{LineDashPattern: &pattern})
		return
	}
	extra := line.Extra()
	extra.DashPattern = changeDashPattern(extra.DashPattern, dash, gap, offset)
	m.shapeManager.Execute(shape.ChangeExtra{Target: line, NewExtra: extra})
}

func (m *Manager) setSelectedLineCorner(isRoundedCorner bool) {
	line, ok := m.singleSelectedLine()
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsLineStrokeRoundedCorner: &isRoundedCorner})
		return
	}
	extra := line.Extra()
	extra.IsRoundedCorner = isRoundedCorner
	m.shapeManager.Execute(shape.ChangeExtra{Target: line, NewExtra: extra})
}

func (m *Manager) setSelectedLineStartAnchor(isEnabled *bool, anchorID *string) {
	line, ok := m.singleSelectedLine()
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsStartAnchorEnabled: isEnabled, StartAnchorID: anchorID})
		return
	}
	extra := line.Extra()
	extra.IsStartAnchorEnabled = valueOr(isEnabled, extra.IsStartAnchorEnabled)
	extra.UserSelectedStartAnchor = shape.AnchorCharOr(valueOr(anchorID, ""), extra.UserSelectedStartAnchor)
	m.shapeManager.Execute(shape.ChangeExtra{Target: line, NewExtra: extra})
}

func (m *Manager) setSelectedLineEndAnchor(isEnabled *bool, anchorID *string) {
	line, ok := m.singleSelectedLine()
	if !ok {
		shape.SetDefaultValues(shape.DefaultValuesUpdate{IsEndAnchorEnabled: isEnabled, EndAnchorID: anchorID})
		return
	}
	extra := line.Extra()
	extra.IsEndAnchorEnabled = valueOr(isEnabled, extra.IsEndAnchorEnabled)
	extra.UserSelectedEndAnchor = shape.AnchorCharOr(valueOr(anchorID, ""), extra.UserSelectedEndAnchor)
	m.shapeManager.Execute(shape.ChangeExtra{Target: line, NewExtra: extra})
}

func changeDashPattern(pattern shape.StraightStrokeDashPattern, dash, gap, offset *int) shape.StraightStrokeDashPattern {
	return shape.StraightStrokeDashPattern{
		Dash:   valueOr(dash, pattern.Dash),
		Gap:    valueOr(gap, pattern.Gap),
		Offset: valueOr(offset, pattern.Offset),
	}
}

func valueOr[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}
