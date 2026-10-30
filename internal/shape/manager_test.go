package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/geo"
)

func TestManager_Add(t *testing.T) {
	target := NewManager()

	shape1 := newMockShape(geo.Rect{}, "")
	target.Execute(AddShape{Shape: shape1})
	assert.Equal(t, []Shape{shape1}, target.Root().Items())

	group1 := NewGroup("", "")
	target.Execute(AddShape{Shape: group1})
	assert.Equal(t, []Shape{shape1, group1}, target.Root().Items())

	shape2 := newMockShape(geo.Rect{}, group1.ID())
	target.Execute(AddShape{Shape: shape2})
	assert.Equal(t, []Shape{shape1, group1}, target.Root().Items())
	assert.Equal(t, []Shape{shape2}, group1.Items())

	shape3 := newMockShape(geo.Rect{}, "1000")
	target.Execute(AddShape{Shape: shape3})
	assert.Equal(t, []Shape{shape1, group1}, target.Root().Items())
	assert.Equal(t, []Shape{shape2}, group1.Items())
}

func TestManager_Remove_SingleGroupItemRemovesGroup(t *testing.T) {
	target := NewManager()
	group := NewGroup("", "")
	shape := NewRectangle(geo.Rect{}, "", group.ID())

	target.Execute(AddShape{Shape: group})
	target.Execute(AddShape{Shape: shape})
	target.Execute(RemoveShape{Shape: shape})
	assert.Equal(t, 0, target.Root().ItemCount())
}

func TestManager_Remove_GroupItemUngroups(t *testing.T) {
	target := NewManager()
	group := NewGroup("", "")
	shape1 := NewRectangle(geo.Rect{}, "", "")
	shape2 := NewRectangle(geo.Rect{}, "", group.ID())
	shape3 := NewRectangle(geo.Rect{}, "", group.ID())

	for _, shape := range []Shape{group, shape1, shape2, shape3} {
		target.Execute(AddShape{Shape: shape})
	}

	target.Execute(RemoveShape{Shape: shape3})
	assert.Equal(t, []Shape{shape2, shape1}, target.Root().Items())
	assert.Equal(t, target.Root().ID(), shape2.ParentID())
}

func TestManager_Remove_GroupItemKeepsRoot(t *testing.T) {
	target := NewManager()
	group := NewGroup("", "")
	shape1 := NewRectangle(geo.Rect{}, "", "")
	shape2 := NewRectangle(geo.Rect{}, "", group.ID())
	shape3 := NewRectangle(geo.Rect{}, "", group.ID())
	shape4 := NewRectangle(geo.Rect{}, "", group.ID())

	for _, shape := range []Shape{group, shape1, shape2, shape3, shape4} {
		target.Execute(AddShape{Shape: shape})
	}

	target.Execute(RemoveShape{Shape: shape4})
	assert.Equal(t, []Shape{group, shape1}, target.Root().Items())
	assert.Equal(t, []Shape{shape2, shape3}, group.Items())
}

func TestManager_Group_Invalid(t *testing.T) {
	target := NewManager()
	group := NewGroup("", "")
	shape1 := NewRectangle(geo.Rect{}, "", "")
	shape2 := NewRectangle(geo.Rect{}, "", group.ID())

	for _, shape := range []Shape{group, shape1, shape2} {
		target.Execute(AddShape{Shape: shape})
	}

	target.Execute(GroupShapes{Shapes: []Shape{shape1}})
	assert.Equal(t, []Shape{group, shape1}, target.Root().Items())

	target.Execute(GroupShapes{Shapes: []Shape{shape1, shape2}})
	assert.Equal(t, []Shape{group, shape1}, target.Root().Items())
	assert.Equal(t, []Shape{shape2}, group.Items())
}

func TestManager_Group_Valid(t *testing.T) {
	target := NewManager()
	shapes := []Shape{
		NewRectangle(geo.Rect{}, "", ""),
		NewRectangle(geo.Rect{}, "", ""),
		NewRectangle(geo.Rect{}, "", ""),
		NewRectangle(geo.Rect{}, "", ""),
	}
	for _, shape := range shapes {
		target.Execute(AddShape{Shape: shape})
	}
	target.Execute(GroupShapes{Shapes: shapes[1:3]})

	items := target.Root().Items()
	assert.Len(t, items, 3)
	assert.Equal(t, shapes[0], items[0])
	assert.Equal(t, shapes[3], items[2])
	group := items[1].(*Group)
	assert.Equal(t, shapes[1:3], group.Items())
	assert.Equal(t, group, target.group(shapes[1].ParentID()))
	assert.Equal(t, group, target.group(shapes[2].ParentID()))
}

func TestManager_Ungroup(t *testing.T) {
	target := NewManager()
	group := NewGroup("", "")
	shape0 := NewRectangle(geo.Rect{}, "", "")
	shape1 := NewRectangle(geo.Rect{}, "", group.ID())
	shape2 := NewRectangle(geo.Rect{}, "", group.ID())
	shape3 := NewRectangle(geo.Rect{}, "", "")

	for _, shape := range []Shape{shape0, group, shape1, shape2, shape3} {
		target.Execute(AddShape{Shape: shape})
	}

	target.Execute(Ungroup{Group: group})
	assert.Equal(t, []Shape{shape0, shape1, shape2, shape3}, target.Root().Items())
	assert.Equal(t, target.Root().ID(), shape1.ParentID())
	assert.Equal(t, target.Root().ID(), shape2.ParentID())
}

func TestManager_RecursiveVersionUpdate(t *testing.T) {
	target := NewManager()
	group0 := NewGroup("", "")
	target.Execute(AddShape{Shape: group0})
	group1 := NewGroup("", group0.ID())
	target.Execute(AddShape{Shape: group1})

	rootOldVersion := target.Root().VersionCode()
	group0OldVersion := group0.VersionCode()
	group1OldVersion := group1.VersionCode()

	target.Execute(AddShape{Shape: NewGroup("", group1.ID())})

	assert.NotEqual(t, rootOldVersion, target.Root().VersionCode())
	assert.NotEqual(t, group0OldVersion, group0.VersionCode())
	assert.NotEqual(t, group1OldVersion, group1.VersionCode())
}

// The original loops forever when the root has a child group.
func TestManager_ReplaceRoot_NestedGroup(t *testing.T) {
	source := NewManager()
	group := NewGroup("", "")
	source.Execute(AddShape{Shape: group})
	rectangle := NewRectangle(geo.Rect{}, "", group.ID())
	source.Execute(AddShape{Shape: rectangle})
	source.Execute(AddShape{Shape: NewRectangle(geo.Rect{}, "", group.ID())})

	target := NewManager()
	target.ReplaceRoot(RestoreRoot(source.Root().Serializable(true).(*SerializableGroup)), NewShapeConnector())

	restored, ok := target.Shape(rectangle.ID())
	assert.True(t, ok)
	assert.Equal(t, group.ID(), restored.ParentID())
}
