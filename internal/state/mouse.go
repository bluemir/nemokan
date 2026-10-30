package state

import "github.com/bluemir/monosketch/internal/geo"

// MousePointer is a mouse state on the board: MouseIdle, MouseMove, MouseDown, MouseDrag,
// MouseUp, MouseClick or MouseDoubleClick.
type MousePointer interface {
	BoardCoordinate() geo.Point
}

type MouseIdle struct{}

type MouseMove struct {
	Point   geo.Point
	PointPx geo.Point
}

type MouseDown struct {
	Point          geo.Point
	PointPx        geo.Point
	IsWithShiftKey bool
}

type MouseDrag struct {
	MouseDownPoint geo.Point
	Point          geo.Point
	PointF         geo.PointF
	IsWithShiftKey bool
}

type MouseUp struct {
	MouseDownPoint geo.Point
	Point          geo.Point
	PointF         geo.PointF
	IsWithShiftKey bool
}

type MouseClick struct {
	Point          geo.Point
	IsWithShiftKey bool
}

type MouseDoubleClick struct {
	Point geo.Point
}

func (MouseIdle) BoardCoordinate() geo.Point          { return geo.Point{} }
func (p MouseMove) BoardCoordinate() geo.Point        { return p.Point }
func (p MouseDown) BoardCoordinate() geo.Point        { return p.Point }
func (p MouseDrag) BoardCoordinate() geo.Point        { return p.Point }
func (p MouseUp) BoardCoordinate() geo.Point          { return p.Point }
func (p MouseClick) BoardCoordinate() geo.Point       { return p.Point }
func (p MouseDoubleClick) BoardCoordinate() geo.Point { return p.Point }

// MouseEvent is a mouse event on the canvas, sent from JS.
type MouseEvent struct {
	// OffsetPx is the position from the top-left of the canvas.
	OffsetPx       geo.Point `json:"offsetPx"`
	IsWithShiftKey bool      `json:"isWithShiftKey"`
	// TimeMillis is the event time, used for detecting double clicks.
	TimeMillis float64 `json:"timeMillis"`
}

// doubleClickMillis is the longest time between two mouse ups of a double click.
const doubleClickMillis = 500

// mousePointerTracker turns mouse events into mouse pointers.
type mousePointerTracker struct {
	current MousePointer

	lastMouseUpMillis float64
	mouseUpCount      int
}

func newMousePointerTracker() *mousePointerTracker {
	return &mousePointerTracker{current: MouseIdle{}}
}

func (t *mousePointerTracker) down(event MouseEvent, drawingInfo DrawingInfo) []MousePointer {
	switch t.current.(type) {
	case MouseIdle, MouseMove:
		return t.set(MouseDown{
			Point:          drawingInfo.toBoardPoint(event.OffsetPx),
			PointPx:        event.OffsetPx,
			IsWithShiftKey: event.IsWithShiftKey,
		})
	}
	return nil
}

func (t *mousePointerTracker) up(event MouseEvent, drawingInfo DrawingInfo) []MousePointer {
	isDoubleClick := event.TimeMillis-t.lastMouseUpMillis < doubleClickMillis && t.mouseUpCount > 0
	t.lastMouseUpMillis = event.TimeMillis
	t.mouseUpCount++

	point := drawingInfo.toBoardPoint(event.OffsetPx)
	pointF := drawingInfo.toBoardPointF(event.OffsetPx)

	var pointers []MousePointer
	switch current := t.current.(type) {
	case MouseDown:
		pointers = append(pointers, t.set(MouseUp{MouseDownPoint: current.Point, Point: point, PointF: pointF, IsWithShiftKey: event.IsWithShiftKey})...)
		if isDoubleClick {
			pointers = append(pointers, t.set(MouseDoubleClick{Point: point})...)
		} else {
			pointers = append(pointers, t.set(MouseClick{Point: point, IsWithShiftKey: event.IsWithShiftKey})...)
		}
	case MouseDrag:
		pointers = append(pointers, t.set(MouseUp{MouseDownPoint: current.MouseDownPoint, Point: point, PointF: pointF, IsWithShiftKey: event.IsWithShiftKey})...)
	}
	return append(pointers, t.set(MouseIdle{})...)
}

func (t *mousePointerTracker) move(event MouseEvent, drawingInfo DrawingInfo) []MousePointer {
	t.lastMouseUpMillis = 0
	t.mouseUpCount = 0

	point := drawingInfo.toBoardPoint(event.OffsetPx)
	switch current := t.current.(type) {
	case MouseDown:
		if point == current.Point {
			return nil
		}
		return t.set(MouseDrag{MouseDownPoint: current.Point, Point: point, PointF: drawingInfo.toBoardPointF(event.OffsetPx), IsWithShiftKey: event.IsWithShiftKey})
	case MouseDrag:
		return t.set(MouseDrag{MouseDownPoint: current.MouseDownPoint, Point: point, PointF: drawingInfo.toBoardPointF(event.OffsetPx), IsWithShiftKey: event.IsWithShiftKey})
	default:
		return t.set(MouseMove{Point: point, PointPx: event.OffsetPx})
	}
}

// shiftKey updates the shift key of a dragging pointer, for switching the line mode while
// dragging.
func (t *mousePointerTracker) shiftKey(isPressed bool) []MousePointer {
	current, ok := t.current.(MouseDrag)
	if !ok {
		return nil
	}
	current.IsWithShiftKey = isPressed
	return t.set(current)
}

// set returns pointer to handle, or nothing if it is the same as the current one.
func (t *mousePointerTracker) set(pointer MousePointer) []MousePointer {
	if pointer == t.current {
		return nil
	}
	t.current = pointer
	return []MousePointer{pointer}
}
