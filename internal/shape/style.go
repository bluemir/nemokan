package shape

import (
	"fmt"

	"github.com/bluemir/monosketch/internal/bitmap"
)

// AnchorChar is the char drawn at an end of a line.
// ID is the key to find the predefined AnchorChar when restoring.
type AnchorChar struct {
	ID          string
	DisplayName string
	Left        rune
	Right       rune
	Top         rune
	Bottom      rune
}

// RectangleFillStyle is how the inside of a rectangle is filled.
// ID is the key to find the predefined style when restoring.
type RectangleFillStyle struct {
	ID          string
	DisplayName string
	Drawable    bitmap.Drawable
}

// StraightStrokeStyle is the set of chars for drawing a straight line.
type StraightStrokeStyle struct {
	ID          string
	DisplayName string
	Horizontal  rune
	Vertical    rune
	DownLeft    rune
	UpRight     rune
	UpLeft      rune
	DownRight   rune

	// The junctions are where a divider meets the border or another divider, such as in a
	// table. Each is named by the directions it goes to.
	VerticalRight  rune
	VerticalLeft   rune
	HorizontalDown rune
	HorizontalUp   rune
	Cross          rune
}

// StraightStrokeDashPattern is the dash pattern of a straight stroke. Dash is at least 1.
type StraightStrokeDashPattern struct {
	Dash   int `json:"dash"`
	Gap    int `json:"gap"`
	Offset int `json:"offset"`
}

var SolidDashPattern = StraightStrokeDashPattern{Dash: 1, Gap: 0, Offset: 0}

func (p StraightStrokeDashPattern) IsGap(index int) bool {
	segment := max(p.Dash, 1)
	gap := max(p.Gap, 0)
	if gap == 0 {
		return false
	}
	total := segment + gap
	// Keep offset in [0, total). The modulo does not work well with negative numbers.
	offset := ((p.Offset % total) + total) % total
	return (index+offset)%total >= segment
}

type RectangleBorderCornerPattern struct {
	LeftTopRounded     bool `json:"leftTop"`
	LeftBottomRounded  bool `json:"leftBottom"`
	RightTopRounded    bool `json:"rightTop"`
	RightBottomRounded bool `json:"rightBottom"`
}

var (
	CornerNotRounded = RectangleBorderCornerPattern{}
	CornerRounded    = RectangleBorderCornerPattern{
		LeftTopRounded:     true,
		LeftBottomRounded:  true,
		RightTopRounded:    true,
		RightBottomRounded: true,
	}
)

type HorizontalAlign int

const (
	AlignLeft HorizontalAlign = iota
	AlignCenter
	AlignRight
)

func (a HorizontalAlign) MarshalText() ([]byte, error) {
	switch a {
	case AlignLeft:
		return []byte("left"), nil
	case AlignCenter:
		return []byte("middle"), nil
	case AlignRight:
		return []byte("right"), nil
	}
	return nil, fmt.Errorf("unknown horizontal align: %d", a)
}

func (a *HorizontalAlign) UnmarshalText(text []byte) error {
	switch string(text) {
	case "left":
		*a = AlignLeft
	case "middle":
		*a = AlignCenter
	case "right":
		*a = AlignRight
	default:
		return fmt.Errorf("unknown horizontal align: %q", text)
	}
	return nil
}

type VerticalAlign int

const (
	AlignTop VerticalAlign = iota
	AlignMiddle
	AlignBottom
)

func (a VerticalAlign) MarshalText() ([]byte, error) {
	switch a {
	case AlignTop:
		return []byte("top"), nil
	case AlignMiddle:
		return []byte("middle"), nil
	case AlignBottom:
		return []byte("bottom"), nil
	}
	return nil, fmt.Errorf("unknown vertical align: %d", a)
}

func (a *VerticalAlign) UnmarshalText(text []byte) error {
	switch string(text) {
	case "top":
		*a = AlignTop
	case "middle":
		*a = AlignMiddle
	case "bottom":
		*a = AlignBottom
	default:
		return fmt.Errorf("unknown vertical align: %q", text)
	}
	return nil
}

type TextAlign struct {
	Horizontal HorizontalAlign `json:"horizontal"`
	Vertical   VerticalAlign   `json:"vertical"`
}
