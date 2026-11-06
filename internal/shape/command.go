package shape

import (
	"fmt"

	"github.com/bluemir/nemokan/internal/geo"
)

// Command changes shapes through Manager.Execute.
// A command is not executed if affectedParent returns nil.
type Command interface {
	affectedParent(manager *Manager) *Group
	execute(manager *Manager, parent *Group)
}

// updateParentIfChanged marks parent changed if the version of target changed by change.
func updateParentIfChanged(parent *Group, target Shape, change func()) {
	currentVersion := target.VersionCode()
	change()
	if currentVersion != target.VersionCode() {
		parent.update(func() bool { return true })
	}
}

type ChangeBound struct {
	Target   Shape
	NewBound geo.Rect
}

func (c ChangeBound) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c ChangeBound) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.SetBound(c.NewBound) })
}

type ChangeExtra struct {
	Target   Shape
	NewExtra Extra
}

func (c ChangeExtra) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c ChangeExtra) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.SetExtra(c.NewExtra) })
}

// MoveLineAnchor moves an end of a line.
// IsUpdateConfirmed merges the points on a straight line. ConnectShape is the shape to
// attach the end to, or nil to detach it.
type MoveLineAnchor struct {
	Target            *Line
	AnchorPointUpdate AnchorPointUpdate
	IsUpdateConfirmed bool
	JustMoveAnchor    bool
	ConnectShape      Shape
}

func (c MoveLineAnchor) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c MoveLineAnchor) execute(manager *Manager, parent *Group) {
	currentVersion := c.Target.VersionCode()
	c.Target.MoveAnchorPoint(c.AnchorPointUpdate, c.IsUpdateConfirmed, c.JustMoveAnchor)
	if currentVersion == c.Target.VersionCode() {
		return
	}

	if c.ConnectShape != nil {
		manager.shapeConnector.AddConnector(c.Target, c.AnchorPointUpdate.Anchor, c.ConnectShape)
	} else {
		manager.shapeConnector.RemoveConnector(c.Target, c.AnchorPointUpdate.Anchor)
	}
	parent.update(func() bool { return true })
}

// MoveLineEdge moves an edge of a line. IsUpdateConfirmed merges the points on a straight line.
type MoveLineEdge struct {
	Target            *Line
	EdgeID            int
	Point             geo.Point
	IsUpdateConfirmed bool
}

func (c MoveLineEdge) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c MoveLineEdge) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.MoveEdge(c.EdgeID, c.Point, c.IsUpdateConfirmed) })
}

type ChangeText struct {
	Target  *Text
	NewText string
}

func (c ChangeText) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c ChangeText) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.SetText(c.NewText) })
}

// MakeTextEditable makes an uneditable text editable.
type MakeTextEditable struct {
	Target *Text
}

func (c MakeTextEditable) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c MakeTextEditable) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, c.Target.MakeTextEditable)
}

type UpdateTextEditingMode struct {
	Target    *Text
	IsEditing bool
}

func (c UpdateTextEditingMode) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c UpdateTextEditingMode) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.SetTextEditing(c.IsEditing) })
}

// AddShape adds Shape to the group of its parent id, or to the root if it has none.
type AddShape struct {
	Shape Shape
}

func (c AddShape) affectedParent(manager *Manager) *Group {
	return manager.group(c.Shape.ParentID())
}

func (c AddShape) execute(manager *Manager, parent *Group) {
	parent.add(c.Shape)
	c.Shape.setParentID(parent.id)
	manager.register(c.Shape)
}

// RemoveShape removes Shape. A group left with one item is ungrouped, and an empty group
// is removed.
type RemoveShape struct {
	Shape Shape
}

func (c RemoveShape) affectedParent(manager *Manager) *Group {
	return manager.group(c.Shape.ParentID())
}

func (c RemoveShape) execute(manager *Manager, parent *Group) {
	parent.remove(c.Shape)
	manager.unregister(c.Shape)

	if parent == manager.root {
		return
	}
	switch parent.ItemCount() {
	case 1:
		manager.Execute(Ungroup{Group: parent})
	case 0:
		manager.Execute(RemoveShape{Shape: parent})
	}
}

// GroupShapes groups at least 2 shapes that have the same parent.
type GroupShapes struct {
	Shapes []Shape
}

func (c GroupShapes) affectedParent(manager *Manager) *Group {
	if len(c.Shapes) < 2 {
		return nil
	}
	parentID := c.Shapes[0].ParentID()
	for _, shape := range c.Shapes {
		if shape.ParentID() != parentID {
			// Shapes in different groups cannot be grouped.
			return nil
		}
	}
	return manager.group(parentID)
}

func (c GroupShapes) execute(manager *Manager, parent *Group) {
	group := NewGroup("", parent.id)
	parent.addAfter(group, c.Shapes[len(c.Shapes)-1].ID())
	manager.register(group)

	for _, shape := range c.Shapes {
		parent.remove(shape)
		shape.setParentID(group.id)
		group.add(shape)
	}
}

// Ungroup moves the items of Group to its parent and removes Group.
type Ungroup struct {
	Group *Group
}

func (c Ungroup) affectedParent(manager *Manager) *Group {
	return manager.group(c.Group.ParentID())
}

func (c Ungroup) execute(manager *Manager, parent *Group) {
	items := c.Group.Items()
	for i := len(items) - 1; i >= 0; i-- {
		c.Group.remove(items[i])
		items[i].setParentID("")
		parent.addAfter(items[i], c.Group.id)
	}
	manager.Execute(RemoveShape{Shape: c.Group})
}

type OrderType int

const (
	OrderForward OrderType = iota
	OrderBackward
	OrderFront
	OrderBack
)

func (t OrderType) MarshalText() ([]byte, error) {
	switch t {
	case OrderForward:
		return []byte("forward"), nil
	case OrderBackward:
		return []byte("backward"), nil
	case OrderFront:
		return []byte("front"), nil
	case OrderBack:
		return []byte("back"), nil
	}
	return nil, fmt.Errorf("unknown order type: %d", t)
}

func (t *OrderType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "forward":
		*t = OrderForward
	case "backward":
		*t = OrderBackward
	case "front":
		*t = OrderFront
	case "back":
		*t = OrderBack
	default:
		return fmt.Errorf("unknown order type: %q", text)
	}
	return nil
}

type ChangeOrder struct {
	Shape     Shape
	OrderType OrderType
}

func (c ChangeOrder) affectedParent(manager *Manager) *Group {
	return manager.group(c.Shape.ParentID())
}

func (c ChangeOrder) execute(manager *Manager, parent *Group) {
	actions := map[OrderType]moveAction{
		OrderForward:  moveUp,
		OrderBackward: moveDown,
		OrderFront:    moveTop,
		OrderBack:     moveBottom,
	}
	parent.changeOrder(c.Shape, actions[c.OrderType])
}

// Rename sets the name shown in the layers. An empty name shows the default name.
type Rename struct {
	Target Shape
	Name   string
}

func (c Rename) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c Rename) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.setName(c.Name) })
}

// SetHidden hides or shows a shape. A hidden shape is not drawn, exported or found by the
// mouse. See ADR-0013.
type SetHidden struct {
	Target   Shape
	IsHidden bool
}

func (c SetHidden) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c SetHidden) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.setHidden(c.IsHidden) })
}

// SetLocked locks or unlocks a shape. A locked shape cannot be selected or changed on the
// canvas, but can be in the layers. See ADR-0013.
type SetLocked struct {
	Target   Shape
	IsLocked bool
}

func (c SetLocked) affectedParent(manager *Manager) *Group {
	return manager.group(c.Target.ParentID())
}

func (c SetLocked) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.setLocked(c.IsLocked) })
}

// MoveShape moves Shape into NewParent, right above the item with BelowID, or to the bottom
// if BelowID is empty. A shape cannot move into itself or its own items. The old group is
// ungrouped or removed if it is left with one item or none, as RemoveShape does.
type MoveShape struct {
	Shape     Shape
	NewParent *Group
	BelowID   string
}

func (c MoveShape) affectedParent(manager *Manager) *Group {
	if c.BelowID == c.Shape.ID() {
		return nil
	}
	if c.BelowID != "" {
		if _, ok := c.NewParent.shapes.get(c.BelowID); !ok {
			return nil
		}
	}
	for group := c.NewParent; group != nil; group = manager.group(group.ParentID()) {
		if group.ID() == c.Shape.ID() {
			return nil
		}
		if group.ParentID() == "" {
			break
		}
	}
	return manager.group(c.Shape.ParentID())
}

func (c MoveShape) execute(manager *Manager, oldParent *Group) {
	oldParent.remove(c.Shape)
	c.Shape.setParentID("")
	c.NewParent.update(func() bool {
		c.Shape.setParentID(c.NewParent.id)
		if c.BelowID == "" {
			return c.NewParent.shapes.addFirst(c.Shape)
		}
		return c.NewParent.shapes.addAfter(c.Shape, c.BelowID)
	})
	if oldParent != c.NewParent {
		// The new parent is not the affected parent, so its ancestors are updated here.
		for _, ancestor := range manager.ancestors(c.NewParent) {
			ancestor.update(func() bool { return true })
		}
	}

	if oldParent == manager.root || oldParent == c.NewParent {
		return
	}
	switch oldParent.ItemCount() {
	case 1:
		manager.Execute(Ungroup{Group: oldParent})
	case 0:
		manager.Execute(RemoveShape{Shape: oldParent})
	}
}
