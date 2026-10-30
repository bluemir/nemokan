// Package clipboard converts shapes to clipboard text and back. Reading and writing the
// browser clipboard is done by JS.
package clipboard

import (
	"encoding/json"
	"strings"

	"github.com/bluemir/monosketch/internal/bitmap"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

const (
	defaultTextBoundWidth = 400
	nonBreakingSpace      = ' '
)

// Object is the shapes and their connectors in the clipboard.
type Object struct {
	Shapes     shape.SerializableShapes          `json:"shapes"`
	Connectors []shape.SerializableLineConnector `json:"connectors"`
}

func (o Object) Text() (string, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Parse reads pasted text. Text which is not an Object becomes a text shape. Blank text
// returns false.
func Parse(text string) (Object, bool) {
	if strings.TrimSpace(text) == "" {
		return Object{}, false
	}
	var object Object
	if err := json.Unmarshal([]byte(text), &object); err == nil && object.Shapes != nil {
		return object, true
	}
	return Object{Shapes: shape.SerializableShapes{textShape(text)}}, true
}

func textShape(text string) shape.SerializableShape {
	// Spaces are replaced with non-breaking spaces so the text keeps its spaces when drawn.
	// Then the text is not wrapped at spaces, only chunked by cells.
	toBeUsedText := strings.ReplaceAll(text, " ", string(nonBreakingSpace))

	width := 0
	height := 0
	for _, line := range strings.Split(toBeUsedText, "\n") {
		// Wide chars take two cells. See ADR-0008.
		lines := (&shape.RenderableText{Text: line, MaxRowWidth: defaultTextBoundWidth}).Lines()
		height += len(lines)
		for _, wrapped := range lines {
			width = max(width, bitmap.StringWidth(wrapped))
		}
	}

	textShape := shape.NewText(geo.RectByLTWH(0, 0, width, height), "", "", true)
	textShape.SetExtra(shape.TextExtraNoBound)
	textShape.SetText(toBeUsedText)
	return textShape.Serializable(false)
}
