// Package selection manages the selected shapes and the focusing shape.
package selection

import (
	"slices"

	"github.com/bluemir/monosketch/internal/observable"
	"github.com/bluemir/monosketch/internal/shape"
)

// FocusType is why a shape is focused.
type FocusType int

const (
	FocusLineConnecting FocusType = iota
	FocusSelectModeHover
)

type FocusingShape struct {
	Shape     shape.Shape
	FocusType FocusType
}

type Manager struct {
	// SelectedShapes keeps each shape once, in the order they are selected.
	SelectedShapes *observable.Value[[]shape.Shape]
	// FocusingShape is nil if no shape is focused.
	FocusingShape *observable.Value[*FocusingShape]
}

func New() *Manager {
	return &Manager{
		SelectedShapes: observable.New[[]shape.Shape](nil),
		FocusingShape:  observable.New[*FocusingShape](nil),
	}
}

func (m *Manager) Add(target shape.Shape) {
	selected := m.SelectedShapes.Value()
	if slices.Contains(selected, target) {
		m.SelectedShapes.Set(selected)
		return
	}
	m.SelectedShapes.Set(append(slices.Clone(selected), target))
}

func (m *Manager) Toggle(target shape.Shape) {
	selected := m.SelectedShapes.Value()
	if slices.Contains(selected, target) {
		m.SelectedShapes.Set(slices.DeleteFunc(slices.Clone(selected), func(item shape.Shape) bool { return item == target }))
		return
	}
	m.SelectedShapes.Set(append(slices.Clone(selected), target))
}

func (m *Manager) Clear() {
	m.SelectedShapes.Set(nil)
}

// SetFocusingShape focuses target. A nil target clears the focus.
func (m *Manager) SetFocusingShape(target shape.Shape, focusType FocusType) {
	if target == nil {
		m.FocusingShape.Set(nil)
		return
	}
	m.FocusingShape.Set(&FocusingShape{Shape: target, FocusType: focusType})
}

// FocusType returns the focus type of target if target is the focusing shape.
func (m *Manager) FocusType(target shape.Shape) (FocusType, bool) {
	focusing := m.FocusingShape.Value()
	if focusing == nil || focusing.Shape != target {
		return 0, false
	}
	return focusing.FocusType, true
}
