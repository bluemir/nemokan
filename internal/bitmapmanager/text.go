package bitmapmanager

import (
	"github.com/bluemir/nemokan/internal/bitmap"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func textBitmap(boundSize geo.Size, renderableText []string, extra shape.TextExtra, isTextEditing bool) *bitmap.Bitmap {
	builder := bitmap.NewBuilder(boundSize.Width, boundSize.Height)
	if !(boundSize.Width == 1 && boundSize.Height == 1 && isTextEditing) {
		builder.FillBitmap(0, 0, rectangleBitmap(boundSize, extra.BoundExtra))
	}

	// While editing, the text is shown by the text editor on the UI.
	if !isTextEditing {
		fillText(builder, renderableText, boundSize, extra)
	}
	return builder.Bitmap()
}

func fillText(builder *bitmap.Builder, renderableText []string, boundSize geo.Size, extra shape.TextExtra) {
	offset := 0
	if extra.HasBorder() {
		offset = 1
	}

	maxTextWidth := boundSize.Width - offset*2
	maxTextHeight := max(boundSize.Height-offset*2, 0)

	row0 := offset
	if maxTextHeight >= len(renderableText) {
		switch extra.TextAlign.Vertical {
		case shape.AlignMiddle:
			row0 = (maxTextHeight-len(renderableText))/2 + offset
		case shape.AlignBottom:
			row0 = maxTextHeight - len(renderableText) + offset
		}
	}

	for rowIndex, line := range renderableText[:min(len(renderableText), maxTextHeight)] {
		lineWidth := bitmap.StringWidth(line)
		column0 := offset
		switch extra.TextAlign.Horizontal {
		case shape.AlignCenter:
			column0 = (maxTextWidth-lineWidth)/2 + offset
		case shape.AlignRight:
			column0 = maxTextWidth - lineWidth + offset
		}
		column := column0
		for _, char := range line {
			if char != ' ' {
				builder.Put(row0+rowIndex, column, char, char)
				// The tail hides what is below, as the wide char covers two cells.
				if bitmap.CharWidth(char) == 2 {
					builder.Put(row0+rowIndex, column+1, bitmap.WideCharTail, bitmap.WideCharTail)
				}
			}
			column += bitmap.CharWidth(char)
		}
	}
}
