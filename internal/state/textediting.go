package state

import (
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// minFreeTextWidth is the narrowest text editor for a free text.
const minFreeTextWidth = 30

// TextEditing is a text being edited, of a text shape or a field of a widget. JS shows a text
// editor at BoundPx with Text, and calls UpdateEditingText and FinishTextEditing.
type TextEditing struct {
	Text    string
	BoundPx RectF
	// IsSingleLine finishes editing on Enter, as a field of a widget has one line.
	IsSingleLine bool

	onChange func(text string)
	onFinish func(text string)
}

type RectF struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// startTextEditing asks JS to show the text editor for target. onFinish is called with the
// final text when the editor is closed.
func (m *Manager) startTextEditing(target *shape.Text, isFreeText bool, onFinish func(text string)) {
	contentBound := target.ContentBound()
	contentWidth := contentBound.Width()
	contentHeight := contentBound.Height()
	if isFreeText {
		contentWidth = max(m.WindowBound().Width()-contentBound.Left(), minFreeTextWidth)
		contentHeight = 4
	}

	m.showTextEditor(&TextEditing{
		Text:    target.Text(),
		BoundPx: m.boundPx(geo.Rect{Position: contentBound.Position, Size: geo.Size{Width: contentWidth, Height: contentHeight}}),
		onChange: func(text string) {
			m.shapeManager.Execute(shape.ChangeText{Target: target, NewText: text})
			if !isFreeText {
				m.adjustNormalTextSize(target)
				m.addSelectedShape(target)
			}
		},
		onFinish: func(string) { onFinish(target.Text()) },
	})
}

func (m *Manager) boundPx(bound geo.Rect) RectF {
	return RectF{
		Left:   m.drawingInfo.toXPx(float64(bound.Left())),
		Top:    m.drawingInfo.toYPx(float64(bound.Top())),
		Width:  m.drawingInfo.toWidthPx(float64(bound.Width())),
		Height: m.drawingInfo.toHeightPx(float64(bound.Height())),
	}
}

func (m *Manager) showTextEditor(editing *TextEditing) {
	m.textEditing = editing
	m.notify(ChangeTextEditing)
}

// UpdateEditingText is called by JS when the text in the editor changes.
func (m *Manager) UpdateEditingText(text string) {
	editing := m.textEditing
	if editing == nil {
		return
	}
	editing.Text = text
	editing.onChange(text)
}

// FinishTextEditing is called by JS when the text editor is closed.
func (m *Manager) FinishTextEditing() {
	editing := m.textEditing
	if editing == nil {
		return
	}
	m.textEditing = nil
	m.notify(ChangeTextEditing)
	editing.onFinish(editing.Text)
}

// adjustNormalTextSize grows the text height to show all lines.
func (m *Manager) adjustNormalTextSize(text *shape.Text) {
	height := len(text.RenderableText().Lines())
	if text.Extra().HasBorder() {
		height += 2
	}
	bound := text.Bound()
	newBound := geo.Rect{Position: bound.Position, Size: geo.Size{Width: bound.Width(), Height: max(height, bound.Height())}}
	m.shapeManager.Execute(shape.ChangeBound{Target: text, NewBound: newBound})
}
