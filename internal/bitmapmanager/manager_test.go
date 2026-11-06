package bitmapmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func TestManager_RemoveMissing(t *testing.T) {
	shapes := shape.NewManager()
	kept := shape.NewRectangle(geo.RectByLTWH(0, 0, 3, 3), "", "")
	removed := shape.NewRectangle(geo.RectByLTWH(5, 0, 3, 3), "", "")
	shapes.Execute(shape.AddShape{Shape: kept})
	shapes.Execute(shape.AddShape{Shape: removed})

	target := New()
	target.Bitmap(kept)
	target.Bitmap(removed)
	shapes.Execute(shape.RemoveShape{Shape: removed})
	target.RemoveMissing(shapes)

	assert.Contains(t, target.idToBitmap, kept.ID())
	assert.NotContains(t, target.idToBitmap, removed.ID())
}
