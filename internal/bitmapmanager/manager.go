// Package bitmapmanager draws shapes into bitmaps and caches them.
package bitmapmanager

import (
	"github.com/bluemir/nemokan/internal/bitmap"
	"github.com/bluemir/nemokan/internal/shape"
)

// Manager caches the bitmap of each shape. The cache hits when both the id and the version
// code of the shape match.
type Manager struct {
	idToBitmap map[string]versionedBitmap
}

type versionedBitmap struct {
	versionCode int
	bitmap      *bitmap.Bitmap
}

func New() *Manager {
	return &Manager{idToBitmap: map[string]versionedBitmap{}}
}

// RemoveMissing drops the bitmaps of the shapes which are no longer in shapes: removed shapes
// and the shapes of a document which was closed.
func (m *Manager) RemoveMissing(shapes *shape.Manager) {
	for id := range m.idToBitmap {
		if _, ok := shapes.Shape(id); !ok {
			delete(m.idToBitmap, id)
		}
	}
}

// Bitmap returns the bitmap of target, or nil for a group. A group has no bitmap of its own;
// its items are drawn one by one.
func (m *Manager) Bitmap(target shape.Shape) *bitmap.Bitmap {
	if cached, ok := m.idToBitmap[target.ID()]; ok && cached.versionCode == target.VersionCode() {
		return cached.bitmap
	}

	var drawn *bitmap.Bitmap
	switch target := target.(type) {
	case *shape.Rectangle:
		drawn = rectangleBitmap(target.Bound().Size, target.Extra())
	case *shape.Text:
		drawn = textBitmap(target.Bound().Size, target.RenderableText().Lines(), target.Extra(), target.IsTextEditing())
	case *shape.Line:
		drawn = lineBitmap(target.ReducedJointPoints(), target.Extra())
	case *shape.Widget:
		drawn = widgetBitmap(target.Bound().Size, target.Layout(), target.Extra())
	default:
		return nil
	}
	m.idToBitmap[target.ID()] = versionedBitmap{versionCode: target.VersionCode(), bitmap: drawn}
	return drawn
}
