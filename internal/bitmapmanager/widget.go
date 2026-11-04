package bitmapmanager

import (
	"slices"

	"github.com/bluemir/nemokan/internal/bitmap"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

// widgetBitmap draws what the layout of a widget shows. A widget in a box is drawn as a
// rectangle with the extra of the widget, and the other widgets are filled with spaces, so they
// cover what is below and are selectable anywhere. See ADR-0014.
func widgetBitmap(size geo.Size, layout shape.WidgetLayout, extra shape.RectangleExtra) *bitmap.Bitmap {
	builder := bitmap.NewBuilder(size.Width, size.Height)
	if layout.HasBox {
		builder.FillBitmap(0, 0, rectangleBitmap(size, extra))
		strokeStyle := extra.StrokeStyle()
		if strokeStyle == nil {
			strokeStyle = shape.NoStrokeStyle()
		}
		drawDividers(builder, size, layout, strokeStyle)
	} else {
		builder.FillChar(' ')
	}

	for _, run := range layout.Runs {
		column := run.Column
		for _, char := range run.Text {
			builder.Put(run.Row, column, char, char)
			// The tail hides what is below, as the wide char covers two cells.
			if bitmap.CharWidth(char) == 2 {
				builder.Put(run.Row, column+1, bitmap.WideCharTail, bitmap.WideCharTail)
			}
			column += bitmap.CharWidth(char)
		}
	}
	return builder.Bitmap()
}

// drawDividers draws the lines inside the box, with junctions where they meet the border and
// each other.
func drawDividers(builder *bitmap.Builder, size geo.Size, layout shape.WidgetLayout, style *shape.StraightStrokeStyle) {
	right := size.Width - 1
	bottom := size.Height - 1
	for _, row := range layout.DividerRows {
		if row <= 0 || row >= bottom {
			continue
		}
		builder.Put(row, 0, style.VerticalRight, style.VerticalRight)
		for column := 1; column < right; column++ {
			builder.Put(row, column, style.Horizontal, style.Horizontal)
		}
		builder.Put(row, right, style.VerticalLeft, style.VerticalLeft)
	}
	for _, column := range layout.DividerColumns {
		if column <= 0 || column >= right {
			continue
		}
		builder.Put(0, column, style.HorizontalDown, style.HorizontalDown)
		for row := 1; row < bottom; row++ {
			char := style.Vertical
			if slices.Contains(layout.DividerRows, row) {
				char = style.Cross
			}
			builder.Put(row, column, char, char)
		}
		builder.Put(bottom, column, style.HorizontalUp, style.HorizontalUp)
	}
}
