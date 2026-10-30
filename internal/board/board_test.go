package board

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
)

// fillRect forces the pixels in rect to be char, even if char is transparent.
func (p *painterBoard) fillRect(rect geo.Rect, char rune, highlight Highlight) {
	overlap, ok := p.bound.OverlappedRect(rect)
	if !ok {
		return
	}
	start := overlap.Position.Minus(p.bound.Position)
	for r := range overlap.Height() {
		for c := range overlap.Width() {
			p.matrix[start.Row()+r][start.Column()+c].set(char, char, highlight)
		}
	}
}

func (p *painterBoard) set(point geo.Point, char rune, highlight Highlight) {
	if pixel := p.pixel(point.Left, point.Top); pixel != nil {
		pixel.set(char, char, highlight)
	}
}

func (b *Board) fillRect(rect geo.Rect, char rune, highlight Highlight) {
	for _, painter := range b.createOverlappedBoards(rect) {
		painter.fillRect(rect, char, highlight)
	}
}

func (b *Board) set(point geo.Point, char rune, highlight Highlight) {
	for _, painter := range b.createOverlappedBoards(geo.Rect{Position: point, Size: geo.Size{Width: 1, Height: 1}}) {
		painter.set(point, char, highlight)
	}
}

// grid builds a width x height text of spaces with the given rows replaced.
func grid(width, height int, rows map[int]string) string {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = rows[i] + strings.Repeat(" ", width-len([]rune(rows[i])))
	}
	return strings.Join(lines, "\n")
}

func newTestBoard() *Board {
	target := New()
	target.ClearAndSetWindow(geo.RectByLTWH(-100, -100, 200, 200))
	return target
}

func TestBoard_GetSet(t *testing.T) {
	target := newTestBoard()

	values := []int{-48, -32, -18, -16, 0, 16, 18, 32, 48}
	for _, value := range values {
		assert.Equal(t, Pixel{}, target.Get(value, value))
	}
	chars := []rune("012345678")
	for i, value := range values {
		target.set(geo.Point{Left: value, Top: value}, chars[i], HighlightNone)
	}
	for i, value := range values {
		assert.Equal(t, chars[i], target.Get(value, value).Visual)
	}
	assert.Equal(t, 7, len(target.painterBoards))
}

func TestBoard_Fill(t *testing.T) {
	target := newTestBoard()

	target.fillRect(geo.RectByLTWH(1, 1, 3, 3), 'A', HighlightNone)
	assert.Equal(t, grid(16, 16, map[int]string{
		1: " AAA",
		2: " AAA",
		3: " AAA",
	}), target.String())
	assert.Equal(t, 1, len(target.painterBoards))

	target.fillRect(geo.RectByLTWH(-3, -3, 3, 3), 'B', HighlightNone)
	assert.Equal(t, grid(32, 32, map[int]string{
		13: "             BBB",
		14: "             BBB",
		15: "             BBB",
		17: "                 AAA",
		18: "                 AAA",
		19: "                 AAA",
	}), target.String())
	assert.Equal(t, 2, len(target.painterBoards))

	target.fillRect(geo.RectByLTWH(-1, 0, 3, 1), 'C', HighlightNone)
	assert.Equal(t, grid(32, 32, map[int]string{
		13: "             BBB",
		14: "             BBB",
		15: "             BBB",
		16: "               CCC",
		17: "                 AAA",
		18: "                 AAA",
		19: "                 AAA",
	}), target.String())
	assert.Equal(t, 3, len(target.painterBoards))
}

func TestBoard_FillCrossing(t *testing.T) {
	target := newTestBoard()

	horizontal := bitmap.NewBuilder(3, 1)
	horizontal.FillChar('─')
	vertical := bitmap.NewBuilder(1, 3)
	vertical.FillChar('│')

	target.Fill(geo.Point{Left: 0, Top: 1}, horizontal.Bitmap(), HighlightNone)
	target.Fill(geo.Point{Left: 1, Top: 0}, vertical.Bitmap(), HighlightNone)

	assert.Equal(t, "   \n │ \n─┼─\n │ ", target.StringInBound(geo.RectByLTWH(0, -1, 3, 4)))
}

func TestPainterBoard_FillInside(t *testing.T) {
	board := newPainterBoard(geo.RectByLTWH(0, 0, 4, 4))
	board.fillRect(geo.RectByLTWH(1, 2, 3, 2), '#', HighlightNone)
	assert.Equal(t, "    \n    \n ###\n ###", board.String())
}

func TestPainterBoard_FillOutside(t *testing.T) {
	board := newPainterBoard(geo.RectByLTWH(1, 1, 4, 4))
	board.fillRect(geo.RectByLTWH(0, 0, 5, 5), '#', HighlightNone)
	assert.Equal(t, "####\n####\n####\n####", board.String())
}

func TestPainterBoard_FillBoard(t *testing.T) {
	board1 := newPainterBoard(geo.RectByLTWH(0, 0, 4, 4))
	board1.fillRect(geo.RectByLTWH(0, 0, 2, 2), 'a', HighlightNone)
	board1.fillRect(geo.RectByLTWH(0, 2, 2, 2), 'b', HighlightNone)
	board1.fillRect(geo.RectByLTWH(2, 0, 2, 2), 'c', HighlightNone)
	board1.fillRect(geo.RectByLTWH(2, 2, 2, 2), 'd', HighlightNone)
	board1.set(geo.Point{Left: 2, Top: 1}, bitmap.Transparent, HighlightNone)

	board2 := newPainterBoard(geo.RectByLTWH(1, 1, 3, 2))
	board2.set(geo.Point{Left: 2, Top: 1}, 'x', HighlightNone)
	board2.fillBoard(board1)
	assert.Equal(t, "axc\nbdd", board2.String())
}

func TestPainterBoard_FillBitmap(t *testing.T) {
	builder := bitmap.NewBuilder(2, 2)
	builder.Put(0, 0, 'a', 'a')
	builder.Put(0, 1, 'b', 'b')

	board := newPainterBoard(geo.RectByLTWH(0, 0, 3, 3))
	board.set(geo.Point{Left: 2, Top: 2}, 'x', HighlightNone)
	board.fillBitmap(geo.Point{Left: 1, Top: 1}, builder.Bitmap(), HighlightNone)
	assert.Equal(t, "   \n ab\n  x", board.String())
}

func TestExcludeMask(t *testing.T) {
	for _, char := range []rune{'│', '┃', '║'} {
		assert.Equal(t, 0b001100110011, excludeMask(charMask(char, maskCross)), string(char))
	}
	for _, char := range []rune{'─', '━', '═'} {
		assert.Equal(t, 0b110011001100, excludeMask(charMask(char, maskCross)), string(char))
	}
}
