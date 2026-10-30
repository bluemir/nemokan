package bitmapmanager

import (
	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func rectangleBitmap(size geo.Size, extra shape.RectangleExtra) *bitmap.Bitmap {
	builder := bitmap.NewBuilder(size.Width, size.Height)

	fillStyle := extra.FillStyle()
	strokeStyle := extra.StrokeStyle()

	if fillStyle == nil && strokeStyle == nil {
		// Half transparent chars are not displayed on the canvas but make the shape selectable.
		drawBorder(builder, size, shape.NoStrokeStyle(), extra.DashPattern)
		return builder.Bitmap()
	}

	if fillStyle != nil {
		builder.FillBitmap(0, 0, fillStyle.Drawable.Bitmap(size.Width, size.Height))
	}

	isStrokeAllowed := fillStyle == nil || size.Width > 1 && size.Height > 1
	if isStrokeAllowed && strokeStyle != nil {
		drawBorder(builder, size, strokeStyle, extra.DashPattern)
	}
	return builder.Bitmap()
}

func drawBorder(builder *bitmap.Builder, size geo.Size, strokeStyle *shape.StraightStrokeStyle, dashPattern shape.StraightStrokeDashPattern) {
	if size.Width == 1 && size.Height == 1 {
		builder.Put(0, 0, '□', '□')
		return
	}

	left := 0
	top := 0
	right := size.Width - 1
	bottom := size.Height - 1

	var points []pointChar
	switch {
	// The original leaves the last cell of a one cell wide border empty. This draws it.
	case size.Width == 1:
		points = verticalLine(left, top-1, bottom+1, strokeStyle.Vertical)
	case size.Height == 1:
		points = horizontalLine(left-1, right+1, top, strokeStyle.Horizontal)
	default:
		points = append(points, pointChar{left: left, top: top, char: strokeStyle.UpRight})
		points = append(points, horizontalLine(left, right, top, strokeStyle.Horizontal)...)
		points = append(points, pointChar{left: right, top: top, char: strokeStyle.DownLeft})
		points = append(points, verticalLine(right, top, bottom, strokeStyle.Vertical)...)
		points = append(points, pointChar{left: right, top: bottom, char: strokeStyle.UpLeft})
		points = append(points, horizontalLine(right, left, bottom, strokeStyle.Horizontal)...)
		points = append(points, pointChar{left: left, top: bottom, char: strokeStyle.DownRight})
		points = append(points, verticalLine(left, bottom, top, strokeStyle.Vertical)...)
	}

	for index, point := range points {
		visual := point.char
		if dashPattern.IsGap(index) {
			visual = ' '
		}
		builder.Put(point.top, point.left, visual, point.char)
	}
}
