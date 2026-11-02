package shape

import (
	"fmt"

	"github.com/bluemir/monosketch/internal/geo"
)

// Group is a shape that holds other shapes in order.
// The root of a sketch is a Group without a parent.
type Group struct {
	base
	shapes *quickList[Shape]
}

// NewGroup generates an id if id is empty.
func NewGroup(id, parentID string) *Group {
	return &Group{base: newBase(id, parentID), shapes: newQuickList[Shape]()}
}

func restoreGroup(serializable *SerializableGroup, parentID string) *Group {
	group := NewGroup(serializable.actualID(), parentID)
	for _, item := range serializable.Shapes {
		group.addInternal(RestoreShape(item, group.id))
	}
	group.restore(serializable.serializableBase)
	return group
}

// RestoreShape creates the shape described by serializable.
func RestoreShape(serializable SerializableShape, parentID string) Shape {
	switch serializable := serializable.(type) {
	case *SerializableRectangle:
		return restoreRectangle(serializable, parentID)
	case *SerializableText:
		return restoreText(serializable, parentID)
	case *SerializableLine:
		return restoreLine(serializable, parentID)
	case *SerializableGroup:
		return restoreGroup(serializable, parentID)
	case *SerializableWidget:
		return restoreWidget(serializable, parentID)
	}
	panic(fmt.Sprintf("unknown serializable shape: %T", serializable))
}

func (g *Group) Items() []Shape { return g.shapes.items() }
func (g *Group) ItemCount() int { return g.shapes.size() }

func (g *Group) Bound() geo.Rect {
	items := g.shapes.items()
	if len(items) == 0 {
		return geo.Rect{}
	}
	// TODO: Use heap or R-tree to improve performance
	bounds := make([]geo.Rect, len(items))
	for i, item := range items {
		bounds[i] = item.Bound()
	}
	return geo.BoundOf(bounds)
}

func (g *Group) SetBound(geo.Rect) {}

func (g *Group) SetExtra(newExtra Extra) {
	panic(fmt.Sprintf("group does not support extra (%T)", newExtra))
}

func (g *Group) Contains(point geo.Point) bool   { return g.Bound().Contains(point) }
func (g *Group) IsVertex(point geo.Point) bool   { return g.Bound().IsVertex(point) }
func (g *Group) IsOverlapped(rect geo.Rect) bool { return g.Bound().IsOverlapped(rect) }

func (g *Group) Serializable(isIDIncluded bool) SerializableShape {
	items := g.shapes.items()
	shapes := make([]SerializableShape, len(items))
	for i, item := range items {
		shapes[i] = item.Serializable(isIDIncluded)
	}
	return &SerializableGroup{
		serializableBase: g.serializableBase(isIDIncluded),
		Shapes:           shapes,
	}
}

func (g *Group) add(shape Shape) {
	g.update(func() bool { return g.addInternal(shape) })
}

// addAfter adds shape right after the item with afterID.
func (g *Group) addAfter(shape Shape, afterID string) {
	g.update(func() bool {
		if !g.acceptParent(shape) {
			return false
		}
		shape.setParentID(g.id)
		g.shapes.addAfter(shape, afterID)
		return true
	})
}

func (g *Group) addInternal(shape Shape) bool {
	if !g.acceptParent(shape) {
		return false
	}
	shape.setParentID(g.id)
	g.shapes.addLast(shape)
	return true
}

// acceptParent reports whether shape has no parent or already belongs to this group.
func (g *Group) acceptParent(shape Shape) bool {
	return shape.ParentID() == "" || shape.ParentID() == g.id
}

func (g *Group) remove(shape Shape) {
	g.update(func() bool {
		_, ok := g.shapes.remove(shape.ID())
		return ok
	})
}

func (g *Group) changeOrder(shape Shape, action moveAction) {
	g.update(func() bool { return g.shapes.move(shape.ID(), action) })
}

func (g *Group) String() string {
	return "Group(" + g.id + ")"
}

// RestoreRoot creates the root group of a sketch from its stored form.
func RestoreRoot(serializable *SerializableGroup) *Group {
	return restoreGroup(serializable, "")
}
