// Package state connects the shapes, the board and the user interactions. It is the Go port
// of the original statemanager, without the UI. See ADR-0001 and ADR-0006.
package state

import (
	"math"
	"slices"

	"github.com/bluemir/nemokan/internal/action"
	"github.com/bluemir/nemokan/internal/bitmap"
	"github.com/bluemir/nemokan/internal/bitmapmanager"
	"github.com/bluemir/nemokan/internal/board"
	"github.com/bluemir/nemokan/internal/cursor"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/interactionbound"
	"github.com/bluemir/nemokan/internal/observable"
	"github.com/bluemir/nemokan/internal/selection"
	"github.com/bluemir/nemokan/internal/shape"
	"github.com/bluemir/nemokan/internal/shapesearcher"
)

// Change is a kind of change JS reads on the next frame. See ADR-0006.
type Change int

const (
	ChangeBoard Change = iota
	ChangeInteractionBounds
	ChangeSelectionBound
	ChangeCursor
	ChangeTextEditing
	ChangeClipboard
	// ChangeBackup asks JS to call Backup after a while. See ADR-0007.
	ChangeBackup
	ChangeRetainableAction
	// ChangeShapeTool asks JS to read ShapeTool again.
	ChangeShapeTool
	// ChangeLayers asks JS to read Layers again.
	ChangeLayers
)

// EditingMode tells whether shapes are being changed by a mouse command or a text editor.
// SkippedVersion is the version not to keep in the history, when the change is canceled.
type EditingMode struct {
	IsEditing      bool
	SkippedVersion *int
}

// interactionPointRadiusPx is how close the mouse must be to an interaction point.
const interactionPointRadiusPx = 6

type Manager struct {
	shapeManager  *shape.Manager
	selection     *selection.Manager
	bitmapManager *bitmapmanager.Manager
	searcher      *shapesearcher.Searcher
	actions       *action.Manager
	board         *board.Board

	// workingParentGroup is the group shape actions apply to, like the current directory of
	// a file system.
	workingParentGroup *shape.Group

	access            Access
	drawingInfo       DrawingInfo
	interactionBounds []interactionbound.Bound
	selectionBound    *geo.Rect
	mouseCursor       cursor.Cursor
	textEditing       *TextEditing
	// clipboardText is the text JS has not written to the clipboard yet.
	clipboardText *string

	history                historyStack
	backupRequestedVersion *int

	EditingMode *observable.Value[EditingMode]

	isRedrawRequested bool
	mouse             *mouseInteractionController
	pointerTracker    *mousePointerTracker

	// notify tells JS what changed.
	notify func(Change)
}

func New(shapeManager *shape.Manager, actions *action.Manager, notify func(Change)) *Manager {
	bitmapManager := bitmapmanager.New()
	m := &Manager{
		shapeManager:       shapeManager,
		selection:          selection.New(),
		bitmapManager:      bitmapManager,
		searcher:           shapesearcher.New(shapeManager, bitmapManager),
		actions:            actions,
		board:              board.New(),
		workingParentGroup: shapeManager.Root(),
		drawingInfo:        defaultDrawingInfo,
		mouseCursor:        cursor.Default,
		EditingMode:        observable.New(EditingMode{}),
		pointerTracker:     newMousePointerTracker(),
		notify:             notify,
	}
	m.mouse = newMouseInteractionController(m)

	shapeManager.Version.Observe(func(int) { m.requestRedraw() })
	m.selection.SelectedShapes.Observe(m.updateInteractionBounds)
	actions.OneTime.Observe(m.handleOneTimeAction)
	shapeManager.Version.Observe(func(int) { m.requestBackupIfIdle() })
	m.EditingMode.Observe(func(EditingMode) { m.requestBackupIfIdle() })
	actions.Retainable.Observe(func(retainable action.Retainable) {
		// Adding shapes is not allowed with AccessView.
		if m.access == AccessView && retainable != action.RetainableIdle {
			actions.SetRetainable(action.RetainableIdle)
			return
		}
		m.notify(ChangeRetainableAction)
	})

	shapeManager.Version.Observe(func(int) { m.leaveRemovedWorkingGroup() })
	notifyLayers := func() { m.notify(ChangeLayers) }
	m.selection.SelectedShapes.Observe(func([]shape.Shape) { notifyLayers() })
	shapeManager.Version.Observe(func(int) { notifyLayers() })

	notifyShapeTool := func() { m.notify(ChangeShapeTool) }
	m.selection.SelectedShapes.Observe(func([]shape.Shape) { notifyShapeTool() })
	shapeManager.Version.Observe(func(int) { notifyShapeTool() })
	actions.Retainable.Observe(func(action.Retainable) { notifyShapeTool() })
	shape.DefaultExtraUpdated.Observe(func(struct{}) { notifyShapeTool() })
	return m
}

func (m *Manager) ShapeManager() *shape.Manager     { return m.shapeManager }
func (m *Manager) Selection() *selection.Manager    { return m.selection }
func (m *Manager) SelectionBound() (geo.Rect, bool) { return derefRect(m.selectionBound) }
func (m *Manager) MouseCursor() cursor.Cursor       { return m.mouseCursor }
func (m *Manager) DrawingInfo() DrawingInfo         { return m.drawingInfo }
func (m *Manager) TextEditing() *TextEditing        { return m.textEditing }
func (m *Manager) WindowBound() geo.Rect            { return m.drawingInfo.BoardBound() }
func (m *Manager) InteractionBounds() []interactionbound.Bound {
	return m.interactionBounds
}

func derefRect(rect *geo.Rect) (geo.Rect, bool) {
	if rect == nil {
		return geo.Rect{}, false
	}
	return *rect, true
}

func (m *Manager) SetDrawingInfo(drawingInfo DrawingInfo) {
	m.drawingInfo = drawingInfo
	m.requestRedraw()
}

func (m *Manager) OnMouseDown(event MouseEvent) {
	m.handleMousePointers(m.pointerTracker.down(event, m.drawingInfo))
}

func (m *Manager) OnMouseUp(event MouseEvent) {
	m.handleMousePointers(m.pointerTracker.up(event, m.drawingInfo))
}

func (m *Manager) OnMouseMove(event MouseEvent) {
	m.handleMousePointers(m.pointerTracker.move(event, m.drawingInfo))
}

func (m *Manager) OnShiftKey(isPressed bool) {
	m.handleMousePointers(m.pointerTracker.shiftKey(isPressed))
}

func (m *Manager) handleMousePointers(pointers []MousePointer) {
	for _, pointer := range pointers {
		m.mouse.onMouseEvent(pointer)
		m.updateMouseCursor(pointer)
	}
	// The original posts this to run after the click following a mouse up. Running it after
	// all pointers of the event does the same.
	m.mouse.runPostedActions()
}

func (m *Manager) updateMouseCursor(pointer MousePointer) {
	var mouseCursor cursor.Cursor
	switch pointer := pointer.(type) {
	case MouseMove:
		if point, ok := m.interactionPoint(pointer.PointPx); ok {
			mouseCursor = point.Cursor()
		} else {
			mouseCursor = m.actions.Retainable.Value().Cursor()
		}
	case MouseDrag:
		mouseCursor = cursor.Default
		if command := m.mouse.currentCommand; command != nil {
			commandCursor, ok := command.mouseCursor()
			if !ok {
				return
			}
			mouseCursor = commandCursor
		}
	case MouseUp:
		mouseCursor = cursor.Default
	default:
		return
	}
	if mouseCursor != m.mouseCursor {
		m.mouseCursor = mouseCursor
		m.notify(ChangeCursor)
	}
}

func (m *Manager) requestRedraw() {
	if m.isRedrawRequested {
		return
	}
	m.isRedrawRequested = true
	m.notify(ChangeBoard)
}

// BoardView is the board area shown on the canvas.
type BoardView struct {
	Bound geo.Rect
	// Rows are the chars of each row, one char for each cell. Transparent cells are spaces
	// and the tails of wide chars are WideCharTailInView. See ADR-0008.
	Rows []string
	// Highlights are the board.Highlight of each cell, row by row.
	Highlights []byte
}

// WideCharTailInView is the tail of a wide char in BoardView.Rows. A char followed by it
// takes two cells.
const WideCharTailInView = '\x00'

// Board redraws the board if requested and returns the area shown on the canvas.
func (m *Manager) Board() BoardView {
	if m.isRedrawRequested {
		m.redraw()
	}
	windowBound := m.WindowBound()
	// The original also draws one more row and column, for partially shown cells.
	bound := geo.RectByLTWH(windowBound.Left(), windowBound.Top(), windowBound.Width()+1, windowBound.Height()+1)

	view := BoardView{Bound: bound, Rows: make([]string, 0, bound.Height()), Highlights: make([]byte, 0, bound.Width()*bound.Height())}
	chars := make([]rune, bound.Width())
	for row := bound.Top(); row <= bound.Bottom(); row++ {
		for column := bound.Left(); column <= bound.Right(); column++ {
			pixel := m.board.Get(column, row)
			view.Highlights = append(view.Highlights, byte(pixel.Highlight))
			chars[column-bound.Left()] = viewChar(pixel, m.board.Get(column+1, row))
		}
		view.Rows = append(view.Rows, string(chars))
	}
	return view
}

func viewChar(pixel, nextPixel board.Pixel) rune {
	switch {
	case pixel.Visual == bitmap.WideCharTail:
		return WideCharTailInView
	case pixel.IsTransparent():
		return ' '
	// A wide char whose tail is covered by another shape would be drawn over that shape.
	case bitmap.CharWidth(pixel.Visual) == 2 && nextPixel.Visual != bitmap.WideCharTail:
		return ' '
	}
	return pixel.Visual
}

func (m *Manager) redraw() {
	m.isRedrawRequested = false
	windowBound := m.WindowBound()
	m.searcher.Clear(windowBound)
	m.board.ClearAndSetWindow(windowBound)
	m.drawShape(m.shapeManager.Root())
}

func (m *Manager) drawShape(target shape.Shape) {
	if target.IsHidden() {
		return
	}
	if group, ok := target.(*shape.Group); ok {
		for _, child := range group.Items() {
			m.drawShape(child)
		}
		return
	}
	drawn := m.bitmapManager.Bitmap(target)
	if drawn == nil {
		return
	}
	m.board.Fill(target.Bound().Position, drawn, m.highlight(target))
	m.searcher.Register(target)
}

func (m *Manager) highlight(target shape.Shape) board.Highlight {
	if text, ok := target.(*shape.Text); ok && text.IsTextEditing() {
		return board.HighlightTextEditing
	}
	if m.isSelectedInGroups(target) {
		return board.HighlightSelected
	}
	var focusType selection.FocusType
	ok := false
	// A hovered group highlights the shapes in it.
	for current := target; current != nil && !ok; current = m.parentShape(current) {
		focusType, ok = m.selection.FocusType(current)
	}
	if !ok {
		return board.HighlightNone
	}
	switch focusType {
	case selection.FocusLineConnecting:
		return board.HighlightLineConnectFocusing
	default:
		return board.HighlightSelected
	}
}

func (m *Manager) updateInteractionBounds(selectedShapes []shape.Shape) {
	var bounds []interactionbound.Bound
	for _, selected := range m.interactionBoundShapes(selectedShapes) {
		// A group, a locked shape and a widget of a fixed size cannot be scaled, so only their
		// bound is shown.
		widget, isWidget := selected.(*shape.Widget)
		if _, ok := selected.(*shape.Group); ok || m.isLockedInGroups(selected) || isWidget && !widget.IsResizeable() {
			bounds = append(bounds, interactionbound.NewOutlineBound(selected.Bound()))
			continue
		}
		switch selected := selected.(type) {
		case *shape.Rectangle, *shape.Text, *shape.Widget:
			bounds = append(bounds, interactionbound.NewScalableBound(selected.ID(), selected.Bound()))
		case *shape.Line:
			bounds = append(bounds, interactionbound.NewLineBound(selected.ID(), selected.Edges()))
		}
	}
	m.interactionBounds = bounds
	m.notify(ChangeInteractionBounds)
	m.requestRedraw()
}

func (m *Manager) refreshInteractionBounds() {
	m.updateInteractionBounds(m.selection.SelectedShapes.Value())
}

// interactionPoint returns the interaction point around pointPx. Later bounds and points are
// on top.
func (m *Manager) interactionPoint(pointPx geo.Point) (interactionbound.Point, bool) {
	for i := len(m.interactionBounds) - 1; i >= 0; i-- {
		points := m.interactionBounds[i].Points()
		for j := len(points) - 1; j >= 0; j-- {
			position := points[j].Position()
			if math.Abs(m.drawingInfo.toXPx(position.Left)-float64(pointPx.Left)) < interactionPointRadiusPx &&
				math.Abs(m.drawingInfo.toYPx(position.Top)-float64(pointPx.Top)) < interactionPointRadiusPx {
				return points[j], true
			}
		}
	}
	return nil, false
}

// isPointInInteractionBounds reports whether point is on a selected shape the mouse can move.
func (m *Manager) isPointInInteractionBounds(point geo.Point) bool {
	return slices.ContainsFunc(m.selection.SelectedShapes.Value(), func(selected shape.Shape) bool {
		return selected.Contains(point) && !m.isLockedInGroups(selected)
	})
}

func (m *Manager) setSelectionBound(bound *geo.Rect) {
	m.selectionBound = bound
	m.notify(ChangeSelectionBound)
}

func (m *Manager) enterEditingMode() {
	m.EditingMode.Set(EditingMode{IsEditing: true})
}

func (m *Manager) exitEditingMode() {
	m.EditingMode.Set(EditingMode{})
}

// exitEditingModeSkippingHistory does not keep the current state in the history.
func (m *Manager) exitEditingModeSkippingHistory() {
	skippedVersion := m.shapeManager.Version.Value()
	m.EditingMode.Set(EditingMode{SkippedVersion: &skippedVersion})
}

func (m *Manager) addShape(target shape.Shape) {
	m.shapeManager.Execute(shape.AddShape{Shape: target})
}

func (m *Manager) removeShape(target shape.Shape) {
	m.shapeManager.Execute(shape.RemoveShape{Shape: target})
}

func (m *Manager) addSelectedShape(target shape.Shape) {
	if target != nil {
		m.selection.Add(target)
	}
}

// selectAllShapes selects the items of the working group which the canvas can select.
func (m *Manager) selectAllShapes() {
	for _, item := range m.workingParentGroup.Items() {
		if !item.IsHidden() && !item.IsLocked() {
			m.addSelectedShape(item)
		}
	}
}
