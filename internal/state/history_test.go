package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/shape"
)

// backup does what JS does a while after ChangeBackup.
func (s *testState) backup() {
	s.Backup()
}

func TestBackup_SkippedWhileEditingAndAfterChange(t *testing.T) {
	state := newTestState(t)
	_, ok := state.Backup()
	assert.True(t, ok)
	_, ok = state.Backup()
	assert.False(t, ok, "only once per request")

	state.enterEditingMode()
	state.addShape(shape.NewRectangle(geo.RectByLTWH(0, 0, 2, 2), "", ""))
	_, ok = state.Backup()
	assert.False(t, ok, "no request while editing")

	state.exitEditingMode()
	backup, ok := state.Backup()
	assert.True(t, ok)
	assert.Len(t, backup.Root.Shapes, 1)
}

func TestUndoRedo(t *testing.T) {
	state := newTestState(t)
	state.backup()
	state.addRectangle(point(1, 1), point(3, 3))
	state.backup()
	state.addRectangle(point(10, 1), point(12, 3))
	state.backup()
	assert.Equal(t, 2, state.ShapeManager().Root().ItemCount())

	state.actions.SetOneTime(action.Undo{})
	assert.Equal(t, 1, state.ShapeManager().Root().ItemCount())
	assert.Empty(t, state.Selection().SelectedShapes.Value())

	state.actions.SetOneTime(action.Undo{})
	assert.Equal(t, 0, state.ShapeManager().Root().ItemCount())

	// Nothing before the first backup.
	state.actions.SetOneTime(action.Undo{})
	assert.Equal(t, 0, state.ShapeManager().Root().ItemCount())

	state.actions.SetOneTime(action.Redo{})
	state.actions.SetOneTime(action.Redo{})
	assert.Equal(t, 2, state.ShapeManager().Root().ItemCount())

	// A new change drops the redo stack.
	state.actions.SetOneTime(action.Undo{})
	state.backup()
	state.addRectangle(point(20, 1), point(22, 3))
	state.backup()
	state.actions.SetOneTime(action.Redo{})
	assert.Equal(t, 2, state.ShapeManager().Root().ItemCount())
}

func TestUndo_CanceledTextIsNotKept(t *testing.T) {
	state := newTestState(t)
	state.backup()
	state.actions.SetRetainable(action.AddText)
	state.click(point(2, 2), false)
	state.FinishTextEditing()

	_, ok := state.Backup()
	assert.False(t, ok)
}

func TestRestore(t *testing.T) {
	source := newTestState(t)
	source.addRectangle(point(1, 1), point(5, 3))
	source.click(point(30, 20), false)
	source.actions.SetRetainable(action.AddLine)
	source.drag(point(5, 2), point(12, 2))
	backup, ok := source.Backup()
	assert.True(t, ok)

	state := newTestState(t)
	state.Restore("", &backup)
	assert.Equal(t, source.ShapeManager().Root().ID(), state.ShapeManager().Root().ID())
	assert.Equal(t, 2, state.ShapeManager().Root().ItemCount())
	line := state.ShapeManager().Root().Items()[1].(*shape.Line)
	assert.True(t, state.ShapeManager().ShapeConnector().HasConnector(line, shape.AnchorStart))

	state.Restore("new-document", nil)
	assert.Equal(t, "new-document", state.ShapeManager().Root().ID())
	assert.Equal(t, 0, state.ShapeManager().Root().ItemCount())
}
