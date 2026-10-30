// Package action defines the actions from the UI and keeps the current ones. JS converts
// keys and toolbar events into actions. See ADR-0005.
package action

import (
	"fmt"

	"github.com/bluemir/monosketch/internal/cursor"
	"github.com/bluemir/monosketch/internal/observable"
)

// Retainable is an action which keeps having effect after it is set, such as adding
// rectangles by mouse.
type Retainable int

const (
	RetainableIdle Retainable = iota
	AddRectangle
	AddText
	AddLine
)

func (r Retainable) Cursor() cursor.Cursor {
	switch r {
	case AddRectangle, AddLine:
		return cursor.Crosshair
	case AddText:
		return cursor.Text
	default:
		return cursor.Default
	}
}

func (r Retainable) MarshalText() ([]byte, error) {
	switch r {
	case RetainableIdle:
		return []byte("idle"), nil
	case AddRectangle:
		return []byte("addRectangle"), nil
	case AddText:
		return []byte("addText"), nil
	case AddLine:
		return []byte("addLine"), nil
	}
	return nil, fmt.Errorf("unknown retainable action: %d", r)
}

func (r *Retainable) UnmarshalText(text []byte) error {
	switch string(text) {
	case "idle":
		*r = RetainableIdle
	case "addRectangle":
		*r = AddRectangle
	case "addText":
		*r = AddText
	case "addLine":
		*r = AddLine
	default:
		return fmt.Errorf("unknown retainable action: %q", text)
	}
	return nil
}

type Manager struct {
	Retainable *observable.Value[Retainable]
	// OneTime goes back to Idle right after notifying an action.
	OneTime *observable.Value[OneTime]
}

func NewManager() *Manager {
	return &Manager{
		Retainable: observable.New(RetainableIdle),
		OneTime:    observable.New[OneTime](Idle{}),
	}
}

func (m *Manager) SetRetainable(action Retainable) {
	m.Retainable.Set(action)
}

func (m *Manager) SetOneTime(action OneTime) {
	m.OneTime.Set(action)
	m.OneTime.Set(Idle{})
}
