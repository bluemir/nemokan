package bitmapmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/geo"
	"github.com/bluemir/nemokan/internal/shape"
)

func drawWidget(t *testing.T, kind shape.WidgetKind, content string, size geo.Size) string {
	t.Helper()
	parsed, err := shape.ParseWidgetContent(kind, []byte(content))
	assert.NoError(t, err)
	widget := shape.NewWidget(geo.Point{}, parsed, "", "")
	widget.SetBound(geo.Rect{Size: size})
	return widgetBitmap(widget.Bound().Size, widget.Layout(), widget.Extra()).String()
}

func TestWidgetBitmap(t *testing.T) {
	for _, testCase := range []struct {
		kind     shape.WidgetKind
		content  string
		size     geo.Size
		expected string
	}{
		{"button", `{"label": "Save"}`, geo.Size{Width: 10}, "[  Save  ]"},
		{"input", `{"placeholder": "Email"}`, geo.Size{Width: 12}, "[Email_____]"},
		{"checkbox", `{"label": "Remember", "isChecked": true}`, geo.Size{}, "[x] Remember"},
		{"radio", `{"label": "한글"}`, geo.Size{}, "( ) 한글"},
		{"toggle", `{"label": "Dark", "isOn": true}`, geo.Size{}, "[==o] Dark"},
		{"progress", `{"value": 50}`, geo.Size{Width: 12}, "[###---] 50%"},
		{"pagination", `{"current": 7, "total": 9}`, geo.Size{}, "< 5 6 [7] 8 9 >"},
		{"tabs", `{"tabs": ["A", "B"], "active": 1}`, geo.Size{Width: 12}, "" +
			"  A   [ B ] \n" +
			"────────────"},
		{"table", `{"columns": ["A", "B"]}`, geo.Size{Width: 11, Height: 5}, "" +
			"┌────┬────┐\n" +
			"│ A  │ B  │\n" +
			"├────┼────┤\n" +
			"│    │    │\n" +
			"└────┴────┘"},
		{"card", `{"title": "Profile"}`, geo.Size{Width: 12, Height: 4}, "" +
			"┌──────────┐\n" +
			"│ Profile  │\n" +
			"├──────────┤\n" +
			"└──────────┘"},
	} {
		t.Run(string(testCase.kind), func(t *testing.T) {
			assert.Equal(t, testCase.expected, drawWidget(t, testCase.kind, testCase.content, testCase.size))
		})
	}
}

func TestWidgetBitmap_FitsSize(t *testing.T) {
	// A checkbox takes the size of what it shows, and a button only changes the width.
	assert.Equal(t, "[ ] A", drawWidget(t, "checkbox", `{"label": "A"}`, geo.Size{Width: 20, Height: 3}))
	assert.Equal(t, "[ A ]", drawWidget(t, "button", `{"label": "A"}`, geo.Size{Width: 5, Height: 3}))
	// A button is at least as wide as its label, and a dropdown cuts it.
	assert.Equal(t, "[ Long ]", drawWidget(t, "button", `{"label": "Long"}`, geo.Size{Width: 6}))
	assert.Equal(t, "[v Lon]", drawWidget(t, "dropdown", `{"label": "Long"}`, geo.Size{Width: 7}))
}
