// Package bitmap holds the look of a shape after drawing.
// Ported from MonoSketch monobitmap.
package bitmap

import (
	"sort"
	"strings"

	"github.com/bluemir/nemokan/internal/geo"
)

const (
	// Transparent is transparent in both rendering and selection.
	Transparent rune = 0
	// HalfTransparent is transparent in rendering but visible for selection.
	HalfTransparent rune = 1
)

// Bitmap is immutable. Create it with Builder.
type Bitmap struct {
	rows []Row
}

func (b *Bitmap) Rows() []Row { return b.rows }

func (b *Bitmap) Size() geo.Size {
	if len(b.rows) == 0 {
		return geo.Size{}
	}
	return geo.Size{Width: b.rows[0].size, Height: len(b.rows)}
}

func (b *Bitmap) IsEmpty() bool {
	return b.Size() == geo.Size{}
}

func (b *Bitmap) Visual(row, column int) rune {
	if row < 0 || row >= len(b.rows) {
		return Transparent
	}
	cell, ok := b.rows[row].cell(column)
	if !ok {
		return Transparent
	}
	return cell.visual
}

func (b *Bitmap) Direction(row, column int) rune {
	if row < 0 || row >= len(b.rows) {
		return Transparent
	}
	cell, ok := b.rows[row].cell(column)
	if !ok {
		return Transparent
	}
	return cell.direction
}

func (b *Bitmap) String() string {
	lines := make([]string, len(b.rows))
	for i, row := range b.rows {
		lines[i] = row.String()
	}
	return strings.Join(lines, "\n")
}

type Builder struct {
	bound geo.Rect

	// visualMatrix stores the characters displayed on the canvas, except
	// crossing points which the board decides after combining with what is
	// already drawn.
	visualMatrix [][]rune

	// directionMatrix stores the characters used to find crossing points
	// when painting on the board.
	directionMatrix [][]rune
}

func NewBuilder(width, height int) *Builder {
	visualMatrix := make([][]rune, height)
	directionMatrix := make([][]rune, height)
	for row := range height {
		// rune slices start as Transparent (0).
		visualMatrix[row] = make([]rune, width)
		directionMatrix[row] = make([]rune, width)
	}
	return &Builder{
		bound:           geo.RectByLTWH(0, 0, width, height),
		visualMatrix:    visualMatrix,
		directionMatrix: directionMatrix,
	}
}

func (b *Builder) Put(row, column int, visual, direction rune) {
	if row < 0 || row >= b.bound.Height() || column < 0 || column >= b.bound.Width() {
		return
	}
	b.visualMatrix[row][column] = visual

	// The direction char is only overridden with non-transparent chars.
	if direction != Transparent {
		b.directionMatrix[row][column] = direction
	}
}

func (b *Builder) FillChar(char rune) {
	for row := range b.visualMatrix {
		for column := range b.visualMatrix[row] {
			b.visualMatrix[row][column] = char
			// TODO: Delegate direction char
			b.directionMatrix[row][column] = char
		}
	}
}

// FillBitmap draws bitmap with its top-left at (row, column).
func (b *Builder) FillBitmap(row, column int, bitmap *Bitmap) {
	if bitmap.IsEmpty() {
		return
	}
	size := bitmap.Size()
	// The original passes (row, column) as (left, top) here. Every caller uses (0, 0), so this
	// fixes the order without changing behavior.
	inBound := geo.RectByLTWH(column, row, size.Width, size.Height)

	overlap, ok := b.bound.OverlappedRect(inBound)
	if !ok {
		return
	}
	start := overlap.Position.Minus(b.bound.Position)
	inStart := overlap.Position.Minus(inBound.Position)

	for r := range overlap.Height() {
		destVisual := b.visualMatrix[start.Row()+r]
		destDirection := b.directionMatrix[start.Row()+r]

		bitmap.rows[inStart.Row()+r].ForEachIndex(inStart.Column(), inStart.Column()+overlap.Width(), func(index int, visual, direction rune) {
			destIndex := start.Column() + index
			// visual from a Row is never Transparent because Row drops those cells.
			if visual != HalfTransparent || destVisual[destIndex] == Transparent {
				destVisual[destIndex] = visual
			}
			// TODO: Double check this condition
			if direction != HalfTransparent || destDirection[destIndex] == Transparent {
				destDirection[destIndex] = direction
			}
		})
	}
}

func (b *Builder) Bitmap() *Bitmap {
	rows := make([]Row, len(b.visualMatrix))
	for i := range b.visualMatrix {
		rows[i] = newRow(b.visualMatrix[i], b.directionMatrix[i])
	}
	return &Bitmap{rows: rows}
}

// Row keeps only the cells that have a visible character, to save memory.
// For "ab      c", only a, b and c are kept.
type Row struct {
	size int
	// cells are sorted by index.
	cells []cell
}

type cell struct {
	index     int
	visual    rune
	direction rune
}

func newRow(visualChars, directionChars []rune) Row {
	var cells []cell
	for index, visual := range visualChars {
		direction := directionChars[index]
		if visual != Transparent || direction != Transparent {
			cells = append(cells, cell{index: index, visual: visual, direction: direction})
		}
	}
	return Row{size: len(visualChars), cells: cells}
}

// ForEachIndex calls action for each kept cell in [fromIndex, toExclusiveIndex).
// The index passed to action is relative to fromIndex.
func (r Row) ForEachIndex(fromIndex, toExclusiveIndex int, action func(index int, visual, direction rune)) {
	low := sort.Search(len(r.cells), func(i int) bool { return r.cells[i].index >= fromIndex })
	for _, cell := range r.cells[low:] {
		if cell.index >= toExclusiveIndex {
			break
		}
		action(cell.index-fromIndex, cell.visual, cell.direction)
	}
}

func (r Row) cell(column int) (cell, bool) {
	i := sort.Search(len(r.cells), func(i int) bool { return r.cells[i].index >= column })
	if i < len(r.cells) && r.cells[i].index == column {
		return r.cells[i], true
	}
	return cell{}, false
}

// String leaves out the tails of wide chars, so a wide char shows as one char.
func (r Row) String() string {
	chars := []rune(strings.Repeat(" ", r.size))
	for _, cell := range r.cells {
		chars[cell.index] = cell.visual
	}
	var text strings.Builder
	for _, char := range chars {
		if char != WideCharTail {
			text.WriteRune(char)
		}
	}
	return text.String()
}
