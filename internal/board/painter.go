package board

import (
	"strings"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
)

// painterBoard holds the pixels of one unit area, at their absolute position.
type painterBoard struct {
	bound  geo.Rect
	matrix [][]Pixel
}

func newPainterBoard(bound geo.Rect) *painterBoard {
	matrix := make([][]Pixel, bound.Height())
	for row := range matrix {
		matrix[row] = make([]Pixel, bound.Width())
	}
	return &painterBoard{bound: bound, matrix: matrix}
}

func (p *painterBoard) clear() {
	for row := range p.matrix {
		clear(p.matrix[row])
	}
}

// fillBoard copies other onto this board. Transparent pixels of other do not overwrite.
func (p *painterBoard) fillBoard(other *painterBoard) {
	overlap, ok := p.bound.OverlappedRect(other.bound)
	if !ok {
		return
	}
	start := overlap.Position.Minus(p.bound.Position)
	inStart := overlap.Position.Minus(other.bound.Position)

	for r := range overlap.Height() {
		src := other.matrix[inStart.Row()+r][inStart.Column() : inStart.Column()+overlap.Width()]
		dest := p.matrix[start.Row()+r]
		for index, pixel := range src {
			if !pixel.IsTransparent() {
				dest[start.Column()+index] = pixel
			}
		}
	}
}

// fillBitmap draws bitmap at position, except crossing points.
// A crossing point is where a connectable char lands on an already drawn pixel. The
// returned crossing points are drawn by Board, because a painter board cannot see the
// pixels outside its bound that decide the final character.
// Transparent pixels of bitmap do not overwrite.
func (p *painterBoard) fillBitmap(position geo.Point, bitmapToDraw *bitmap.Bitmap, highlight Highlight) []crossPoint {
	if bitmapToDraw.IsEmpty() {
		return nil
	}
	inBound := geo.Rect{Position: position, Size: bitmapToDraw.Size()}

	overlap, ok := p.bound.OverlappedRect(inBound)
	if !ok {
		return nil
	}
	start := overlap.Position.Minus(p.bound.Position)
	inStart := overlap.Position.Minus(position)

	var crossPoints []crossPoint
	for r := range overlap.Height() {
		bitmapRow := inStart.Row() + r
		painterRow := start.Row() + r
		dest := p.matrix[painterRow]

		bitmapToDraw.Rows()[bitmapRow].ForEachIndex(inStart.Column(), inStart.Column()+overlap.Width(), func(index int, visual, direction rune) {
			bitmapColumn := inStart.Column() + index
			painterColumn := start.Column() + index
			pixel := &dest[painterColumn]

			if pixel.IsTransparent() || pixel.Visual == visual || !isConnectable(visual) {
				// Half transparent chars are not drawn. Fully transparent chars are already
				// removed by the bitmap.
				if visual != bitmap.HalfTransparent {
					pixel.set(visual, direction, highlight)
				}
				return
			}
			crossPoints = append(crossPoints, crossPoint{
				left:      painterColumn + p.bound.Left(),
				top:       painterRow + p.bound.Top(),
				visual:    visual,
				direction: direction,
				neighbors: neighbors{
					left:   bitmapToDraw.Direction(bitmapRow, bitmapColumn-1),
					right:  bitmapToDraw.Direction(bitmapRow, bitmapColumn+1),
					top:    bitmapToDraw.Direction(bitmapRow-1, bitmapColumn),
					bottom: bitmapToDraw.Direction(bitmapRow+1, bitmapColumn),
				},
			})
		})
	}
	return crossPoints
}

// pixel returns nil when (left, top) is outside the board.
func (p *painterBoard) pixel(left, top int) *Pixel {
	column := left - p.bound.Left()
	row := top - p.bound.Top()
	if row < 0 || row >= len(p.matrix) || column < 0 || column >= len(p.matrix[row]) {
		return nil
	}
	return &p.matrix[row][column]
}

func (p *painterBoard) String() string {
	lines := make([]string, len(p.matrix))
	for row, pixels := range p.matrix {
		var line strings.Builder
		for _, pixel := range pixels {
			line.WriteString(pixel.String())
		}
		lines[row] = line.String()
	}
	return strings.Join(lines, "\n")
}

// crossPoint is a crossing point found while drawing a bitmap. It is drawn after the
// other pixels of the bitmap.
type crossPoint struct {
	left      int
	top       int
	visual    rune
	direction rune
	// neighbors are the direction chars around the point in the bitmap.
	neighbors neighbors
}
