package shape

import (
	"fmt"

	"github.com/bluemir/monosketch/internal/geo"
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
	// TODO: This hasn't cleaned group's items. Fix this
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
