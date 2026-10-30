package bitmapmanager

import (
	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func lineBitmap(jointPoints []geo.Point, extra shape.LineExtra) *bitmap.Bitmap {
	builder := newOffsetBuilder(jointPoints)

	strokeStyle := extra.StrokeStyle()
	if strokeStyle == nil {
		strokeStyle = shape.NoStrokeStyle()
	}
	for index, point := range linePointChars(jointPoints, strokeStyle) {
		visual := point.char
		if extra.DashPattern.IsGap(index) {
			visual = ' '
		}
		builder.put(point.top, point.left, visual, point.char)
	}

	if startAnchor := extra.StartAnchor(); startAnchor != nil {
		builder.putAnchor(jointPoints[0], jointPoints[1], startAnchor)
	}
	if endAnchor := extra.EndAnchor(); endAnchor != nil {
		lastIndex := len(jointPoints) - 1
		builder.putAnchor(jointPoints[lastIndex], jointPoints[lastIndex-1], endAnchor)
	}
	return builder.Bitmap()
}

func linePointChars(jointPoints []geo.Point, strokeStyle *shape.StraightStrokeStyle) []pointChar {
	if len(jointPoints) < 2 {
		return nil
	}

	first, second := jointPoints[0], jointPoints[1]
	points := []pointChar{{left: first.Left, top: first.Top, char: straightChar(first, second, strokeStyle)}}

	for i := 0; i+2 < len(jointPoints); i++ {
		point0, point1, point2 := jointPoints[i], jointPoints[i+1], jointPoints[i+2]
		points = append(points, straightLine(point0, point1, strokeStyle)...)
		points = append(points, pointChar{left: point1.Left, top: point1.Top, char: rightAngleChar(point0, point1, point2, strokeStyle)})
	}

	beforeLast, last := jointPoints[len(jointPoints)-2], jointPoints[len(jointPoints)-1]
	points = append(points, straightLine(beforeLast, last, strokeStyle)...)
	return append(points, pointChar{left: last.Left, top: last.Top, char: straightChar(beforeLast, last, strokeStyle)})
}

// straightLine returns the points between point0 and point1, excluding both.
func straightLine(point0, point1 geo.Point, strokeStyle *shape.StraightStrokeStyle) []pointChar {
	if isHorizontal(point0, point1) {
		return horizontalLine(point0.Left, point1.Left, point0.Top, strokeStyle.Horizontal)
	}
	return verticalLine(point0.Left, point0.Top, point1.Top, strokeStyle.Vertical)
}

func straightChar(point0, point1 geo.Point, strokeStyle *shape.StraightStrokeStyle) rune {
	if isHorizontal(point0, point1) {
		return strokeStyle.Horizontal
	}
	return strokeStyle.Vertical
}

func rightAngleChar(point0, point1, point2 geo.Point, strokeStyle *shape.StraightStrokeStyle) rune {
	isHorizontal0 := isHorizontal(point0, point1)
	isHorizontal1 := isHorizontal(point1, point2)
	if isHorizontal0 == isHorizontal1 {
		return straightChar(point0, point1, strokeStyle)
	}

	isLeft := point0.Left < point1.Left || point2.Left < point1.Left
	isUpper := point0.Top < point1.Top || point2.Top < point1.Top
	switch {
	case isLeft && isUpper:
		return strokeStyle.UpLeft
	case isLeft:
		return strokeStyle.DownLeft
	case isUpper:
		return strokeStyle.DownRight
	default:
		return strokeStyle.UpRight
	}
}

func isHorizontal(point0, point1 geo.Point) bool {
	return point0.Top == point1.Top
}

// offsetBuilder draws points in board coordinates into a bitmap covering the joint points.
type offsetBuilder struct {
	*bitmap.Builder
	top  int
	left int
}

func newOffsetBuilder(jointPoints []geo.Point) offsetBuilder {
	left, right := jointPoints[0].Left, jointPoints[0].Left
	top, bottom := jointPoints[0].Top, jointPoints[0].Top
	for _, point := range jointPoints {
		left = min(point.Left, left)
		right = max(point.Left, right)
		top = min(point.Top, top)
		bottom = max(point.Top, bottom)
	}
	return offsetBuilder{
		Builder: bitmap.NewBuilder(right-left+1, bottom-top+1),
		top:     top,
		left:    left,
	}
}

func (b offsetBuilder) put(row, column int, visual, direction rune) {
	b.Put(row-b.top, column-b.left, visual, direction)
}

func (b offsetBuilder) putAnchor(anchor, previous geo.Point, anchorChar *shape.AnchorChar) {
	var char rune
	switch {
	case isHorizontal(anchor, previous) && anchor.Left < previous.Left:
		char = anchorChar.Left
	case isHorizontal(anchor, previous):
		char = anchorChar.Right
	case anchor.Top < previous.Top:
		char = anchorChar.Top
	default:
		char = anchorChar.Bottom
	}
	// The anchor does not override the direction char.
	b.put(anchor.Top, anchor.Left, char, bitmap.Transparent)
}
