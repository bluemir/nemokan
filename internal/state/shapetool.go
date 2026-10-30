package state

import (
	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// ShapeTool is what the shape tool panel shows. It is the Go port of the original
// ShapeToolViewModel. A nil field hides its section.
//
// The appearance and text alignment come from the single selected shape. If there is none
// of the kind, they come from the default extras while the matching shape is being added.
type ShapeTool struct {
	// Bound is the bound of the single selected shape. Reorder and transform need it.
	Bound        *geo.Rect `json:"bound"`
	IsResizeable bool      `json:"isResizeable"`

	Rectangle *RectangleAppearance `json:"rectangle"`
	Line      *LineAppearance      `json:"line"`
	TextAlign *shape.TextAlign     `json:"textAlign"`
}

// StyleSelection is a style that can be turned off while keeping the selected one.
type StyleSelection struct {
	IsEnabled  bool   `json:"isEnabled"`
	SelectedID string `json:"selectedId"`
}

type RectangleAppearance struct {
	Fill        StyleSelection                  `json:"fill"`
	Border      StyleSelection                  `json:"border"`
	DashPattern shape.StraightStrokeDashPattern `json:"dashPattern"`
	// IsRoundedCorner is nil if the border cannot have rounded corners now.
	IsRoundedCorner *bool `json:"isRoundedCorner"`
}

type LineAppearance struct {
	Stroke      StyleSelection                  `json:"stroke"`
	DashPattern shape.StraightStrokeDashPattern `json:"dashPattern"`
	// IsRoundedCorner is nil if the stroke cannot have rounded corners now.
	IsRoundedCorner *bool          `json:"isRoundedCorner"`
	StartHead       StyleSelection `json:"startHead"`
	EndHead         StyleSelection `json:"endHead"`
}

func (m *Manager) ShapeTool() ShapeTool {
	if m.access == AccessView {
		return ShapeTool{}
	}
	retainable := m.actions.Retainable.Value()
	selected, isSingle := m.singleSelectedShape()

	var tool ShapeTool
	if isSingle {
		bound := selected.Bound()
		tool.Bound = &bound
	}

	switch selected := selected.(type) {
	case *shape.Rectangle:
		tool.IsResizeable = true
		tool.Rectangle = newRectangleAppearance(selected.Extra())
	case *shape.Text:
		tool.IsResizeable = true
		tool.Rectangle = newRectangleAppearance(selected.Extra().BoundExtra)
		if selected.IsTextEditable() {
			textAlign := selected.Extra().TextAlign
			tool.TextAlign = &textAlign
		}
	case *shape.Line:
		tool.Line = newLineAppearance(selected.Extra())
	}

	if tool.Rectangle == nil && (retainable == action.AddRectangle || retainable == action.AddText) {
		tool.Rectangle = newRectangleAppearance(shape.DefaultRectangleExtra())
	}
	if tool.Line == nil && retainable == action.AddLine {
		tool.Line = newLineAppearance(shape.DefaultLineExtra())
	}
	if tool.TextAlign == nil && retainable == action.AddText {
		textAlign := shape.DefaultTextAlign()
		tool.TextAlign = &textAlign
	}
	return tool
}

func newRectangleAppearance(extra shape.RectangleExtra) *RectangleAppearance {
	border := StyleSelection{IsEnabled: extra.IsBorderEnabled, SelectedID: extra.UserSelectedBorderStyle.ID}
	return &RectangleAppearance{
		Fill:            StyleSelection{IsEnabled: extra.IsFillEnabled, SelectedID: extra.UserSelectedFillStyle.ID},
		Border:          border,
		DashPattern:     extra.DashPattern,
		IsRoundedCorner: roundedCorner(border, extra.IsRoundedCorner()),
	}
}

func newLineAppearance(extra shape.LineExtra) *LineAppearance {
	stroke := StyleSelection{IsEnabled: extra.IsStrokeEnabled, SelectedID: extra.UserSelectedStrokeStyle.ID}
	return &LineAppearance{
		Stroke:          stroke,
		DashPattern:     extra.DashPattern,
		IsRoundedCorner: roundedCorner(stroke, extra.IsRoundedCorner),
		StartHead:       StyleSelection{IsEnabled: extra.IsStartAnchorEnabled, SelectedID: extra.UserSelectedStartAnchor.ID},
		EndHead:         StyleSelection{IsEnabled: extra.IsEndAnchorEnabled, SelectedID: extra.UserSelectedEndAnchor.ID},
	}
}

// roundedCorner returns nil if the stroke is off or its style has no rounded version, as
// the original hides the rounded corner button then.
func roundedCorner(stroke StyleSelection, isRoundedCorner bool) *bool {
	if !stroke.IsEnabled || !shape.IsCornerRoundable(stroke.SelectedID) {
		return nil
	}
	return &isRoundedCorner
}
