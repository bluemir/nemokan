package action

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/cursor"
	"github.com/bluemir/monosketch/internal/shape"
)

func TestParseOneTime(t *testing.T) {
	horizontalAlign := shape.AlignRight
	width := 10
	for _, testCase := range []struct {
		json     string
		expected OneTime
	}{
		{`{"type":"undo"}`, Undo{}},
		{`{"type":"cut"}`, Cut{}},
		{`{"type":"moveShapes","offsetRow":0,"offsetColumn":-1}`, MoveShapes{OffsetRow: 0, OffsetColumn: -1}},
		{`{"type":"textAlignment","newHorizontalAlign":"right"}`, TextAlignment{NewHorizontalAlign: &horizontalAlign}},
		{`{"type":"changeShapeBound","newWidth":10}`, ChangeShapeBound{NewWidth: &width}},
		{`{"type":"reorderShape","orderType":"front"}`, ReorderShape{OrderType: shape.OrderFront}},
	} {
		parsed, err := ParseOneTime([]byte(testCase.json))
		assert.NoError(t, err, testCase.json)
		assert.Equal(t, testCase.expected, parsed, testCase.json)
	}
}

func TestParseOneTime_Invalid(t *testing.T) {
	for _, data := range []string{
		`{`,
		`{"type":"unknown"}`,
		// EditSelectedShape is only created in Go.
		`{"type":"editSelectedShape"}`,
		`{"type":"reorderShape","orderType":"sideways"}`,
		// Document actions are handled by JS. See ADR-0010.
		`{"type":"switchDocument","documentId":"p1"}`,
	} {
		_, err := ParseOneTime([]byte(data))
		assert.Error(t, err, data)
	}
}

func TestManager_SetOneTime(t *testing.T) {
	manager := NewManager()
	var notified []OneTime
	manager.OneTime.Observe(func(action OneTime) { notified = append(notified, action) })

	manager.SetOneTime(Redo{})
	assert.Equal(t, []OneTime{Idle{}, Redo{}, Idle{}}, notified)
	assert.Equal(t, Idle{}, manager.OneTime.Value())
}

func TestRetainable_Cursor(t *testing.T) {
	assert.Equal(t, cursor.Default, RetainableIdle.Cursor())
	assert.Equal(t, cursor.Crosshair, AddRectangle.Cursor())
	assert.Equal(t, cursor.Text, AddText.Cursor())
	assert.Equal(t, cursor.Crosshair, AddLine.Cursor())
}
