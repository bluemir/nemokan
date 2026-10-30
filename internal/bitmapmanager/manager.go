// Package bitmapmanager draws shapes into bitmaps and caches them.
package bitmapmanager

import (
	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/shape"
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

// Bitmap returns the bitmap of target, or nil for a group. Groups are not drawn since they
// change very frequently.
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
	default:
		return nil
	}
	m.idToBitmap[target.ID()] = versionedBitmap{versionCode: target.VersionCode(), bitmap: drawn}
	return drawn
}
