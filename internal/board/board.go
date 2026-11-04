// Package board manages every pixel of the sketch. It allows infinite drawing by
// splitting the area into fixed size painter boards.
// Ported from MonoSketch monoboard.
package board

import (
	"github.com/bluemir/nemokan/internal/bitmap"
	"github.com/bluemir/nemokan/internal/geo"
)

var unitSize = geo.Size{Width: 16, Height: 16}

type Board struct {
	painterBoards map[boardAddress]*painterBoard
	windowBound   geo.Rect
}

type boardAddress struct {
	row    int
	column int
}

func New() *Board {
	return &Board{painterBoards: map[boardAddress]*painterBoard{}}
}

// ClearAndSetWindow clears the pixels in windowBound. Only painter boards overlapping the
// window are drawn afterward.
func (b *Board) ClearAndSetWindow(windowBound geo.Rect) {
	b.windowBound = windowBound
	for _, painter := range b.overlappedBoards(windowBound) {
		painter.clear()
	}
}

func (b *Board) Fill(position geo.Point, bitmapToDraw *bitmap.Bitmap, highlight Highlight) {
	rect := geo.Rect{Position: position, Size: bitmapToDraw.Size()}

	var crossPoints []crossPoint
	for _, painter := range b.createOverlappedBoards(rect) {
		crossPoints = append(crossPoints, painter.fillBitmap(position, bitmapToDraw, highlight)...)
	}
	b.drawCrossPoints(crossPoints, highlight)
}

func (b *Board) drawCrossPoints(crossPoints []crossPoint, highlight Highlight) {
	for _, point := range crossPoints {
		current := b.pixel(point.left, point.top)
		if current == nil {
			continue
		}
		char, ok := crossingChar(
			point.visual, point.neighbors,
			current.Visual, neighbors{
				left:   b.Get(point.left-1, point.top).Direction,
				right:  b.Get(point.left+1, point.top).Direction,
				top:    b.Get(point.left, point.top-1).Direction,
				bottom: b.Get(point.left, point.top+1).Direction,
			},
		)
		if ok {
			current.set(char, char, highlight)
		} else {
			current.set(point.visual, point.direction, highlight)
		}
	}
}

// Get returns a transparent pixel when nothing is drawn at (left, top).
func (b *Board) Get(left, top int) Pixel {
	if pixel := b.pixel(left, top); pixel != nil {
		return *pixel
	}
	return Pixel{}
}

func (b *Board) pixel(left, top int) *Pixel {
	painter, ok := b.painterBoards[toBoardAddress(left, top)]
	if !ok {
		return nil
	}
	return painter.pixel(left, top)
}

// overlappedBoards returns the existing painter boards in the window that overlap rect.
func (b *Board) overlappedBoards(rect geo.Rect) []*painterBoard {
	var result []*painterBoard
	for _, address := range overlappedAddresses(rect) {
		painter, ok := b.painterBoards[address]
		if ok && b.windowBound.IsOverlapped(painter.bound) {
			result = append(result, painter)
		}
	}
	return result
}

// createOverlappedBoards creates the missing painter boards that overlap rect, and returns
// the ones in the window.
func (b *Board) createOverlappedBoards(rect geo.Rect) []*painterBoard {
	var result []*painterBoard
	for _, address := range overlappedAddresses(rect) {
		painter := b.createBoard(address)
		if b.windowBound.IsOverlapped(painter.bound) {
			result = append(result, painter)
		}
	}
	return result
}

func (b *Board) createBoard(address boardAddress) *painterBoard {
	painter, ok := b.painterBoards[address]
	if !ok {
		position := geo.Point{Left: address.column * unitSize.Width, Top: address.row * unitSize.Height}
		painter = newPainterBoard(geo.Rect{Position: position, Size: unitSize})
		b.painterBoards[address] = painter
	}
	return painter
}

func overlappedAddresses(rect geo.Rect) []boardAddress {
	leftIndex := floorDiv(rect.Left(), unitSize.Width)
	rightIndex := floorDiv(rect.Right(), unitSize.Width)
	topIndex := floorDiv(rect.Top(), unitSize.Height)
	bottomIndex := floorDiv(rect.Bottom(), unitSize.Height)

	var addresses []boardAddress
	for column := leftIndex; column <= rightIndex; column++ {
		for row := topIndex; row <= bottomIndex; row++ {
			addresses = append(addresses, boardAddress{row: row, column: column})
		}
	}
	return addresses
}

func toBoardAddress(left, top int) boardAddress {
	return boardAddress{
		row:    floorDiv(top, unitSize.Height),
		column: floorDiv(left, unitSize.Width),
	}
}

func floorDiv(value, denominator int) int {
	if value >= 0 || value%denominator == 0 {
		return value / denominator
	}
	return value/denominator - 1
}

// String renders all painter boards. It is for debugging and tests.
func (b *Board) String() string {
	if len(b.painterBoards) == 0 {
		return ""
	}
	var left, right, top, bottom int
	first := true
	for address := range b.painterBoards {
		if first {
			left, right, top, bottom = address.column, address.column, address.row, address.row
			first = false
		}
		left = min(left, address.column)
		right = max(right, address.column)
		top = min(top, address.row)
		bottom = max(bottom, address.row)
	}
	return b.StringInBound(geo.RectByLTWH(
		left*unitSize.Width,
		top*unitSize.Height,
		(right-left+1)*unitSize.Width,
		(bottom-top+1)*unitSize.Height,
	))
}

// StringInBound renders the pixels in bound as text. Used for exporting.
func (b *Board) StringInBound(bound geo.Rect) string {
	result := newPainterBoard(bound)
	for _, painter := range b.painterBoards {
		result.fillBoard(painter)
	}
	return result.String()
}
