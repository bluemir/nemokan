package clipboard

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func TestParse_Blank(t *testing.T) {
	_, ok := Parse(" \n ")
	assert.False(t, ok)
}

func TestParse_Object(t *testing.T) {
	rectangle := shape.NewRectangle(geo.RectByLTWH(1, 2, 3, 4), "", "")
	object := Object{
		Shapes:     shape.SerializableShapes{rectangle.Serializable(false)},
		Connectors: []shape.SerializableLineConnector{{LineID: "line", Anchor: shape.AnchorEnd, TargetID: rectangle.ID()}},
	}
	text, err := object.Text()
	assert.NoError(t, err)

	parsed, ok := Parse(text)
	assert.True(t, ok)
	assert.Equal(t, object, parsed)
}

func TestParse_Text(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		text         string
		expectedSize geo.Size
	}{
		{"single line", "a b", geo.Size{Width: 3, Height: 1}},
		{"multi line", "abc\n\nde", geo.Size{Width: 3, Height: 3}},
		{"long line", strings.Repeat("a", 401), geo.Size{Width: 400, Height: 2}},
		// Wide chars take two cells, so 200 fit in a row.
		{"long wide line", strings.Repeat("가", 401), geo.Size{Width: 400, Height: 3}},
		{"wide chars", "한글 a", geo.Size{Width: 6, Height: 1}},
		{"long line with spaces", strings.Repeat("a ", 201), geo.Size{Width: 400, Height: 2}},
		{"json but not an object", `[1, 2]`, geo.Size{Width: 6, Height: 1}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, ok := Parse(testCase.text)
			assert.True(t, ok)
			assert.Len(t, parsed.Shapes, 1)

			text := parsed.Shapes[0].(*shape.SerializableText)
			assert.Equal(t, testCase.expectedSize, text.Bound.Size)
			assert.Equal(t, strings.ReplaceAll(testCase.text, " ", " "), text.Text)
			assert.True(t, text.IsIDTemporary)
			assert.True(t, text.IsTextEditable)
		})
	}
}
