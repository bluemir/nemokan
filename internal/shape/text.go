package shape

import (
	"fmt"
	"strings"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
)

// Text is a shape with a bound and a text.
// TODO: Resize bound by text
type Text struct {
	base
	bound geo.Rect
	// userSettingSize is the size set by the user. It is zero if smaller than 3x3.
	userSettingSize geo.Size
	text            string
	isTextEditable  bool
	isTextEditing   bool
	extra           TextExtra
	renderableText  *RenderableText
}

// NewText generates an id if id is empty.
func NewText(bound geo.Rect, id, parentID string, isTextEditable bool) *Text {
	text := &Text{
		base:           newBase(id, parentID),
		bound:          bound,
		isTextEditable: isTextEditable,
		extra:          DefaultTextExtra(),
		renderableText: emptyRenderableText,
	}
	text.setUserSettingSize(bound.Size)
	text.updateRenderableText()
	return text
}

// NewTextByPoints makes a text whose bound corners are startPoint and endPoint.
func NewTextByPoints(startPoint, endPoint geo.Point, id, parentID string, isTextEditable bool) *Text {
	return NewText(geo.RectByLTRB(startPoint.Left, startPoint.Top, endPoint.Left, endPoint.Top), id, parentID, isTextEditable)
}

func restoreText(serializable *SerializableText, parentID string) *Text {
	text := NewText(serializable.Bound, serializable.actualID(), parentID, serializable.IsTextEditable)
	text.extra = serializable.Extra.restore()
	text.SetText(serializable.Text)
	text.restore(serializable.serializableBase)
	return text
}

func (t *Text) Bound() geo.Rect                 { return t.bound }
func (t *Text) Text() string                    { return t.text }
func (t *Text) IsTextEditable() bool            { return t.isTextEditable }
func (t *Text) IsTextEditing() bool             { return t.isTextEditing }
func (t *Text) Extra() TextExtra                { return t.extra }
func (t *Text) RenderableText() *RenderableText { return t.renderableText }

// ContentBound is the area for the text, inside the border if there is one.
func (t *Text) ContentBound() geo.Rect {
	if t.extra.BoundExtra.IsBorderEnabled {
		return geo.RectByLTWH(t.bound.Left()+1, t.bound.Top()+1, t.bound.Width()-2, t.bound.Height()-2)
	}
	return t.bound
}

func (t *Text) setUserSettingSize(size geo.Size) {
	if size.Width >= 3 && size.Height >= 3 {
		t.userSettingSize = size
	} else {
		t.userSettingSize = geo.Size{}
	}
}

func (t *Text) SetBound(newBound geo.Rect) {
	t.update(func() bool {
		isUpdated := t.bound != newBound
		t.setUserSettingSize(newBound.Size)
		t.bound = newBound
		t.updateRenderableText()
		return isUpdated
	})
}

func (t *Text) SetText(newText string) {
	t.update(func() bool {
		isTextChanged := newText != t.text
		t.text = newText
		t.updateRenderableText()
		return isTextChanged
	})
}

func (t *Text) SetTextEditing(isEditing bool) {
	t.update(func() bool {
		isUpdated := t.isTextEditing != isEditing
		t.isTextEditing = isEditing
		return isUpdated
	})
}

func (t *Text) MakeTextEditable() {
	if t.isTextEditable {
		return
	}
	t.update(func() bool {
		t.isTextEditable = true
		return true
	})
}

func (t *Text) SetExtra(newExtra Extra) {
	extra, ok := newExtra.(TextExtra)
	if !ok {
		panic(fmt.Sprintf("new extra is not a TextExtra (%T)", newExtra))
	}
	if extra == t.extra {
		return
	}
	t.update(func() bool {
		t.extra = extra
		t.updateRenderableText()
		return true
	})
}

func (t *Text) updateRenderableText() {
	maxRowWidth := t.bound.Width()
	if t.extra.HasBorder() {
		maxRowWidth = t.bound.Width() - 2
	}
	if t.text != t.renderableText.Text || maxRowWidth != t.renderableText.MaxRowWidth {
		t.renderableText = &RenderableText{Text: t.text, MaxRowWidth: max(maxRowWidth, 1)}
	}
}

func (t *Text) Contains(point geo.Point) bool   { return t.bound.Contains(point) }
func (t *Text) IsVertex(point geo.Point) bool   { return t.bound.IsVertex(point) }
func (t *Text) IsOverlapped(rect geo.Rect) bool { return t.bound.IsOverlapped(rect) }

func (t *Text) Serializable(isIDIncluded bool) SerializableShape {
	return &SerializableText{
		serializableBase: t.serializableBase(isIDIncluded),
		Bound:            t.bound,
		Text:             t.text,
		Extra:            serializableTextExtra(t.extra),
		IsTextEditable:   t.isTextEditable,
	}
}

// RenderableText wraps Text into lines of at most MaxRowWidth cells. A wide char takes two
// cells and is never split. See ADR-0008.
type RenderableText struct {
	Text        string
	MaxRowWidth int
	lines       []string
}

var emptyRenderableText = &RenderableText{}

// Lines are computed on the first call and cached.
func (r *RenderableText) Lines() []string {
	if r.lines == nil {
		r.lines = r.createLines()
	}
	return r.lines
}

func (r *RenderableText) createLines() []string {
	if r.MaxRowWidth == 1 {
		chars := []rune(r.Text)
		lines := make([]string, len(chars))
		for i, char := range chars {
			lines[i] = string(char)
		}
		return lines
	}

	var lines []string
	for _, line := range strings.Split(r.Text, "\n") {
		adjustedLines := []string{""}
		for _, word := range r.words(line) {
			last := adjustedLines[len(adjustedLines)-1]
			space := ""
			if last != "" {
				space = " "
			}
			if bitmap.StringWidth(last)+bitmap.StringWidth(space)+bitmap.StringWidth(word) <= r.MaxRowWidth {
				adjustedLines[len(adjustedLines)-1] = last + space + word
			} else {
				adjustedLines = append(adjustedLines, word)
			}
		}
		lines = append(lines, adjustedLines...)
	}
	return lines
}

// words splits line by spaces and chunks the words wider than MaxRowWidth.
func (r *RenderableText) words(line string) []string {
	var words []string
	for _, word := range strings.Split(line, " ") {
		if bitmap.StringWidth(word) <= r.MaxRowWidth {
			words = append(words, word)
			continue
		}
		var chunk strings.Builder
		chunkWidth := 0
		for _, char := range word {
			charWidth := bitmap.CharWidth(char)
			if chunkWidth > 0 && chunkWidth+charWidth > r.MaxRowWidth {
				words = append(words, chunk.String())
				chunk.Reset()
				chunkWidth = 0
			}
			chunk.WriteRune(char)
			chunkWidth += charWidth
		}
		words = append(words, chunk.String())
	}
	return words
}
