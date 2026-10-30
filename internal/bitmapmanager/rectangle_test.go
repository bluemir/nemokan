package bitmapmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func TestRectangleBitmap(t *testing.T) {
	rectangle := shape.NewRectangle(geo.RectByLTWH(10, 10, 5, 5), "", "")
	drawn := rectangleBitmap(rectangle.Bound().Size, rectangle.Extra())
	assert.Equal(t, rectangle.Bound().Size, drawn.Size())
	assert.Equal(t, ""+
		"┌───┐\n"+
		"│   │\n"+
		"│   │\n"+
		"│   │\n"+
		"└───┘", drawn.String())
}

func TestRectangleBitmap_OneCellWide(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		bound    geo.Rect
		expected string
	}{
		{"width 1", geo.RectByLTWH(0, 0, 1, 4), "│\n│\n│\n│"},
		{"height 1", geo.RectByLTWH(0, 0, 4, 1), "────"},
		{"1x1", geo.RectByLTWH(0, 0, 1, 1), "□"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rectangle := shape.NewRectangle(testCase.bound, "", "")
			assert.Equal(t, testCase.expected, rectangleBitmap(rectangle.Bound().Size, rectangle.Extra()).String())
		})
	}
}

func TestTextBitmap(t *testing.T) {
	text := shape.NewText(geo.RectByLTWH(0, 0, 7, 5), "", "", true)
	text.SetText("012345678\nabc")
	drawn := textBitmap(text.Bound().Size, text.RenderableText().Lines(), text.Extra(), false)
	assert.Equal(t, ""+
		"┌─────┐\n"+
		"│01234│\n"+
		"│5678 │\n"+
		"│ abc │\n"+
		"└─────┘", drawn.String())
}

func TestTextBitmap_WideChars(t *testing.T) {
	text := shape.NewText(geo.RectByLTWH(0, 0, 8, 4), "", "", true)
	text.SetText("안녕 hi")
	drawn := textBitmap(text.Bound().Size, text.RenderableText().Lines(), text.Extra(), false)
	// The tails are left out of the string, so each row with Korean is shorter.
	assert.Equal(t, ""+
		"┌──────┐\n"+
		"│ 안녕 │\n"+
		"│  hi  │\n"+
		"└──────┘", drawn.String())
	assert.Equal(t, '안', drawn.Visual(1, 2))
	assert.Equal(t, bitmap.WideCharTail, drawn.Visual(1, 3))
	assert.Equal(t, '녕', drawn.Visual(1, 4))
}

func TestManager_Cache(t *testing.T) {
	manager := New()
	rectangle := shape.NewRectangle(geo.RectByLTWH(0, 0, 3, 3), "", "")

	first := manager.Bitmap(rectangle)
	assert.Same(t, first, manager.Bitmap(rectangle))

	rectangle.SetBound(geo.RectByLTWH(0, 0, 4, 4))
	assert.NotSame(t, first, manager.Bitmap(rectangle))

	assert.Nil(t, manager.Bitmap(shape.NewGroup("", "")))
}
