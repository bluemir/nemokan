// Package action defines the actions from the UI and keeps the current ones. JS converts
// keys and toolbar events into actions. See ADR-0005.
package action

import (
	"fmt"
	"strings"

	"github.com/bluemir/nemokan/internal/cursor"
	"github.com/bluemir/nemokan/internal/observable"
	"github.com/bluemir/nemokan/internal/shape"
)

// Retainable is an action which keeps having effect after it is set, such as adding
// rectangles by mouse. Retainables are compared with ==.
type Retainable struct {
	kind retainableKind
	// widget is the kind of widget to add, for AddWidget.
	widget shape.WidgetKind
}

type retainableKind int

const (
	retainableIdle retainableKind = iota
	retainableAddRectangle
	retainableAddText
	retainableAddLine
	retainableAddWidget
)

var (
	RetainableIdle = Retainable{kind: retainableIdle}
	AddRectangle   = Retainable{kind: retainableAddRectangle}
	AddText        = Retainable{kind: retainableAddText}
	AddLine        = Retainable{kind: retainableAddLine}
)

// AddWidget adds widgets of kind by mouse. See ADR-0014.
func AddWidget(kind shape.WidgetKind) Retainable {
	return Retainable{kind: retainableAddWidget, widget: kind}
}

// AddingWidget returns the kind of widget the retainable adds, or false if it does not add
// widgets.
func (r Retainable) AddingWidget() (shape.WidgetKind, bool) {
	return r.widget, r.kind == retainableAddWidget
}

func (r Retainable) Cursor() cursor.Cursor {
	switch r.kind {
	case retainableAddRectangle, retainableAddLine, retainableAddWidget:
		return cursor.Crosshair
	case retainableAddText:
		return cursor.Text
	default:
		return cursor.Default
	}
}

// widgetPrefix starts the text of AddWidget, such as "widget:button".
const widgetPrefix = "widget:"

func (r Retainable) MarshalText() ([]byte, error) {
	switch r.kind {
	case retainableIdle:
		return []byte("idle"), nil
	case retainableAddRectangle:
		return []byte("addRectangle"), nil
	case retainableAddText:
		return []byte("addText"), nil
	case retainableAddLine:
		return []byte("addLine"), nil
	case retainableAddWidget:
		return []byte(widgetPrefix + string(r.widget)), nil
	}
	return nil, fmt.Errorf("unknown retainable action: %d", r.kind)
}

func (r *Retainable) UnmarshalText(text []byte) error {
	switch value := string(text); value {
	case "idle":
		*r = RetainableIdle
	case "addRectangle":
		*r = AddRectangle
	case "addText":
		*r = AddText
	case "addLine":
		*r = AddLine
	default:
		kind, ok := strings.CutPrefix(value, widgetPrefix)
		if _, isKnown := shape.NewWidgetContent(shape.WidgetKind(kind)); !ok || !isKnown {
			return fmt.Errorf("unknown retainable action: %q", text)
		}
		*r = AddWidget(shape.WidgetKind(kind))
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
