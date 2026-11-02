package state

import (
	"slices"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/shape"
)

// Groups and layers. See ADR-0013.
//
// The canvas works on the items of the working group. A click selects the item holding the
// shape under the mouse, so a group is selected as a whole. Double clicking a group makes it
// the working group, and clicking outside its items goes back to the root.

// Layer is one shape in the layers, with the items of a group from the top.
type Layer struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Kind is the kind of a widget.
	Kind shape.WidgetKind `json:"kind,omitempty"`
	// Name is empty if the user has not named the shape. Text is the text of a text shape,
	// for the default name.
	Name       string  `json:"name"`
	Text       string  `json:"text,omitempty"`
	IsHidden   bool    `json:"isHidden"`
	IsLocked   bool    `json:"isLocked"`
	IsSelected bool    `json:"isSelected"`
	Items      []Layer `json:"items,omitempty"`
}

// Layers is what the layers panel shows.
type Layers struct {
	// Items are the items of the root, from the top.
	Items []Layer `json:"items"`
	// WorkingGroupID is the group the canvas works on, or empty for the root.
	WorkingGroupID string `json:"workingGroupId"`
}

func (m *Manager) Layers() Layers {
	layers := Layers{Items: m.layers(m.shapeManager.Root())}
	if m.workingParentGroup != m.shapeManager.Root() {
		layers.WorkingGroupID = m.workingParentGroup.ID()
	}
	return layers
}

func (m *Manager) layers(group *shape.Group) []Layer {
	items := group.Items()
	result := make([]Layer, 0, len(items))
	selected := m.selection.SelectedShapes.Value()
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		layer := Layer{
			ID:         item.ID(),
			Name:       item.Name(),
			IsHidden:   item.IsHidden(),
			IsLocked:   item.IsLocked(),
			IsSelected: slices.Contains(selected, item),
		}
		switch item := item.(type) {
		case *shape.Rectangle:
			layer.Type = "rectangle"
		case *shape.Text:
			layer.Type = "text"
			if item.IsTextEditable() {
				layer.Text = item.Text()
			} else {
				// A rectangle drawn with the rectangle tool is a text which is not editable.
				layer.Type = "rectangle"
			}
		case *shape.Line:
			layer.Type = "line"
		case *shape.Widget:
			layer.Type = "widget"
			layer.Kind = item.Content().Kind()
		case *shape.Group:
			layer.Type = "group"
			layer.Items = m.layers(item)
		}
		result = append(result, layer)
	}
	return result
}

// parentGroup returns the group holding target, or nil for the root.
func (m *Manager) parentGroup(target shape.Shape) *shape.Group {
	if target.ParentID() == "" {
		return nil
	}
	found, _ := m.shapeManager.Shape(target.ParentID())
	group, _ := found.(*shape.Group)
	return group
}

// parentShape is parentGroup as a shape, which is nil for the root. A nil *shape.Group would
// not be a nil shape.Shape.
func (m *Manager) parentShape(target shape.Shape) shape.Shape {
	if group := m.parentGroup(target); group != nil {
		return group
	}
	return nil
}

// isHiddenInGroups reports whether target or a group holding it is hidden.
func (m *Manager) isHiddenInGroups(target shape.Shape) bool {
	for current := target; current != nil; current = m.parentShape(current) {
		if current.IsHidden() {
			return true
		}
	}
	return false
}

// isLockedInGroups reports whether target or a group holding it is locked.
func (m *Manager) isLockedInGroups(target shape.Shape) bool {
	for current := target; current != nil; current = m.parentShape(current) {
		if current.IsLocked() {
			return true
		}
	}
	return false
}

// isSelectedInGroups reports whether target or a group holding it is selected.
func (m *Manager) isSelectedInGroups(target shape.Shape) bool {
	selected := m.selection.SelectedShapes.Value()
	for current := target; current != nil; current = m.parentShape(current) {
		if slices.Contains(selected, current) {
			return true
		}
	}
	return false
}

// workingItem returns the item of the working group holding found, a shape the mouse found.
// It returns false if found is outside the working group, or if the item is locked.
func (m *Manager) workingItem(found shape.Shape) (shape.Shape, bool) {
	for current := found; current != nil; current = m.parentShape(current) {
		if current.IsLocked() {
			return nil, false
		}
		if current.ParentID() == m.workingParentGroup.ID() {
			return current, true
		}
	}
	return nil, false
}

// topWorkingItem returns the working item of the top shape in shapes, which are from the
// lowest z-index.
func (m *Manager) topWorkingItem(shapes []shape.Shape) (shape.Shape, bool) {
	for i := len(shapes) - 1; i >= 0; i-- {
		if item, ok := m.workingItem(shapes[i]); ok {
			return item, true
		}
	}
	return nil, false
}

func (m *Manager) setWorkingParentGroup(group *shape.Group) {
	if m.workingParentGroup == group {
		return
	}
	m.workingParentGroup = group
	m.notify(ChangeLayers)
}

// leaveRemovedWorkingGroup goes back to the root if the working group is no longer in the
// shapes, for example after ungrouping or undoing.
func (m *Manager) leaveRemovedWorkingGroup() {
	if _, ok := m.shapeManager.Shape(m.workingParentGroup.ID()); !ok {
		m.setWorkingParentGroup(m.shapeManager.Root())
	}
}

// leafShapes returns shapes with each group replaced by the shapes in it.
func leafShapes(shapes []shape.Shape) []shape.Shape {
	var result []shape.Shape
	for _, target := range shapes {
		if group, ok := target.(*shape.Group); ok {
			result = append(result, leafShapes(group.Items())...)
			continue
		}
		result = append(result, target)
	}
	return result
}

// movableSelectedShapes are the selected shapes the canvas and the arrow keys can move,
// which are not locked. A group moves the shapes in it.
func (m *Manager) movableSelectedShapes() []shape.Shape {
	var movable []shape.Shape
	for _, selected := range m.selection.SelectedShapes.Value() {
		if !m.isLockedInGroups(selected) {
			movable = append(movable, selected)
		}
	}
	return leafShapes(movable)
}

func (m *Manager) groupSelectedShapes() {
	selected := m.selectedShapesInOrder()
	if len(selected) < 2 {
		return
	}
	m.shapeManager.Execute(shape.GroupShapes{Shapes: selected})
	m.selection.Clear()
	m.addSelectedShape(m.parentGroup(selected[0]))
}

func (m *Manager) ungroupSelectedShapes() {
	var items []shape.Shape
	for _, selected := range m.selection.SelectedShapes.Value() {
		group, ok := selected.(*shape.Group)
		if !ok {
			continue
		}
		items = append(items, group.Items()...)
		m.shapeManager.Execute(shape.Ungroup{Group: group})
	}
	if len(items) == 0 {
		return
	}
	m.selection.Clear()
	for _, item := range items {
		m.addSelectedShape(item)
	}
}

// layerShape returns the shape with shapeID, but not the root.
func (m *Manager) layerShape(shapeID string) (shape.Shape, bool) {
	if shapeID == m.shapeManager.Root().ID() {
		return nil, false
	}
	return m.shapeManager.Shape(shapeID)
}

// selectLayer selects target from the layers. The canvas then works on the group holding it.
// A hidden shape is not selected, as it is not on the canvas.
func (m *Manager) selectLayer(target shape.Shape) {
	if m.isHiddenInGroups(target) {
		return
	}
	parent := m.parentGroup(target)
	if parent == nil {
		parent = m.shapeManager.Root()
	}
	m.setWorkingParentGroup(parent)
	m.selection.Clear()
	m.addSelectedShape(target)
}

// toggleLayerSelection adds target to the selection or removes it. Shapes in other groups
// are deselected, as the selection is always in one group.
func (m *Manager) toggleLayerSelection(target shape.Shape) {
	if m.isHiddenInGroups(target) {
		return
	}
	parent := m.parentGroup(target)
	if parent == nil {
		parent = m.shapeManager.Root()
	}
	if parent != m.workingParentGroup {
		m.selectLayer(target)
		return
	}
	m.selection.Toggle(target)
}

func (m *Manager) setShapeHidden(target shape.Shape, isHidden bool) {
	m.shapeManager.Execute(shape.SetHidden{Target: target, IsHidden: isHidden})
	if isHidden {
		m.deselectHiddenShapes()
	}
}

// deselectHiddenShapes deselects the shapes which are hidden now, as they are not on the
// canvas.
func (m *Manager) deselectHiddenShapes() {
	for _, selected := range m.selection.SelectedShapes.Value() {
		if m.isHiddenInGroups(selected) {
			m.selection.Toggle(selected)
		}
	}
}

func (m *Manager) setShapeLocked(target shape.Shape, isLocked bool) {
	m.shapeManager.Execute(shape.SetLocked{Target: target, IsLocked: isLocked})
	m.refreshInteractionBounds()
}

func (m *Manager) moveShapeInLayers(move action.MoveShapeInLayers) {
	target, ok := m.layerShape(move.ShapeID)
	if !ok {
		return
	}
	newParent := m.shapeManager.Root()
	if move.ParentID != "" {
		found, _ := m.shapeManager.Shape(move.ParentID)
		group, ok := found.(*shape.Group)
		if !ok {
			return
		}
		newParent = group
	}
	m.shapeManager.Execute(shape.MoveShape{Shape: target, NewParent: newParent, BelowID: move.BelowID})
	if target.ParentID() == newParent.ID() {
		m.selectLayer(target)
	}
}
