package shape

import "github.com/bluemir/nemokan/internal/observable"

// Extra is the shape specific attributes: RectangleExtra, TextExtra or LineExtra.
type Extra interface {
	isExtra()
}

// RectangleExtra keeps the selected styles even while fill or border is disabled, so
// enabling it again brings back the last selection.
type RectangleExtra struct {
	IsFillEnabled           bool
	UserSelectedFillStyle   *RectangleFillStyle
	IsBorderEnabled         bool
	UserSelectedBorderStyle *StraightStrokeStyle
	DashPattern             StraightStrokeDashPattern
	Corner                  RectangleBorderCornerPattern
}

func (RectangleExtra) isExtra() {}

func (e RectangleExtra) IsRoundedCorner() bool {
	return e.Corner == CornerRounded
}

// FillStyle returns nil if fill is disabled.
func (e RectangleExtra) FillStyle() *RectangleFillStyle {
	if !e.IsFillEnabled {
		return nil
	}
	return e.UserSelectedFillStyle
}

// StrokeStyle returns nil if border is disabled.
func (e RectangleExtra) StrokeStyle() *StraightStrokeStyle {
	if !e.IsBorderEnabled {
		return nil
	}
	if e.IsRoundedCorner() {
		return RoundedStrokeStyle(e.UserSelectedBorderStyle.ID)
	}
	return strokeStyle(e.UserSelectedBorderStyle.ID)
}

type TextExtra struct {
	BoundExtra RectangleExtra
	TextAlign  TextAlign
}

func (TextExtra) isExtra() {}

func (e TextExtra) HasBorder() bool {
	return e.BoundExtra.IsBorderEnabled
}

type LineExtra struct {
	IsStrokeEnabled         bool
	UserSelectedStrokeStyle *StraightStrokeStyle
	IsStartAnchorEnabled    bool
	UserSelectedStartAnchor *AnchorChar
	IsEndAnchorEnabled      bool
	UserSelectedEndAnchor   *AnchorChar
	DashPattern             StraightStrokeDashPattern
	IsRoundedCorner         bool
}

func (LineExtra) isExtra() {}

// StartAnchor returns nil if the start anchor is disabled.
func (e LineExtra) StartAnchor() *AnchorChar {
	if !e.IsStartAnchorEnabled {
		return nil
	}
	return e.UserSelectedStartAnchor
}

// EndAnchor returns nil if the end anchor is disabled.
func (e LineExtra) EndAnchor() *AnchorChar {
	if !e.IsEndAnchorEnabled {
		return nil
	}
	return e.UserSelectedEndAnchor
}

// StrokeStyle returns nil if stroke is disabled.
func (e LineExtra) StrokeStyle() *StraightStrokeStyle {
	if !e.IsStrokeEnabled {
		return nil
	}
	if e.IsRoundedCorner {
		return RoundedStrokeStyle(e.UserSelectedStrokeStyle.ID)
	}
	return strokeStyle(e.UserSelectedStrokeStyle.ID)
}

// Defaults are used for new shapes. The user changes them with the shape tool.
var (
	defaultRectangleExtra = RectangleExtra{
		IsFillEnabled:           false,
		UserSelectedFillStyle:   predefinedFillStyles[0],
		IsBorderEnabled:         true,
		UserSelectedBorderStyle: predefinedStrokeStyles[0],
		DashPattern:             SolidDashPattern,
		Corner:                  CornerNotRounded,
	}

	defaultLineExtra = LineExtra{
		IsStrokeEnabled:         true,
		UserSelectedStrokeStyle: predefinedStrokeStyles[0],
		IsStartAnchorEnabled:    false,
		UserSelectedStartAnchor: predefinedAnchorChars[0],
		IsEndAnchorEnabled:      false,
		UserSelectedEndAnchor:   predefinedAnchorChars[0],
		DashPattern:             SolidDashPattern,
		IsRoundedCorner:         false,
	}

	defaultTextAlign = TextAlign{Horizontal: AlignCenter, Vertical: AlignMiddle}

	// DefaultExtraUpdated is notified after SetDefaultValues.
	DefaultExtraUpdated = observable.New(struct{}{})
)

// TextExtraNoBound has neither fill nor border, and aligns text to the top left.
var TextExtraNoBound = func() TextExtra {
	boundExtra := defaultRectangleExtra
	boundExtra.IsFillEnabled = false
	boundExtra.IsBorderEnabled = false
	return TextExtra{BoundExtra: boundExtra, TextAlign: TextAlign{Horizontal: AlignLeft, Vertical: AlignTop}}
}()

func DefaultRectangleExtra() RectangleExtra { return defaultRectangleExtra }
func DefaultLineExtra() LineExtra           { return defaultLineExtra }
func DefaultTextAlign() TextAlign           { return defaultTextAlign }

func DefaultTextExtra() TextExtra {
	return TextExtra{BoundExtra: defaultRectangleExtra, TextAlign: defaultTextAlign}
}

// DefaultValuesUpdate changes only the non-nil fields.
type DefaultValuesUpdate struct {
	IsFillEnabled *bool
	FillStyleID   *string

	IsBorderEnabled       *bool
	BorderStyleID         *string
	IsBorderRoundedCorner *bool
	BorderDashPattern     *StraightStrokeDashPattern

	IsLineStrokeEnabled       *bool
	LineStrokeStyleID         *string
	IsLineStrokeRoundedCorner *bool
	LineDashPattern           *StraightStrokeDashPattern

	IsStartAnchorEnabled *bool
	StartAnchorID        *string
	IsEndAnchorEnabled   *bool
	EndAnchorID          *string

	TextHorizontalAlign *HorizontalAlign
	TextVerticalAlign   *VerticalAlign
}

func SetDefaultValues(update DefaultValuesUpdate) {
	rectangle := defaultRectangleExtra
	rectangle.IsFillEnabled = valueOr(update.IsFillEnabled, rectangle.IsFillEnabled)
	rectangle.UserSelectedFillStyle = RectangleFillStyleOr(valueOr(update.FillStyleID, ""), rectangle.UserSelectedFillStyle)
	rectangle.IsBorderEnabled = valueOr(update.IsBorderEnabled, rectangle.IsBorderEnabled)
	rectangle.UserSelectedBorderStyle = StrokeStyleOr(valueOr(update.BorderStyleID, ""), rectangle.UserSelectedBorderStyle)
	rectangle.DashPattern = valueOr(update.BorderDashPattern, rectangle.DashPattern)
	rectangle.Corner = CornerNotRounded
	if valueOr(update.IsBorderRoundedCorner, rectangle.IsRoundedCorner()) {
		rectangle.Corner = CornerRounded
	}
	defaultRectangleExtra = rectangle

	line := defaultLineExtra
	line.IsStrokeEnabled = valueOr(update.IsLineStrokeEnabled, line.IsStrokeEnabled)
	line.UserSelectedStrokeStyle = StrokeStyleOr(valueOr(update.LineStrokeStyleID, ""), line.UserSelectedStrokeStyle)
	line.IsStartAnchorEnabled = valueOr(update.IsStartAnchorEnabled, line.IsStartAnchorEnabled)
	line.UserSelectedStartAnchor = AnchorCharOr(valueOr(update.StartAnchorID, ""), line.UserSelectedStartAnchor)
	line.IsEndAnchorEnabled = valueOr(update.IsEndAnchorEnabled, line.IsEndAnchorEnabled)
	line.UserSelectedEndAnchor = AnchorCharOr(valueOr(update.EndAnchorID, ""), line.UserSelectedEndAnchor)
	// The original resets the line dash pattern to solid when it is not given.
	line.DashPattern = valueOr(update.LineDashPattern, SolidDashPattern)
	line.IsRoundedCorner = valueOr(update.IsLineStrokeRoundedCorner, line.IsRoundedCorner)
	defaultLineExtra = line

	defaultTextAlign = TextAlign{
		Horizontal: valueOr(update.TextHorizontalAlign, defaultTextAlign.Horizontal),
		Vertical:   valueOr(update.TextVerticalAlign, defaultTextAlign.Vertical),
	}

	DefaultExtraUpdated.Set(struct{}{})
}

func valueOr[T any](value *T, fallback T) T {
	if value == nil {
		return fallback
	}
	return *value
}
