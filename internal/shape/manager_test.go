package shape

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/geo"
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

func TestManager_LayerAttributes(t *testing.T) {
	target := NewManager()
	rectangle := NewRectangle(geo.Rect{}, "", "")
	target.Execute(AddShape{Shape: rectangle})

	version := target.Version.Value()
	target.Execute(Rename{Target: rectangle, Name: "Header"})
	target.Execute(SetHidden{Target: rectangle, IsHidden: true})
	target.Execute(SetLocked{Target: rectangle, IsLocked: true})
	assert.NotEqual(t, version, target.Version.Value())

	// They are kept in the file, and left out when unset.
	data, err := json.Marshal(SerializableShapes{rectangle.Serializable(true)})
	assert.NoError(t, err)
	for _, field := range []string{`"name":"Header"`, `"isHidden":true`, `"isLocked":true`} {
		assert.Contains(t, string(data), field)
	}
	var restored SerializableShapes
	assert.NoError(t, json.Unmarshal(data, &restored))
	shape := RestoreShape(restored[0], "")
	assert.Equal(t, "Header", shape.Name())
	assert.True(t, shape.IsHidden())
	assert.True(t, shape.IsLocked())

	data, err = json.Marshal(SerializableShapes{NewRectangle(geo.Rect{}, "", "").Serializable(true)})
	assert.NoError(t, err)
	assert.NotContains(t, string(data), "name")
	assert.NotContains(t, string(data), "isHidden")
}

func TestManager_MoveShape(t *testing.T) {
	target := NewManager()
	group := NewGroup("", "")
	inner := NewGroup("", group.ID())
	first := NewRectangle(geo.Rect{}, "", group.ID())
	second := NewRectangle(geo.Rect{}, "", inner.ID())
	third := NewRectangle(geo.Rect{}, "", inner.ID())
	outside := NewRectangle(geo.Rect{}, "", "")
	for _, shape := range []Shape{group, inner, first, second, third, outside} {
		target.Execute(AddShape{Shape: shape})
	}

	// A group cannot move into its own items.
	target.Execute(MoveShape{Shape: group, NewParent: inner})
	assert.Equal(t, []Shape{group, outside}, target.Root().Items())

	// Into another group, to the bottom.
	version := target.Version.Value()
	target.Execute(MoveShape{Shape: outside, NewParent: inner})
	assert.Equal(t, []Shape{outside, second, third}, inner.Items())
	assert.Equal(t, inner.ID(), outside.ParentID())
	assert.NotEqual(t, version, target.Version.Value())

	// In the same group, above another item.
	target.Execute(MoveShape{Shape: outside, NewParent: inner, BelowID: third.ID()})
	assert.Equal(t, []Shape{second, third, outside}, inner.Items())

	// Leaving one item in group ungroups it.
	target.Execute(MoveShape{Shape: inner, NewParent: target.Root(), BelowID: group.ID()})
	assert.Equal(t, []Shape{first, inner}, target.Root().Items())
	_, ok := target.Shape(group.ID())
	assert.False(t, ok)
}

func TestManager_RemoveGroup_UnregistersItems(t *testing.T) {
	target := NewManager()
	outer := NewGroup("", "")
	inner := NewGroup("", outer.ID())
	rectangle := NewRectangle(geo.Rect{}, "", inner.ID())
	for _, shape := range []Shape{outer, inner, rectangle, NewRectangle(geo.Rect{}, "", outer.ID())} {
		target.Execute(AddShape{Shape: shape})
	}

	target.Execute(RemoveShape{Shape: outer})

	for _, shape := range []Shape{outer, inner, rectangle} {
		_, ok := target.Shape(shape.ID())
		assert.False(t, ok, shape.ID())
	}
}

func TestManager_AddGroup_RegistersItems(t *testing.T) {
	// A group with items is added as a whole, as when a group is pasted.
	group := NewGroup("", "")
	inner := NewGroup("", group.ID())
	rectangle := NewRectangle(geo.Rect{}, "", inner.ID())
	group.add(inner)
	inner.add(rectangle)

	target := NewManager()
	target.Execute(AddShape{Shape: group})

	for _, shape := range []Shape{group, inner, rectangle} {
		_, ok := target.Shape(shape.ID())
		assert.True(t, ok, shape.ID())
	}
}
