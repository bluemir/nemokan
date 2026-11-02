package state

import (
	"slices"
	"strings"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/cursor"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/interactionbound"
	"github.com/bluemir/monosketch/internal/selection"
	"github.com/bluemir/monosketch/internal/shape"
)

type commandResult int

const (
	commandWorking commandResult = iota
	// commandWorkingPhase2 means the mouse part is done, but the command still works without
	// the mouse, such as typing text.
	commandWorkingPhase2
	commandDone
	commandUnknown
)

// mouseCommand handles mouse pointers from mouse down until it is done.
type mouseCommand interface {
	// mouseCursor returns false to keep the current cursor.
	mouseCursor() (cursor.Cursor, bool)
	execute(m *Manager, pointer MousePointer) commandResult
}

func rectByPoints(point1, point2 geo.Point) geo.Rect {
	return geo.RectByLTRB(point1.Left, point1.Top, point2.Left, point2.Top)
}

type selectShapeCommand struct{}

func (selectShapeCommand) mouseCursor() (cursor.Cursor, bool) { return cursor.Default, true }

func (selectShapeCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDown:
		return commandWorking
	case MouseDrag:
		bound := rectByPoints(pointer.MouseDownPoint, pointer.Point)
		m.setSelectionBound(&bound)
		return commandWorking
	case MouseUp:
		m.setSelectionBound(nil)
		area := rectByPoints(pointer.MouseDownPoint, pointer.Point)
		var shapes []shape.Shape
		if area.Width()*area.Height() > 1 {
			shapes = m.searcher.ShapesInZone(area)
		}
		if !pointer.IsWithShiftKey {
			m.selection.Clear()
		}
		for _, found := range shapes {
			if item, ok := m.workingItem(found); ok {
				m.addSelectedShape(item)
			}
		}
		return commandWorking
	case MouseClick:
		top, ok := m.topWorkingItem(m.searcher.Shapes(pointer.Point))
		switch {
		case !ok && !pointer.IsWithShiftKey:
			// Clicking outside the working group goes back to the root. See ADR-0013.
			m.setWorkingParentGroup(m.shapeManager.Root())
		case !ok:
		case pointer.IsWithShiftKey:
			m.selection.Toggle(top)
		default:
			m.selection.Clear()
			m.addSelectedShape(top)
		}
		return commandDone
	default:
		return commandDone
	}
}

type moveShapeCommand struct {
	shapes           []shape.Shape
	initialPositions map[string]geo.Point
}

func newMoveShapeCommand(shapes, relatedShapes []shape.Shape) *moveShapeCommand {
	initialPositions := map[string]geo.Point{}
	for _, target := range slices.Concat(shapes, relatedShapes) {
		initialPositions[target.ID()] = target.Bound().Position
	}
	return &moveShapeCommand{shapes: shapes, initialPositions: initialPositions}
}

func (*moveShapeCommand) mouseCursor() (cursor.Cursor, bool) { return cursor.Move, true }

func (c *moveShapeCommand) execute(m *Manager, pointer MousePointer) commandResult {
	var offset geo.Point
	isUpdateConfirmed := false
	switch pointer := pointer.(type) {
	case MouseDrag:
		offset = pointer.Point.Minus(pointer.MouseDownPoint)
	case MouseUp:
		offset = pointer.Point.Minus(pointer.MouseDownPoint)
		isUpdateConfirmed = true
	}

	m.moveShapes(c.shapes, isUpdateConfirmed, func(target shape.Shape) (geo.Point, bool) {
		position, ok := c.initialPositions[target.ID()]
		return position.Plus(offset), ok
	})
	m.refreshInteractionBounds()

	switch pointer.(type) {
	case MouseUp, MouseIdle:
		return commandDone
	}
	return commandWorking
}

type scaleShapeCommand struct {
	target       shape.Shape
	point        *interactionbound.ScalePoint
	initialBound geo.Rect
}

func (*scaleShapeCommand) mouseCursor() (cursor.Cursor, bool) { return "", false }

func (c *scaleShapeCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDrag:
		c.scale(m, pointer.PointF, false)
	case MouseUp:
		c.scale(m, pointer.PointF, true)
	case MouseIdle:
		return commandDone
	}
	return commandWorking
}

func (c *scaleShapeCommand) scale(m *Manager, pointF geo.PointF, isUpdateConfirmed bool) {
	newBound := c.point.NewShapeBound(c.initialBound, pointF)
	m.shapeManager.Execute(shape.ChangeBound{Target: c.target, NewBound: newBound})
	m.updateConnectors(c.target, newBound, isUpdateConfirmed)
	m.refreshInteractionBounds()
}

// addTextCommand adds a text shape: it sets the bound by mouse, then opens the text editor
// on mouse up. A rectangle is added by this too, as a text which is not editable.
type addTextCommand struct {
	isTextEditable bool
	workingShape   *shape.Text
}

func (*addTextCommand) mouseCursor() (cursor.Cursor, bool) { return cursor.Crosshair, true }

func (c *addTextCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDown:
		text := shape.NewTextByPoints(pointer.Point, pointer.Point, "", m.workingParentGroup.ID(), c.isTextEditable)
		c.workingShape = text
		m.addShape(text)
		m.selection.Clear()
		return commandWorking
	case MouseDrag:
		c.changeShapeBound(m, pointer.MouseDownPoint, pointer.Point)
		return commandWorking
	case MouseUp:
		c.onMouseUp(m, pointer)
		if !c.isTextEditable {
			// The original returns phase 2 for a rectangle too, which leaves the editing
			// mode on until the next mouse command is done. A rectangle has no typing phase.
			return commandDone
		}
		return commandWorkingPhase2
	default:
		return commandUnknown
	}
}

func (c *addTextCommand) changeShapeBound(m *Manager, point1, point2 geo.Point) {
	if c.workingShape == nil {
		return
	}
	m.shapeManager.Execute(shape.ChangeBound{Target: c.workingShape, NewBound: rectByPoints(point1, point2)})
}

func (c *addTextCommand) onMouseUp(m *Manager, pointer MouseUp) {
	text := c.workingShape
	if text == nil {
		return
	}
	c.changeShapeBound(m, pointer.MouseDownPoint, pointer.Point)

	// A one cell text grows with its content, without a border.
	isFreeText := text.IsTextEditable() && text.Bound().Width() == 1 && text.Bound().Height() == 1
	if isFreeText {
		extra := text.Extra()
		extra.BoundExtra.IsFillEnabled = false
		extra.BoundExtra.IsBorderEnabled = false
		extra.TextAlign = shape.TextAlign{Horizontal: shape.AlignLeft, Vertical: shape.AlignTop}
		m.shapeManager.Execute(shape.ChangeExtra{Target: text, NewExtra: extra})
	} else {
		m.addSelectedShape(text)
	}

	if c.isTextEditable {
		c.startTyping(m, text, isFreeText)
	}
}

func (c *addTextCommand) startTyping(m *Manager, text *shape.Text, isFreeText bool) {
	m.enterEditingMode()
	m.shapeManager.Execute(shape.UpdateTextEditingMode{Target: text, IsEditing: true})
	m.startTextEditing(text, isFreeText, func(finalText string) {
		m.shapeManager.Execute(shape.UpdateTextEditingMode{Target: text, IsEditing: false})
		c.adjustTextShape(m, text, isFreeText)
		if finalText != "" {
			m.exitEditingMode()
		} else {
			m.exitEditingModeSkippingHistory()
		}
		c.workingShape = nil
	})
}

func (c *addTextCommand) adjustTextShape(m *Manager, text *shape.Text, isFreeText bool) {
	if !isFreeText {
		m.adjustNormalTextSize(text)
		m.addSelectedShape(text)
		return
	}
	if text.Text() == "" {
		m.removeShape(text)
		return
	}
	lines := strings.Split(text.Text(), "\n")
	width := 0
	for _, line := range lines {
		width = max(width, bitmap.StringWidth(line))
	}
	newBound := geo.Rect{Position: text.Bound().Position, Size: geo.Size{Width: width, Height: len(lines)}}
	m.shapeManager.Execute(shape.ChangeBound{Target: text, NewBound: newBound})
	m.addSelectedShape(text)
}

type addLineCommand struct {
	workingShape     *shape.Line
	lineConnectHover *hoverShapeCache
}

func newAddLineCommand(m *Manager) *addLineCommand {
	return &addLineCommand{lineConnectHover: newLineConnectHoverCache(m)}
}

func (*addLineCommand) mouseCursor() (cursor.Cursor, bool) { return cursor.Crosshair, true }

func (c *addLineCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDown:
		c.workingShape = c.createLine(m, pointer.Point)
		m.selection.Clear()
		return commandWorking
	case MouseDrag:
		c.changeEndAnchor(m, pointer.Point, pointer.IsWithShiftKey, false)
		return commandWorking
	case MouseUp:
		c.changeEndAnchor(m, pointer.Point, pointer.IsWithShiftKey, true)
		if c.workingShape != nil {
			m.addSelectedShape(c.workingShape)
		}
		return commandDone
	default:
		return commandUnknown
	}
}

func (c *addLineCommand) createLine(m *Manager, point geo.Point) *shape.Line {
	direction := geo.Horizontal
	if edgeDirection, ok := m.searcher.EdgeDirection(point); ok {
		direction = edgeDirection.Normalized()
	}
	line := shape.NewLine(
		geo.DirectedPoint{Direction: direction, Left: point.Left, Top: point.Top},
		geo.DirectedPoint{Direction: geo.Vertical, Left: point.Left, Top: point.Top},
		"", m.workingParentGroup.ID(),
	)
	m.addShape(line)

	if connectShape, ok := shape.ConnectableShape(point, m.searcher.Shapes(point)); ok {
		m.shapeManager.ShapeConnector().AddConnector(line, shape.AnchorStart, connectShape)
	}
	return line
}

// changeEndAnchor moves the end of the line to point. In straight line mode (shift key), the
// line is horizontal or vertical.
func (c *addLineCommand) changeEndAnchor(m *Manager, point geo.Point, isStraightLineMode, isUpdateConfirmed bool) {
	line := c.workingShape
	if line == nil {
		return
	}
	endPoint := point
	if isStraightLineMode {
		endPoint = straightEndPoint(line.StartPoint().Point(), point)
	}
	direction := line.Direction(shape.AnchorEnd)
	if edgeDirection, ok := m.searcher.EdgeDirection(point); ok {
		direction = edgeDirection.Normalized()
	}
	anchorPointUpdate := shape.AnchorPointUpdate{
		Anchor: shape.AnchorEnd,
		Point:  geo.DirectedPoint{Direction: direction, Left: endPoint.Left, Top: endPoint.Top},
	}
	connectShape := c.lineConnectHover.hoverShape(endPoint)
	setLineConnectFocus(m, connectShape, isUpdateConfirmed)
	m.shapeManager.Execute(shape.MoveLineAnchor{
		Target:            line,
		AnchorPointUpdate: anchorPointUpdate,
		IsUpdateConfirmed: isUpdateConfirmed,
		JustMoveAnchor:    false,
		ConnectShape:      connectShape,
	})
}

// setLineConnectFocus highlights the shape a line end will connect to, until the change is
// confirmed.
func setLineConnectFocus(m *Manager, connectShape shape.Shape, isUpdateConfirmed bool) {
	if isUpdateConfirmed {
		connectShape = nil
	}
	m.selection.SetFocusingShape(connectShape, selection.FocusLineConnecting)
}

func straightEndPoint(startPoint, candidateEndPoint geo.Point) geo.Point {
	width := abs(startPoint.Left - candidateEndPoint.Left)
	height := abs(startPoint.Top - candidateEndPoint.Top)
	if width > height {
		return geo.Point{Left: candidateEndPoint.Left, Top: startPoint.Top}
	}
	return geo.Point{Left: startPoint.Left, Top: candidateEndPoint.Top}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// lineAnchorCommand moves an end of a line. Without shift, only the end moves; with shift,
// the line keeps its shape.
type lineAnchorCommand struct {
	line             *shape.Line
	point            *interactionbound.LineAnchorPoint
	lineConnectHover *hoverShapeCache
}

func newLineAnchorCommand(m *Manager, line *shape.Line, point *interactionbound.LineAnchorPoint) *lineAnchorCommand {
	return &lineAnchorCommand{line: line, point: point, lineConnectHover: newLineConnectHoverCache(m)}
}

func (*lineAnchorCommand) mouseCursor() (cursor.Cursor, bool) { return cursor.Crosshair, true }

func (c *lineAnchorCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDrag:
		c.move(m, pointer.Point, false, !pointer.IsWithShiftKey)
	case MouseUp:
		c.move(m, pointer.Point, true, !pointer.IsWithShiftKey)
	case MouseIdle:
		return commandDone
	}
	return commandWorking
}

func (c *lineAnchorCommand) move(m *Manager, point geo.Point, isUpdateConfirmed, justMoveAnchor bool) {
	direction := c.line.Direction(c.point.Anchor)
	if edgeDirection, ok := m.searcher.EdgeDirection(point); ok {
		direction = edgeDirection.Normalized()
	}
	anchorPointUpdate := shape.AnchorPointUpdate{
		Anchor: c.point.Anchor,
		Point:  geo.DirectedPoint{Direction: direction, Left: point.Left, Top: point.Top},
	}
	connectShape := c.lineConnectHover.hoverShape(point)
	setLineConnectFocus(m, connectShape, isUpdateConfirmed)
	m.shapeManager.Execute(shape.MoveLineAnchor{
		Target:            c.line,
		AnchorPointUpdate: anchorPointUpdate,
		IsUpdateConfirmed: isUpdateConfirmed,
		JustMoveAnchor:    justMoveAnchor,
		ConnectShape:      connectShape,
	})
	m.refreshInteractionBounds()
}

// lineEdgeCommand moves an edge of a line.
type lineEdgeCommand struct {
	line  *shape.Line
	point *interactionbound.LineEdgePoint
}

func (*lineEdgeCommand) mouseCursor() (cursor.Cursor, bool) { return "", false }

func (c *lineEdgeCommand) execute(m *Manager, pointer MousePointer) commandResult {
	switch pointer := pointer.(type) {
	case MouseDrag:
		c.move(m, pointer.Point, false)
	case MouseUp:
		c.move(m, pointer.Point, true)
	case MouseIdle:
		return commandDone
	}
	return commandWorking
}

func (c *lineEdgeCommand) move(m *Manager, point geo.Point, isUpdateConfirmed bool) {
	m.shapeManager.Execute(shape.MoveLineEdge{Target: c.line, EdgeID: c.point.EdgeID, Point: point, IsUpdateConfirmed: isUpdateConfirmed})
	m.refreshInteractionBounds()
}
