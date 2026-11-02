package state

import (
	"slices"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/interactionbound"
	"github.com/bluemir/monosketch/internal/selection"
	"github.com/bluemir/monosketch/internal/shape"
)

type mouseInteractionController struct {
	manager        *Manager
	currentCommand mouseCommand

	lineConnectHover *hoverShapeCache
	hover            *hoverShapeCache

	postedActions []func()
}

func newMouseInteractionController(manager *Manager) *mouseInteractionController {
	return &mouseInteractionController{
		manager:          manager,
		lineConnectHover: newLineConnectHoverCache(manager),
		hover:            newHoverCache(manager),
	}
}

func (c *mouseInteractionController) reset() {
	c.lineConnectHover.reset()
	c.hover.reset()
}

func (c *mouseInteractionController) runPostedActions() {
	actions := c.postedActions
	c.postedActions = nil
	for _, posted := range actions {
		posted()
	}
}

func (c *mouseInteractionController) onMouseEvent(pointer MousePointer) {
	m := c.manager
	switch pointer.(type) {
	case MouseDown, MouseUp:
		c.reset()
	case MouseDoubleClick:
		var targeted shape.Shape
		for _, selected := range m.selection.SelectedShapes.Value() {
			if selected.Contains(pointer.BoardCoordinate()) {
				targeted = selected
				break
			}
		}
		switch targeted := targeted.(type) {
		case *shape.Group:
			c.enterGroup(targeted, pointer)
			return
		case *shape.Widget:
			if !m.isLockedInGroups(targeted) {
				m.editWidgetAt(targeted, pointer.BoardCoordinate())
			}
			return
		}
		m.actions.SetOneTime(action.EditSelectedShape{Shape: targeted})
		return
	}

	if c.currentCommand == nil {
		c.detectHoverShape(pointer)
	}

	command := c.newCommand(pointer)
	if command == nil {
		command = c.currentCommand
	}
	if command == nil {
		return
	}
	c.currentCommand = command

	m.enterEditingMode()
	result := command.execute(m, pointer)
	if result == commandDone {
		m.exitEditingMode()
	}
	if result == commandDone || result == commandWorkingPhase2 {
		c.lineConnectHover.reset()
		c.currentCommand = nil
		m.requestRedraw()
		// Avoid the click after adding a shape selecting a shape.
		c.postedActions = append(c.postedActions, func() { m.actions.SetRetainable(action.RetainableIdle) })
	}
}

// enterGroup makes group the working group, and selects its item under the mouse. See
// ADR-0013.
func (c *mouseInteractionController) enterGroup(group *shape.Group, pointer MousePointer) {
	m := c.manager
	m.setWorkingParentGroup(group)
	m.selection.Clear()
	if item, ok := m.topWorkingItem(m.searcher.Shapes(pointer.BoardCoordinate())); ok {
		m.addSelectedShape(item)
	}
	c.hover.reset()
}

func (c *mouseInteractionController) detectHoverShape(pointer MousePointer) {
	m := c.manager
	var hoverShape shape.Shape
	var focusType selection.FocusType
	switch m.actions.Retainable.Value() {
	case action.AddLine:
		hoverShape = c.lineConnectHover.hoverShape(pointer.BoardCoordinate())
		focusType = selection.FocusLineConnecting
	case action.RetainableIdle:
		// Only hover a shape when the mouse is not on the interaction points of the selected
		// shapes.
		if !c.isOnInteractionPoint(pointer) {
			hoverShape = c.hover.hoverShape(pointer.BoardCoordinate())
		}
		focusType = selection.FocusSelectModeHover
	default:
		return
	}
	m.selection.SetFocusingShape(hoverShape, focusType)
	m.requestRedraw()
}

func (c *mouseInteractionController) isOnInteractionPoint(pointer MousePointer) bool {
	switch pointer := pointer.(type) {
	case MouseDown:
		_, ok := c.manager.interactionPoint(pointer.PointPx)
		return ok
	case MouseMove:
		_, ok := c.manager.interactionPoint(pointer.PointPx)
		return ok
	}
	return false
}

// newCommand returns the command a pointer starts, or nil to keep the current command.
func (c *mouseInteractionController) newCommand(pointer MousePointer) mouseCommand {
	m := c.manager
	if m.access == AccessView {
		return viewCommand(pointer)
	}
	switch pointer := pointer.(type) {
	case MouseDown:
		if command := c.interactionCommand(pointer); command != nil {
			return command
		}
		if kind, ok := m.actions.Retainable.Value().AddingWidget(); ok {
			return &addWidgetCommand{kind: kind}
		}
		switch m.actions.Retainable.Value() {
		case action.AddRectangle:
			// A rectangle is a text which is not editable.
			return &addTextCommand{isTextEditable: false}
		case action.AddText:
			return &addTextCommand{isTextEditable: true}
		case action.AddLine:
			return newAddLineCommand(m)
		default:
			return selectShapeCommand{}
		}
	case MouseClick:
		if m.actions.Retainable.Value() == action.RetainableIdle {
			return selectShapeCommand{}
		}
	}
	return nil
}

// viewCommand only selects shapes, as AccessView does not change them.
func viewCommand(pointer MousePointer) mouseCommand {
	switch pointer.(type) {
	case MouseDown, MouseClick:
		return selectShapeCommand{}
	}
	return nil
}

func (c *mouseInteractionController) interactionCommand(pointer MouseDown) mouseCommand {
	m := c.manager
	focusing := m.selection.FocusingShape.Value()
	if focusing != nil && focusing.FocusType != selection.FocusSelectModeHover {
		focusing = nil
	}
	// Without shift (multiple selection), mouse down on a hovered shape selects only it.
	if !pointer.IsWithShiftKey && focusing != nil && !slices.Contains(m.selection.SelectedShapes.Value(), focusing.Shape) {
		m.selection.Clear()
		m.addSelectedShape(focusing.Shape)
	}

	selectedShapes := m.selection.SelectedShapes.Value()
	if len(selectedShapes) == 0 {
		return nil
	}

	if point, ok := m.interactionPoint(pointer.PointPx); ok {
		if command := c.boundInteractionCommand(point); command != nil {
			return command
		}
	}

	if !pointer.IsWithShiftKey && m.isPointInInteractionBounds(pointer.Point) {
		movableShapes := m.movableSelectedShapes()
		var relatedShapes []shape.Shape
		for _, selected := range movableShapes {
			for _, connector := range m.shapeManager.ShapeConnector().Connectors(selected) {
				if line, ok := m.shapeManager.Shape(connector.LineID); ok {
					relatedShapes = append(relatedShapes, line)
				}
			}
		}
		return newMoveShapeCommand(movableShapes, relatedShapes)
	}
	return nil
}

func (c *mouseInteractionController) boundInteractionCommand(point interactionbound.Point) mouseCommand {
	target, ok := c.manager.shapeManager.Shape(point.ShapeID())
	if !ok {
		return nil
	}
	switch point := point.(type) {
	case *interactionbound.ScalePoint:
		return &scaleShapeCommand{target: target, point: point, initialBound: target.Bound()}
	case *interactionbound.LineAnchorPoint:
		if line, ok := target.(*shape.Line); ok {
			return newLineAnchorCommand(c.manager, line, point)
		}
	case *interactionbound.LineEdgePoint:
		if line, ok := target.(*shape.Line); ok {
			return &lineEdgeCommand{line: line, point: point}
		}
	}
	return nil
}
