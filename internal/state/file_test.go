package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

func TestExportText(t *testing.T) {
	state := newTestState(t)
	assert.Equal(t, "", state.ExportText())

	state.addRectangle(point(1, 1), point(4, 3))
	state.Paste("한글")
	state.click(point(30, 20), false)

	assert.Equal(t, ""+
		"┌──┐       \n"+
		"│  │       \n"+
		"└──┘       \n"+
		"           \n"+
		"           \n"+
		"       한글", state.ExportText())

	_, ok := state.SelectedShapesText()
	assert.False(t, ok)
}

func TestCopyText(t *testing.T) {
	state := newTestState(t)
	state.addRectangle(point(1, 1), point(4, 3))

	state.actions.SetOneTime(action.CopyText{})
	text, ok := state.TakeClipboardText()
	assert.True(t, ok)
	assert.Equal(t, "┌──┐\n│  │\n└──┘", text)
}

func TestFile_SaveAndOpen(t *testing.T) {
	source := newTestState(t)
	source.addRectangle(point(1, 1), point(4, 3))
	data, err := json.Marshal(source.File("diagram", point(10, 20)))
	assert.NoError(t, err)

	file, err := shape.ParseFile(data)
	assert.NoError(t, err)
	assert.Equal(t, "diagram", file.Name)
	assert.Equal(t, point(10, 20), file.Offset)

	opened := newTestState(t)
	opened.OpenFile(file)
	assert.Equal(t, source.ShapeManager().Root().ID(), opened.ShapeManager().Root().ID())
	assert.Equal(t, geo.RectByLTRB(1, 1, 4, 3), opened.ShapeManager().Root().Items()[0].Bound())

	opened.OpenFileAsCopy(file)
	assert.NotEqual(t, source.ShapeManager().Root().ID(), opened.ShapeManager().Root().ID())
	assert.Equal(t, 1, opened.ShapeManager().Root().ItemCount())
}
