package shape

import (
	"fmt"

	"github.com/bluemir/nemokan/internal/geo"
)

type Rectangle struct {
	base
	bound geo.Rect
	extra RectangleExtra
}

// NewRectangle generates an id if id is empty.
func NewRectangle(bound geo.Rect, id, parentID string) *Rectangle {
	return &Rectangle{base: newBase(id, parentID), bound: bound, extra: defaultRectangleExtra}
}

// NewRectangleByPoints makes a rectangle whose corners are startPoint and endPoint.
func NewRectangleByPoints(startPoint, endPoint geo.Point, id, parentID string) *Rectangle {
	return NewRectangle(geo.RectByLTRB(startPoint.Left, startPoint.Top, endPoint.Left, endPoint.Top), id, parentID)
}

func restoreRectangle(serializable *SerializableRectangle, parentID string) *Rectangle {
	rectangle := NewRectangle(serializable.Bound, serializable.actualID(), parentID)
	rectangle.extra = serializable.Extra.restore()
	rectangle.restore(serializable.serializableBase)
	return rectangle
}

func (r *Rectangle) Bound() geo.Rect       { return r.bound }
func (r *Rectangle) Extra() RectangleExtra { return r.extra }

func (r *Rectangle) SetBound(newBound geo.Rect) {
	r.update(func() bool {
		isUpdated := r.bound != newBound
		r.bound = newBound
		return isUpdated
	})
}

func (r *Rectangle) SetExtra(newExtra Extra) {
	extra, ok := newExtra.(RectangleExtra)
	if !ok {
		panic(fmt.Sprintf("new extra is not a RectangleExtra (%T)", newExtra))
	}
	if extra == r.extra {
		return
	}
	r.update(func() bool {
		r.extra = extra
		return true
	})
}

func (r *Rectangle) Contains(point geo.Point) bool   { return r.bound.Contains(point) }
func (r *Rectangle) IsVertex(point geo.Point) bool   { return r.bound.IsVertex(point) }
func (r *Rectangle) IsOverlapped(rect geo.Rect) bool { return r.bound.IsOverlapped(rect) }

func (r *Rectangle) Serializable(isIDIncluded bool) SerializableShape {
	return &SerializableRectangle{
		serializableBase: r.serializableBase(isIDIncluded),
		Bound:            r.bound,
		Extra:            serializableRectangleExtra(r.extra),
	}
}
