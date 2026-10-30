package bitmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilder_FillBitmap(t *testing.T) {
	source := NewBuilder(2, 1)
	source.Put(0, 0, 'a', 'a')
	source.Put(0, 1, 'b', 'b')

	target := NewBuilder(4, 3)
	target.FillBitmap(1, 2, source.Bitmap())

	assert.Equal(t, lines(
		"    ",
		"  ab",
		"    ",
	), target.Bitmap().String())
}

func TestBuilder_FillBitmap_HalfTransparentKeepsDrawn(t *testing.T) {
	source := NewBuilder(2, 1)
	source.Put(0, 0, HalfTransparent, HalfTransparent)
	source.Put(0, 1, HalfTransparent, HalfTransparent)

	target := NewBuilder(2, 1)
	target.Put(0, 0, 'x', 'x')
	target.FillBitmap(0, 0, source.Bitmap())

	bitmap := target.Bitmap()
	assert.Equal(t, 'x', bitmap.Visual(0, 0))
	assert.Equal(t, HalfTransparent, bitmap.Visual(0, 1))
}
