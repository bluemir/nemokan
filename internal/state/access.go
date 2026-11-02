package state

import (
	"fmt"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/shape"
)

// Access is what the user can do with the shapes. A server document the user cannot edit is
// opened with AccessView. See ADR-0012.
type Access int

const (
	AccessEdit Access = iota
	// AccessView only allows selecting and copying. Nothing changes the shapes.
	AccessView
)

func (a Access) MarshalText() ([]byte, error) {
	switch a {
	case AccessEdit:
		return []byte("edit"), nil
	case AccessView:
		return []byte("view"), nil
	}
	return nil, fmt.Errorf("unknown access: %d", a)
}

func (a *Access) UnmarshalText(text []byte) error {
	switch string(text) {
	case "edit":
		*a = AccessEdit
	case "view":
		*a = AccessView
	default:
		return fmt.Errorf("unknown access: %q", text)
	}
	return nil
}

func (m *Manager) Access() Access { return m.access }

// SetAccess also stops adding shapes and hides the handles for changing shapes, which
// AccessView does not allow.
func (m *Manager) SetAccess(access Access) {
	m.access = access
	if access == AccessView {
		m.actions.SetRetainable(action.RetainableIdle)
	}
	m.refreshInteractionBounds()
	m.notify(ChangeShapeTool)
}

// isAllowed reports whether a one time action can run with the current access.
func (m *Manager) isAllowed(oneTime action.OneTime) bool {
	if m.access == AccessEdit {
		return true
	}
	switch oneTime.(type) {
	case action.SelectAllShapes, action.DeselectShapes, action.SelectShape, action.ToggleShapeSelection, action.Copy, action.CopyText:
		return true
	}
	return false
}

// interactionBoundShapes are the shapes which show handles for changing them.
func (m *Manager) interactionBoundShapes(selectedShapes []shape.Shape) []shape.Shape {
	if m.access == AccessView {
		return nil
	}
	return selectedShapes
}
