// Package shapesearcher finds shapes at a point or in an area of the board.
package shapesearcher

import (
	"github.com/bluemir/nemokan/internal/bitmap"
	"github.com/bluemir/nemokan/internal/bitmapmanager"
	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

// zoneAddress identifies a 16x16 zone of the board.
type zoneAddress struct {
	row    int
	column int
}

// toZoneIndex returns the zone index of a row or column. Shifting floors negative numbers too.
func toZoneIndex(number int) int {
	return number >> 4
}

func toZoneAddress(row, column int) zoneAddress {
	return zoneAddress{row: toZoneIndex(row), column: toZoneIndex(column)}
}

type versionedZoneAddresses struct {
	versionCode int
	addresses   []zoneAddress
}

// Searcher indexes shapes by the zones they have pixels on, to find shapes at a point fast.
// A shape is only indexed after it is drawn onto the board. Do not use this for pre-draw check.
type Searcher struct {
	shapeManager  *shape.Manager
	bitmapManager *bitmapmanager.Manager

	idToZoneAddresses map[string]versionedZoneAddresses
	// zoneToOwners keeps owner ids in the order they are registered.
	zoneToOwners map[zoneAddress][]string
}

func New(shapeManager *shape.Manager, bitmapManager *bitmapmanager.Manager) *Searcher {
	return &Searcher{
		shapeManager:      shapeManager,
		bitmapManager:     bitmapManager,
		idToZoneAddresses: map[string]versionedZoneAddresses{},
		zoneToOwners:      map[zoneAddress][]string{},
	}
}

func (s *Searcher) Register(target shape.Shape) {
	for _, address := range s.zoneAddresses(target) {
		s.zoneToOwners[address] = append(s.zoneToOwners[address], target.ID())
	}
}

// Clear removes all owners of the zones overlapping bound.
func (s *Searcher) Clear(bound geo.Rect) {
	for row := toZoneIndex(bound.Top()); row <= toZoneIndex(bound.Bottom()); row++ {
		for column := toZoneIndex(bound.Left()); column <= toZoneIndex(bound.Right()); column++ {
			delete(s.zoneToOwners, zoneAddress{row: row, column: column})
		}
	}
}

// Shapes returns the shapes having a non-transparent pixel at point, from the lowest z-index.
// Groups are not included.
func (s *Searcher) Shapes(point geo.Point) []shape.Shape {
	var shapes []shape.Shape
	for _, owner := range s.potentialOwners(point) {
		found, ok := s.shapeManager.Shape(owner)
		if !ok {
			continue
		}
		drawn := s.bitmapManager.Bitmap(found)
		if drawn == nil {
			continue
		}
		position := found.Bound().Position
		if drawn.Visual(point.Row()-position.Row(), point.Column()-position.Column()) != bitmap.Transparent {
			shapes = append(shapes, found)
		}
	}
	return shapes
}

// ShapesInZone returns the shapes overlapping bound.
func (s *Searcher) ShapesInZone(bound geo.Rect) []shape.Shape {
	topLeft := toZoneAddress(bound.Top(), bound.Left())
	bottomRight := toZoneAddress(bound.Bottom(), bound.Right())

	var shapes []shape.Shape
	seen := map[string]bool{}
	for row := topLeft.row; row <= bottomRight.row; row++ {
		for column := topLeft.column; column <= bottomRight.column; column++ {
			for _, owner := range s.zoneToOwners[zoneAddress{row: row, column: column}] {
				if seen[owner] {
					continue
				}
				seen[owner] = true
				found, ok := s.shapeManager.Shape(owner)
				if ok && found.IsOverlapped(bound) {
					shapes = append(shapes, found)
				}
			}
		}
	}
	return shapes
}

// EdgeDirection returns the direction of the edge at point of a rectangle or a text. Vertices
// are excluded to keep the anchor direction similar to the previous setup. If many shapes have
// an edge at point, the first one is used.
func (s *Searcher) EdgeDirection(point geo.Point) (geo.Direction, bool) {
	for _, owner := range s.potentialOwners(point) {
		found, ok := s.shapeManager.Shape(owner)
		if !ok {
			continue
		}
		switch found.(type) {
		case *shape.Rectangle, *shape.Text:
		default:
			continue
		}
		if !found.Contains(point) || found.IsVertex(point) {
			continue
		}

		bound := found.Bound()
		if bound.Left() == point.Left || bound.Right() == point.Left {
			return geo.Vertical, true
		}
		if bound.Top() == point.Top || bound.Bottom() == point.Top {
			return geo.Horizontal, true
		}
	}
	return geo.Horizontal, false
}

func (s *Searcher) potentialOwners(point geo.Point) []string {
	return s.zoneToOwners[toZoneAddress(point.Row(), point.Column())]
}

// zoneAddresses returns the zones target has pixels on, cached by the version code.
func (s *Searcher) zoneAddresses(target shape.Shape) []zoneAddress {
	if cached, ok := s.idToZoneAddresses[target.ID()]; ok && cached.versionCode == target.VersionCode() {
		return cached.addresses
	}

	drawn := s.bitmapManager.Bitmap(target)
	if drawn == nil {
		delete(s.idToZoneAddresses, target.ID())
		return nil
	}

	position := target.Bound().Position
	var addresses []zoneAddress
	seen := map[zoneAddress]bool{}
	for rowIndex, row := range drawn.Rows() {
		row.ForEachIndex(0, drawn.Size().Width, func(columnIndex int, _, _ rune) {
			address := toZoneAddress(rowIndex+position.Row(), columnIndex+position.Column())
			if !seen[address] {
				seen[address] = true
				addresses = append(addresses, address)
			}
		})
	}
	s.idToZoneAddresses[target.ID()] = versionedZoneAddresses{versionCode: target.VersionCode(), addresses: addresses}
	return addresses
}
