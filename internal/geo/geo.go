// Package geo holds the integer geometry used on the character board.
// Ported from MonoSketch graphicsgeo.
package geo

import (
	"encoding/json"
	"fmt"
)

type Point struct {
	Left int `json:"left"`
	Top  int `json:"top"`
}

func (p Point) Row() int    { return p.Top }
func (p Point) Column() int { return p.Left }

func (p Point) Minus(base Point) Point {
	return Point{Left: p.Left - base.Left, Top: p.Top - base.Top}
}

func (p Point) Plus(base Point) Point {
	return Point{Left: p.Left + base.Left, Top: p.Top + base.Top}
}

type Direction int

const (
	Horizontal Direction = iota
	Vertical
)

func (d Direction) MarshalText() ([]byte, error) {
	if d == Vertical {
		return []byte("vertical"), nil
	}
	return []byte("horizontal"), nil
}

func (d *Direction) UnmarshalText(text []byte) error {
	switch string(text) {
	case "horizontal":
		*d = Horizontal
	case "vertical":
		*d = Vertical
	default:
		return fmt.Errorf("unknown direction: %q", text)
	}
	return nil
}

// Normalized returns the perpendicular direction.
func (d Direction) Normalized() Direction {
	if d == Vertical {
		return Horizontal
	}
	return Vertical
}

type DirectedPoint struct {
	Direction Direction `json:"direction"`
	Left      int       `json:"left"`
	Top       int       `json:"top"`
}

func (p DirectedPoint) Point() Point {
	return Point{Left: p.Left, Top: p.Top}
}

func (p DirectedPoint) Plus(base Point) DirectedPoint {
	return DirectedPoint{Direction: p.Direction, Left: p.Left + base.Left, Top: p.Top + base.Top}
}

// PointF is only for calculation. Do not store it.
type PointF struct {
	Left float64 `json:"left"`
	Top  float64 `json:"top"`
}

func (p PointF) Row() float64    { return p.Top }
func (p PointF) Column() float64 { return p.Left }

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type SizeF struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Rect is a cell area. Right and Bottom are inclusive.
type Rect struct {
	Position Point
	Size     Size
}

func RectByLTRB(left, top, right, bottom int) Rect {
	return Rect{
		Position: Point{Left: min(left, right), Top: min(top, bottom)},
		Size:     Size{Width: abs(right-left) + 1, Height: abs(bottom-top) + 1},
	}
}

func RectByLTWH(left, top, width, height int) Rect {
	return Rect{
		Position: Point{Left: left, Top: top},
		Size:     Size{Width: width, Height: height},
	}
}

// BoundOf returns the smallest rect covering all rects. rects must not be empty.
func BoundOf(rects []Rect) Rect {
	left, right := rects[0].Left(), rects[0].Right()
	top, bottom := rects[0].Top(), rects[0].Bottom()
	for _, rect := range rects {
		left = min(rect.Left(), left)
		right = max(rect.Right(), right)
		top = min(rect.Top(), top)
		bottom = max(rect.Bottom(), bottom)
	}
	return RectByLTRB(left, top, right, bottom)
}

func (r Rect) Width() int  { return r.Size.Width }
func (r Rect) Height() int { return r.Size.Height }
func (r Rect) Left() int   { return r.Position.Left }
func (r Rect) Right() int  { return r.Position.Left + r.Size.Width - 1 }
func (r Rect) Top() int    { return r.Position.Top }
func (r Rect) Bottom() int { return r.Position.Top + r.Size.Height - 1 }

func (r Rect) Contains(point Point) bool {
	return r.Left() <= point.Left && point.Left <= r.Right() &&
		r.Top() <= point.Top && point.Top <= r.Bottom()
}

func (r Rect) IsOverlapped(other Rect) bool {
	isHorizontalOverlap := inRange(r.Left(), other.Left(), other.Right()) || inRange(other.Left(), r.Left(), r.Right())
	isVerticalOverlap := inRange(r.Top(), other.Top(), other.Bottom()) || inRange(other.Top(), r.Top(), r.Bottom())
	return isHorizontalOverlap && isVerticalOverlap
}

// OverlappedRect returns the intersection of the two rects.
func (r Rect) OverlappedRect(other Rect) (Rect, bool) {
	if !r.IsOverlapped(other) {
		return Rect{}, false
	}
	offset := other.Position.Minus(r.Position)
	top := max(offset.Top, 0)
	bottom := min(offset.Top+other.Height(), r.Height()) - 1
	left := max(offset.Left, 0)
	right := min(offset.Left+other.Width(), r.Width()) - 1
	return RectByLTRB(
		left+r.Position.Left,
		top+r.Position.Top,
		right+r.Position.Left,
		bottom+r.Position.Top,
	), true
}

// IsVertex reports whether point is one of the four corners.
func (r Rect) IsVertex(point Point) bool {
	isOnVerticalEdge := point.Left == r.Left() || point.Left == r.Right()
	isOnHorizontalEdge := point.Top == r.Top() || point.Top == r.Bottom()
	return isOnHorizontalEdge && isOnVerticalEdge
}

// rectJSON flattens Rect as {"left", "top", "width", "height"}.
type rectJSON struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (r Rect) MarshalJSON() ([]byte, error) {
	return json.Marshal(rectJSON{Left: r.Left(), Top: r.Top(), Width: r.Width(), Height: r.Height()})
}

func (r *Rect) UnmarshalJSON(data []byte) error {
	var value rectJSON
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*r = RectByLTWH(value.Left, value.Top, value.Width, value.Height)
	return nil
}

func (r Rect) String() string {
	return fmt.Sprintf("[%d, %d] - [%d x %d]", r.Left(), r.Top(), r.Width(), r.Height())
}

func inRange(value, low, high int) bool {
	return low <= value && value <= high
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
