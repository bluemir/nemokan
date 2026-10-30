package board

import "github.com/bluemir/monosketch/internal/bitmap"

type Highlight int

const (
	HighlightNone Highlight = iota
	HighlightSelected
	HighlightTextEditing
	HighlightLineConnectFocusing
)

// Pixel is one cell on the board.
type Pixel struct {
	Visual    rune
	Direction rune
	Highlight Highlight
}

func (p Pixel) IsTransparent() bool {
	return p.Visual == bitmap.Transparent || p.Visual == bitmap.HalfTransparent
}

func (p *Pixel) set(visual, direction rune, highlight Highlight) {
	p.Visual = visual
	p.Direction = direction
	p.Highlight = highlight
}

// String is empty for the tail of a wide char.
func (p Pixel) String() string {
	if p.Visual == bitmap.WideCharTail {
		return ""
	}
	if p.IsTransparent() {
		return " "
	}
	return string(p.Visual)
}
