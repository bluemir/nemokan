// Package shape holds the shapes of a sketch and the commands that change them.
// Ported from MonoSketch shape.
package shape

import (
	"math/rand/v2"

	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/uuid"
)

// Shape is one of *Rectangle, *Text, *Line, *Widget and *Group.
// Two shapes with the same ID are the same shape regardless of their other attributes.
//
// VersionCode changes whenever an attribute changes. Every change must go through update
// to keep it accurate.
type Shape interface {
	ID() string
	// ParentID is empty if the shape is not in a group yet.
	ParentID() string
	VersionCode() int
	Bound() geo.Rect

	// Name is the name the user gave in the layers. It is empty if the user has not named it.
	Name() string
	// IsHidden and IsLocked are only of the shape itself. A shape in a hidden or locked group is
	// also hidden or locked, which the caller checks with the groups. See ADR-0013.
	IsHidden() bool
	IsLocked() bool

	// SetBound moves or resizes the shape. Group ignores it.
	SetBound(newBound geo.Rect)
	// SetExtra panics if newExtra is not the extra type of the shape.
	SetExtra(newExtra Extra)

	Contains(point geo.Point) bool
	IsVertex(point geo.Point) bool
	IsOverlapped(rect geo.Rect) bool

	// Serializable converts the shape for storing. If isIDIncluded is false, the id is marked
	// temporary and a new id is assigned when restored, for example by copy and paste.
	Serializable(isIDIncluded bool) SerializableShape

	setParentID(parentID string)
	setName(name string)
	setHidden(isHidden bool)
	setLocked(isLocked bool)
	update(action func() bool)
}

// base has the attributes every shape has.
type base struct {
	id          string
	parentID    string
	versionCode int

	name     string
	isHidden bool
	isLocked bool
}

// newBase generates an id if id is empty.
func newBase(id, parentID string) base {
	if id == "" {
		id = uuid.Generate()
	}
	return base{id: id, parentID: parentID, versionCode: nextVersionCode(0)}
}

func (b *base) ID() string                  { return b.id }
func (b *base) ParentID() string            { return b.parentID }
func (b *base) VersionCode() int            { return b.versionCode }
func (b *base) setParentID(parentID string) { b.parentID = parentID }
func (b *base) Name() string                { return b.name }
func (b *base) IsHidden() bool              { return b.isHidden }
func (b *base) IsLocked() bool              { return b.isLocked }

func (b *base) setName(name string) {
	b.update(func() bool {
		isChanged := b.name != name
		b.name = name
		return isChanged
	})
}

func (b *base) setHidden(isHidden bool) {
	b.update(func() bool {
		isChanged := b.isHidden != isHidden
		b.isHidden = isHidden
		return isChanged
	})
}

func (b *base) setLocked(isLocked bool) {
	b.update(func() bool {
		isChanged := b.isLocked != isLocked
		b.isLocked = isLocked
		return isChanged
	})
}

func (b *base) serializableBase(isIDIncluded bool) serializableBase {
	return serializableBase{
		ID:            b.id,
		IsIDTemporary: !isIDIncluded,
		VersionCode:   b.versionCode,
		Name:          b.name,
		IsHidden:      b.isHidden,
		IsLocked:      b.isLocked,
	}
}

// restore sets the attributes kept in serializable.
func (b *base) restore(serializable serializableBase) {
	b.versionCode = serializable.VersionCode
	b.name = serializable.Name
	b.isHidden = serializable.IsHidden
	b.isLocked = serializable.IsLocked
}

// update runs action and changes the version if action reports a change.
func (b *base) update(action func() bool) {
	if action() {
		b.versionCode = nextVersionCode(b.versionCode)
	}
}

// nextVersionCode returns a random number different from excluded.
func nextVersionCode(excluded int) int {
	for {
		// int32 keeps the value safe as a JS number.
		code := int(rand.Int32())
		if code != excluded {
			return code
		}
	}
}

// shapeID identifies a shape without keeping the shape itself.
type shapeID string

func (id shapeID) ID() string { return string(id) }
