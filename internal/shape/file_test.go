package shape

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/nemokan/internal/geo"
)

func TestFile_JSON(t *testing.T) {
	manager := NewManager()
	rectangle := NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	line := NewLine(directed(geo.Horizontal, 5, 2), directed(geo.Horizontal, 20, 2), "", "")
	manager.Execute(AddShape{Shape: rectangle})
	manager.Execute(AddShape{Shape: line})
	manager.ShapeConnector().AddConnector(line, AnchorStart, rectangle)

	file := NewFile("diagram", point(3, 4), manager.Root().Serializable(true).(*SerializableGroup), manager.ShapeConnector().Serializable())
	data, err := json.Marshal(file)
	assert.NoError(t, err)

	parsed, err := ParseFile(data)
	assert.NoError(t, err)
	assert.Equal(t, file, parsed)

	restored := NewManager()
	restored.ReplaceRoot(RestoreRoot(parsed.Root), RestoreShapeConnector(parsed.Connectors))
	assert.Len(t, restored.Root().Items(), 2)
	assert.True(t, restored.ShapeConnector().HasConnector(line, AnchorStart))
}

func TestParseFile_Invalid(t *testing.T) {
	for _, testCase := range []struct {
		name string
		data string
	}{
		{"not json", `{`},
		{"unknown version", `{"version":2,"root":{"shapes":[]}}`},
		{"no root", `{"version":1}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParseFile([]byte(testCase.data))
			assert.Error(t, err)
		})
	}
}

func TestFile_AsCopy(t *testing.T) {
	manager := NewManager()
	rectangle := NewRectangle(geo.RectByLTWH(0, 0, 5, 5), "", "")
	line := NewLine(directed(geo.Horizontal, 5, 2), directed(geo.Horizontal, 20, 2), "", "")
	manager.Execute(AddShape{Shape: rectangle})
	manager.Execute(AddShape{Shape: line})
	manager.ShapeConnector().AddConnector(line, AnchorStart, rectangle)
	file := NewFile("diagram", point(3, 4), manager.Root().Serializable(true).(*SerializableGroup), manager.ShapeConnector().Serializable())

	copied := file.AsCopy()

	assert.NotEqual(t, file.Root.OriginalID(), copied.Root.OriginalID())
	assert.False(t, copied.Root.IsIDTemporary)
	assert.Equal(t, "diagram", copied.Name)
	assert.Equal(t, file.Connectors, copied.Connectors)
	// The original is not changed.
	assert.Equal(t, manager.Root().ID(), file.Root.OriginalID())

	restored := NewManager()
	restored.ReplaceRoot(RestoreRoot(copied.Root), RestoreShapeConnector(copied.Connectors))
	assert.Equal(t, copied.Root.OriginalID(), restored.Root().ID())
	assert.True(t, restored.ShapeConnector().HasConnector(line, AnchorStart))
}
