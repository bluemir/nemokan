package shape

import (
	"encoding/json"
	"fmt"

	"github.com/bluemir/monosketch/internal/geo"
)

// Widget is a UI element of a wireframe, such as a button or a table. What it shows comes
// from its content, one kind of WidgetContent. See ADR-0014.
type Widget struct {
	base
	bound   geo.Rect
	content WidgetContent
	// extra is the border and the fill of a widget drawn in a box, such as a card.
	extra RectangleExtra
}

// WidgetKind names a kind of widget, such as "button". It is also the kind in files.
type WidgetKind string

// WidgetContent is what one kind of widget shows. Each kind is a struct of its own, and the
// JSON of the struct is kept in files.
type WidgetContent interface {
	Kind() WidgetKind
	// fit returns the size the widget takes when the user asks for size. Some kinds only
	// change the width, and some take the size of what they show.
	fit(size geo.Size) geo.Size
	// layout returns what the widget shows in size, which fit returned.
	layout(size geo.Size) WidgetLayout
	// withText returns the content with the text of field changed.
	withText(field WidgetField, text string) WidgetContent
}

// widgetToggle is a content that can be turned on and off, such as a checkbox.
type widgetToggle interface {
	toggled() WidgetContent
}

// WidgetField is a text of a content the user edits. Index is the item of a list, such as a
// column of a table.
type WidgetField struct {
	Name  string
	Index int
}

// widgetToggleField is the mark of a toggle, such as [x] of a checkbox. Double clicking it
// turns the toggle on or off.
var widgetToggleField = WidgetField{Name: "toggle"}

// WidgetLayout is what a widget shows, relative to its top-left.
type WidgetLayout struct {
	// HasBox draws the bound as a rectangle with the extra of the widget.
	HasBox bool
	// DividerRows and DividerColumns are lines inside the box, from border to border.
	DividerRows    []int
	DividerColumns []int
	Runs           []WidgetRun
}

// WidgetRun is a text drawn at Row and Column. A run with Field is a text the user edits,
// which takes Width cells. Its Value is the whole text, as Text may be cut to fit.
type WidgetRun struct {
	Row    int
	Column int
	Text   string
	Field  *WidgetField
	Width  int
	Value  string
}

// IsToggle reports whether the run is the mark of a toggle.
func (r WidgetRun) IsToggle() bool {
	return r.Field != nil && *r.Field == widgetToggleField
}

// widgetKinds creates a content of each kind with its default values.
var widgetKinds = map[WidgetKind]func() WidgetContent{}

// registerWidgetKind adds a kind. It is called by each kind in init.
func registerWidgetKind(newContent func() WidgetContent) {
	widgetKinds[newContent().Kind()] = newContent
}

// NewWidgetContent returns the content of kind with its default values, or false for an
// unknown kind.
func NewWidgetContent(kind WidgetKind) (WidgetContent, bool) {
	newContent, ok := widgetKinds[kind]
	if !ok {
		return nil, false
	}
	return newContent(), true
}

// ParseWidgetContent reads the JSON of a content of kind. Missing values keep the defaults.
func ParseWidgetContent(kind WidgetKind, data []byte) (WidgetContent, error) {
	newContent, ok := widgetKinds[kind]
	if !ok {
		return nil, fmt.Errorf("unknown widget kind: %q", kind)
	}
	content := newContent()
	// Each kind is a pointer to its struct, so the defaults are overwritten in place.
	if err := json.Unmarshal(data, content); err != nil {
		return nil, err
	}
	return content, nil
}

// defaultWidgetExtra draws the box of a widget with a thin border and fills it with spaces,
// so the inside covers what is below and is selectable.
var defaultWidgetExtra = RectangleExtra{
	IsFillEnabled:           true,
	UserSelectedFillStyle:   predefinedFillStyles[0],
	IsBorderEnabled:         true,
	UserSelectedBorderStyle: predefinedStrokeStyles[0],
	DashPattern:             SolidDashPattern,
	Corner:                  CornerNotRounded,
}

// NewWidget makes a widget at position with the default size of content. It generates an id
// if id is empty.
func NewWidget(position geo.Point, content WidgetContent, id, parentID string) *Widget {
	size := content.fit(geo.Size{})
	return &Widget{
		base:    newBase(id, parentID),
		bound:   geo.Rect{Position: position, Size: size},
		content: content,
		extra:   defaultWidgetExtra,
	}
}

func restoreWidget(serializable *SerializableWidget, parentID string) *Widget {
	content, err := ParseWidgetContent(serializable.Kind, serializable.Content)
	if err != nil {
		// ParseFile checks the content, so this is a programming error.
		panic(err)
	}
	widget := &Widget{
		base:    newBase(serializable.actualID(), parentID),
		bound:   geo.Rect{Position: serializable.Bound.Position, Size: content.fit(serializable.Bound.Size)},
		content: content,
		extra:   serializable.Extra.restore(),
	}
	widget.restore(serializable.serializableBase)
	return widget
}

func (w *Widget) Bound() geo.Rect        { return w.bound }
func (w *Widget) Content() WidgetContent { return w.content }
func (w *Widget) Extra() RectangleExtra  { return w.extra }
func (w *Widget) Layout() WidgetLayout   { return w.content.layout(w.bound.Size) }

// HasBox reports whether the widget is drawn in a box, which has a border and a fill.
func (w *Widget) HasBox() bool { return w.Layout().HasBox }

// IsResizeable reports whether the user can change the size of the widget.
func (w *Widget) IsResizeable() bool {
	return w.content.fit(geo.Size{Width: 1000, Height: 1000}) != w.content.fit(geo.Size{})
}

// SetBound keeps the size the content fits in.
func (w *Widget) SetBound(newBound geo.Rect) {
	newBound.Size = w.content.fit(newBound.Size)
	w.update(func() bool {
		isUpdated := w.bound != newBound
		w.bound = newBound
		return isUpdated
	})
}

func (w *Widget) SetExtra(newExtra Extra) {
	extra, ok := newExtra.(RectangleExtra)
	if !ok {
		panic(fmt.Sprintf("new extra is not a RectangleExtra (%T)", newExtra))
	}
	w.update(func() bool {
		isUpdated := w.extra != extra
		w.extra = extra
		return isUpdated
	})
}

// setContent also fits the size to the new content, such as a longer label of a checkbox.
func (w *Widget) setContent(content WidgetContent) {
	w.update(func() bool {
		isUpdated := !SameWidgetContent(w.content, content)
		w.content = content
		w.bound.Size = content.fit(w.bound.Size)
		return isUpdated
	})
}

// SameWidgetContent reports whether two contents show the same.
func SameWidgetContent(content, other WidgetContent) bool {
	data, _ := json.Marshal(content)
	otherData, _ := json.Marshal(other)
	return content.Kind() == other.Kind() && string(data) == string(otherData)
}

// FieldAt returns the run with a field at point, relative to the top-left of the widget.
func (w *Widget) FieldAt(point geo.Point) (WidgetRun, bool) {
	for _, run := range w.Layout().Runs {
		if run.Field != nil && run.Row == point.Top && run.Column <= point.Left && point.Left < run.Column+run.Width {
			return run, true
		}
	}
	return WidgetRun{}, false
}

func (w *Widget) Contains(point geo.Point) bool   { return w.bound.Contains(point) }
func (w *Widget) IsVertex(point geo.Point) bool   { return w.bound.IsVertex(point) }
func (w *Widget) IsOverlapped(rect geo.Rect) bool { return w.bound.IsOverlapped(rect) }

func (w *Widget) Serializable(isIDIncluded bool) SerializableShape {
	content, err := json.Marshal(w.content)
	if err != nil {
		panic(err)
	}
	return &SerializableWidget{
		serializableBase: w.serializableBase(isIDIncluded),
		Bound:            w.bound,
		Kind:             w.content.Kind(),
		Content:          content,
		Extra:            serializableRectangleExtra(w.extra),
	}
}

// ChangeWidgetContent replaces the content of a widget with one of the same kind.
type ChangeWidgetContent struct {
	Target     *Widget
	NewContent WidgetContent
}

func (c ChangeWidgetContent) affectedParent(manager *Manager) *Group {
	if c.NewContent.Kind() != c.Target.content.Kind() {
		return nil
	}
	return manager.group(c.Target.ParentID())
}

func (c ChangeWidgetContent) execute(manager *Manager, parent *Group) {
	updateParentIfChanged(parent, c.Target, func() { c.Target.setContent(c.NewContent) })
}

// WidgetContentWithText returns the content of target with the text of field changed.
func WidgetContentWithText(target *Widget, field WidgetField, text string) WidgetContent {
	return target.content.withText(field, text)
}

// ToggledWidgetContent returns the content of target turned on or off, or false if it is not
// a toggle.
func ToggledWidgetContent(target *Widget) (WidgetContent, bool) {
	toggle, ok := target.content.(widgetToggle)
	if !ok {
		return nil, false
	}
	return toggle.toggled(), true
}
