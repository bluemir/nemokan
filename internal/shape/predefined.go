package shape

import "github.com/bluemir/nemokan/internal/bitmap"

var predefinedAnchorChars = []*AnchorChar{
	{ID: "A1", DisplayName: "▶", Left: '◀', Right: '▶', Top: '▲', Bottom: '▼'},
	{ID: "A12", DisplayName: "▷", Left: '◁', Right: '▷', Top: '△', Bottom: '▽'},
	{ID: "A13", DisplayName: "►", Left: '◄', Right: '►', Top: '▲', Bottom: '▼'},
	{ID: "A14", DisplayName: "▻", Left: '◅', Right: '▻', Top: '△', Bottom: '▽'},
	{ID: "A2", DisplayName: "■", Left: '■', Right: '■', Top: '■', Bottom: '■'},
	{ID: "A21", DisplayName: "□", Left: '□', Right: '□', Top: '□', Bottom: '□'},
	{ID: "A220", DisplayName: "◆", Left: '◆', Right: '◆', Top: '◆', Bottom: '◆'},
	{ID: "A221", DisplayName: "◇", Left: '◇', Right: '◇', Top: '◇', Bottom: '◇'},
	{ID: "A3", DisplayName: "○", Left: '○', Right: '○', Top: '○', Bottom: '○'},
	{ID: "A4", DisplayName: "◎", Left: '◎', Right: '◎', Top: '◎', Bottom: '◎'},
	{ID: "A5", DisplayName: "●", Left: '●', Right: '●', Top: '●', Bottom: '●'},
	{ID: "A6", DisplayName: "├", Left: '├', Right: '┤', Top: '┬', Bottom: '┴'},
	{ID: "A61", DisplayName: "┣", Left: '┣', Right: '┫', Top: '┳', Bottom: '┻'},
	{ID: "A62", DisplayName: "╠", Left: '╠', Right: '╣', Top: '╦', Bottom: '╩'},
}

var predefinedFillStyles = []*RectangleFillStyle{
	// NBSP shows an empty looking but selectable name on the UI.
	{ID: "F1", DisplayName: " ", Drawable: bitmap.CharDrawable{Char: ' '}},
	{ID: "F2", DisplayName: "█", Drawable: bitmap.CharDrawable{Char: '█'}},
	{ID: "F3", DisplayName: "▒", Drawable: bitmap.CharDrawable{Char: '▒'}},
	{ID: "F4", DisplayName: "░", Drawable: bitmap.CharDrawable{Char: '░'}},
	{ID: "F5", DisplayName: "▚", Drawable: bitmap.CharDrawable{Char: '▚'}},
}

var noStroke = &StraightStrokeStyle{
	ID:          "S0",
	DisplayName: "No Stroke",
	Horizontal:  bitmap.HalfTransparent,
	Vertical:    bitmap.HalfTransparent,
	DownLeft:    bitmap.HalfTransparent,
	UpRight:     bitmap.HalfTransparent,
	UpLeft:      bitmap.HalfTransparent,
	DownRight:   bitmap.HalfTransparent,

	VerticalRight:  bitmap.HalfTransparent,
	VerticalLeft:   bitmap.HalfTransparent,
	HorizontalDown: bitmap.HalfTransparent,
	HorizontalUp:   bitmap.HalfTransparent,
	Cross:          bitmap.HalfTransparent,
}

var allStrokeStyles = []*StraightStrokeStyle{
	noStroke,
	{ID: "S1", DisplayName: "─", Horizontal: '─', Vertical: '│', DownLeft: '┐', UpRight: '┌', UpLeft: '┘', DownRight: '└',
		VerticalRight: '├', VerticalLeft: '┤', HorizontalDown: '┬', HorizontalUp: '┴', Cross: '┼'},
	{ID: "S2", DisplayName: "━", Horizontal: '━', Vertical: '┃', DownLeft: '┓', UpRight: '┏', UpLeft: '┛', DownRight: '┗',
		VerticalRight: '┣', VerticalLeft: '┫', HorizontalDown: '┳', HorizontalUp: '┻', Cross: '╋'},
	{ID: "S3", DisplayName: "═", Horizontal: '═', Vertical: '║', DownLeft: '╗', UpRight: '╔', UpLeft: '╝', DownRight: '╚',
		VerticalRight: '╠', VerticalLeft: '╣', HorizontalDown: '╦', HorizontalUp: '╩', Cross: '╬'},
	{ID: "S4", DisplayName: "▢", Horizontal: '─', Vertical: '│', DownLeft: '╮', UpRight: '╭', UpLeft: '╯', DownRight: '╰',
		VerticalRight: '├', VerticalLeft: '┤', HorizontalDown: '┬', HorizontalUp: '┴', Cross: '┼'},
}

// predefinedStrokeStyles are the styles a user can select. S0 and S4 are used internally.
var predefinedStrokeStyles = []*StraightStrokeStyle{allStrokeStyles[1], allStrokeStyles[2], allStrokeStyles[3]}

// roundedStrokeStyleIDs maps a stroke style to its rounded corner version.
var roundedStrokeStyleIDs = map[string]string{"S1": "S4"}

func PredefinedAnchorChars() []*AnchorChar                 { return predefinedAnchorChars }
func PredefinedRectangleFillStyles() []*RectangleFillStyle { return predefinedFillStyles }
func PredefinedStrokeStyles() []*StraightStrokeStyle       { return predefinedStrokeStyles }

// NoStrokeStyle draws half transparent chars, which are not displayed but keep a shape selectable.
func NoStrokeStyle() *StraightStrokeStyle { return noStroke }

// AnchorCharOr returns the predefined anchor char with id, or fallback if there is none.
func AnchorCharOr(id string, fallback *AnchorChar) *AnchorChar {
	for _, anchor := range predefinedAnchorChars {
		if anchor.ID == id {
			return anchor
		}
	}
	return fallback
}

// RectangleFillStyleOr returns the predefined fill style with id, or fallback if there is none.
func RectangleFillStyleOr(id string, fallback *RectangleFillStyle) *RectangleFillStyle {
	for _, style := range predefinedFillStyles {
		if style.ID == id {
			return style
		}
	}
	return fallback
}

// StrokeStyleOr returns the stroke style with id, or fallback if there is none.
func StrokeStyleOr(id string, fallback *StraightStrokeStyle) *StraightStrokeStyle {
	if style := strokeStyle(id); style != nil {
		return style
	}
	return fallback
}

// RoundedStrokeStyle returns the rounded corner version of the style with id if there is one,
// otherwise the style itself.
func RoundedStrokeStyle(id string) *StraightStrokeStyle {
	if roundedID, ok := roundedStrokeStyleIDs[id]; ok {
		return strokeStyle(roundedID)
	}
	return strokeStyle(id)
}

func IsCornerRoundable(id string) bool {
	_, ok := roundedStrokeStyleIDs[id]
	return ok
}

// strokeStyle returns nil if no style has id.
func strokeStyle(id string) *StraightStrokeStyle {
	for _, style := range allStrokeStyles {
		if style.ID == id {
			return style
		}
	}
	return nil
}
