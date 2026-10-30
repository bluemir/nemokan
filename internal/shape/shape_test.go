package shape

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bluemir/monosketch/internal/geo"
)

const testParentID = "1"

// mockShape is a shape with only a bound, for testing.
type mockShape struct {
	base
	bound geo.Rect
}

func newMockShape(bound geo.Rect, parentID string) *mockShape {
	return &mockShape{base: newBase("", parentID), bound: bound}
}

func (m *mockShape) Bound() geo.Rect { return m.bound }
func (m *mockShape) SetBound(newBound geo.Rect) {
	m.update(func() bool {
		isUpdated := m.bound != newBound
		m.bound = newBound
		return isUpdated
	})
}
func (m *mockShape) SetExtra(Extra)                      { panic("not supported") }
func (m *mockShape) Contains(point geo.Point) bool       { return m.bound.Contains(point) }
func (m *mockShape) IsVertex(point geo.Point) bool       { return m.bound.IsVertex(point) }
func (m *mockShape) IsOverlapped(rect geo.Rect) bool     { return m.bound.IsOverlapped(rect) }
func (m *mockShape) Serializable(bool) SerializableShape { panic("not supported") }

func TestRectangle_Serializable(t *testing.T) {
	rectangle := NewRectangle(geo.RectByLTWH(1, 2, 3, 4), "", testParentID)

	serializable := rectangle.Serializable(true).(*SerializableRectangle)
	assert.Equal(t, rectangle.Bound(), serializable.Bound)
	assert.Equal(t, serializableRectangleExtra(rectangle.Extra()), serializable.Extra)

	rectangle.SetBound(geo.RectByLTWH(5, 6, 7, 8))
	serializable = rectangle.Serializable(true).(*SerializableRectangle)
	assert.Equal(t, rectangle.Bound(), serializable.Bound)
}

func TestRectangle_Restore(t *testing.T) {
	rectangle := NewRectangle(geo.RectByLTWH(1, 2, 3, 4), "", testParentID)
	rectangle.SetBound(geo.RectByLTWH(5, 6, 7, 8))

	restored := restoreRectangle(rectangle.Serializable(true).(*SerializableRectangle), testParentID)
	assert.Equal(t, rectangle.ID(), restored.ID())
	assert.Equal(t, rectangle.Bound(), restored.Bound())
	assert.Equal(t, rectangle.Extra(), restored.Extra())
	assert.Equal(t, rectangle.VersionCode(), restored.VersionCode())
	assert.Equal(t, testParentID, restored.ParentID())
}

func TestText_Serializable(t *testing.T) {
	text := NewText(geo.RectByLTWH(1, 2, 3, 4), "", testParentID, true)
	text.SetBound(geo.RectByLTWH(5, 6, 7, 8))
	text.SetText("Hello Hello!")

	serializable := text.Serializable(true).(*SerializableText)
	assert.Equal(t, text.Text(), serializable.Text)
	assert.Equal(t, text.Bound(), serializable.Bound)
	assert.Equal(t, serializableTextExtra(text.Extra()), serializable.Extra)
}

func TestText_Restore(t *testing.T) {
	text := NewText(geo.RectByLTWH(1, 2, 3, 4), "", testParentID, true)
	text.SetText("Hello Hello!")
	text.SetBound(geo.RectByLTWH(5, 5, 2, 2))

	restored := restoreText(text.Serializable(true).(*SerializableText), testParentID)
	assert.Equal(t, testParentID, restored.ParentID())
	assert.Equal(t, text.Text(), restored.Text())
	assert.Equal(t, text.Bound(), restored.Bound())
	assert.Equal(t, text.Extra(), restored.Extra())
	assert.Equal(t, text.RenderableText().Lines(), restored.RenderableText().Lines())
}

func TestText_RenderableText(t *testing.T) {
	target := NewText(geo.RectByLTWH(0, 0, 5, 5), "", "", true)

	target.SetText("0 1234 12345\n1   2 3 4 5678901 23")
	target.SetExtra(TextExtraNoBound)
	assert.Equal(t,
		[]string{"0", "1234", "12345", "1   2", "3 4", "56789", "01 23"},
		target.RenderableText().Lines(),
	)
}

func TestText_RenderableText_WideChars(t *testing.T) {
	for _, testCase := range []struct {
		text     string
		width    int
		expected []string
	}{
		// "안녕" takes 4 cells.
		{"안녕 하세요", 5, []string{"안녕", "하세", "요"}},
		{"ab안녕", 5, []string{"ab안", "녕"}},
		{"가a나", 3, []string{"가a", "나"}},
	} {
		renderable := &RenderableText{Text: testCase.text, MaxRowWidth: testCase.width}
		assert.Equal(t, testCase.expected, renderable.Lines(), testCase.text)
	}
}

func TestText_ContentBound(t *testing.T) {
	target := NewText(geo.RectByLTWH(1, 2, 5, 6), "", "", true)

	extra := target.Extra()
	extra.BoundExtra.IsBorderEnabled = true
	target.SetExtra(extra)
	assert.Equal(t, geo.RectByLTWH(2, 3, 3, 4), target.ContentBound())

	extra.BoundExtra.IsBorderEnabled = false
	target.SetExtra(extra)
	assert.Equal(t, geo.RectByLTWH(1, 2, 5, 6), target.ContentBound())
}

func TestGroup_Add(t *testing.T) {
	target := NewGroup("", "100")
	invalidShape := NewRectangle(geo.Rect{}, "", "10000")
	validShape1 := NewRectangle(geo.Rect{}, "", "")
	validShape2 := NewRectangle(geo.Rect{}, "", target.ID())
	validShape3 := NewRectangle(geo.Rect{}, "", "")

	target.add(invalidShape)
	assert.Equal(t, 0, target.ItemCount())

	target.add(validShape1)
	assert.Equal(t, target.ID(), validShape1.ParentID())
	assert.Equal(t, []Shape{validShape1}, target.Items())

	// Adding an existing shape again does nothing.
	target.add(validShape1)
	assert.Equal(t, []Shape{validShape1}, target.Items())

	target.add(validShape2)
	assert.Equal(t, target.ID(), validShape2.ParentID())
	assert.Equal(t, []Shape{validShape1, validShape2}, target.Items())

	target.addAfter(validShape3, validShape1.ID())
	assert.Equal(t, target.ID(), validShape3.ParentID())
	assert.Equal(t, []Shape{validShape1, validShape3, validShape2}, target.Items())
}

func TestGroup_Remove(t *testing.T) {
	target := NewGroup("", "100")
	shape1 := NewRectangle(geo.Rect{}, "", "")
	shape2 := NewRectangle(geo.Rect{}, "", "")
	target.add(shape1)
	target.add(shape2)

	target.remove(shape1)
	assert.Equal(t, []Shape{shape2}, target.Items())

	target.remove(shape2)
	assert.Equal(t, 0, target.ItemCount())
}

func TestGroup_ChangeOrder(t *testing.T) {
	shape1 := NewRectangle(geo.Rect{}, "", "")
	shape2 := NewRectangle(geo.Rect{}, "", "")
	shape3 := NewRectangle(geo.Rect{}, "", "")

	for _, testCase := range []struct {
		name     string
		target   Shape
		action   moveAction
		expected []Shape
	}{
		{"up", shape1, moveUp, []Shape{shape2, shape1, shape3}},
		{"down", shape3, moveDown, []Shape{shape1, shape3, shape2}},
		{"top", shape1, moveTop, []Shape{shape2, shape3, shape1}},
		{"bottom", shape3, moveBottom, []Shape{shape3, shape1, shape2}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target := NewGroup("", "100")
			for _, shape := range []*Rectangle{shape1, shape2, shape3} {
				shape.setParentID("")
				target.add(shape)
			}
			target.changeOrder(testCase.target, testCase.action)
			assert.Equal(t, testCase.expected, target.Items())
		})
	}
}

func newTestLine() *Line {
	return NewLine(directed(geo.Vertical, 1, 2), directed(geo.Horizontal, 3, 4), "", testParentID)
}

func moveTestLineAnchors(line *Line) {
	line.MoveAnchorPoint(AnchorPointUpdate{Anchor: AnchorStart, Point: directed(geo.Horizontal, 4, 5)}, true, false)
	line.MoveAnchorPoint(AnchorPointUpdate{Anchor: AnchorEnd, Point: directed(geo.Vertical, 7, 8)}, true, false)
}

func TestLine_Serializable(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		change        func(*Line)
		wasMovingEdge bool
	}{
		{"init", func(*Line) {}, false},
		{"move anchor points", moveTestLineAnchors, false},
		{"move edge", func(line *Line) { line.MoveEdge(line.Edges()[0].ID, point(10, 10), true) }, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			line := newTestLine()
			testCase.change(line)

			serializable := line.Serializable(true).(*SerializableLine)
			assert.Equal(t, line.StartPoint(), serializable.StartPoint)
			assert.Equal(t, line.EndPoint(), serializable.EndPoint)
			assert.Equal(t, line.ReducedJointPoints(), reducePoints(serializable.JointPoints))
			assert.Equal(t, serializableLineExtra(line.Extra()), serializable.Extra)
			assert.Equal(t, testCase.wasMovingEdge, serializable.WasMovingEdge)

			restored := restoreLine(serializable, testParentID)
			assert.Equal(t, line.Direction(AnchorStart), restored.Direction(AnchorStart))
			assert.Equal(t, line.Direction(AnchorEnd), restored.Direction(AnchorEnd))
			assert.Equal(t, line.ReducedJointPoints(), restored.ReducedJointPoints())
			assert.Equal(t, line.Extra(), restored.Extra())
			assert.Equal(t, line.WasMovingEdge(), restored.WasMovingEdge())
		})
	}
}

func TestSerializableShapes_JSON(t *testing.T) {
	group := &SerializableGroup{
		serializableBase: serializableBase{IsIDTemporary: true},
		Shapes: SerializableShapes{
			&SerializableRectangle{
				serializableBase: serializableBase{IsIDTemporary: true},
				Bound:            geo.RectByLTRB(0, 0, 1, 1),
				Extra:            serializableRectangleExtra(DefaultRectangleExtra()),
			},
			newTestLine().Serializable(true),
			NewText(geo.RectByLTWH(0, 0, 3, 3), "", "", true).Serializable(true),
			&SerializableGroup{Shapes: SerializableShapes{}},
		},
	}

	data, err := json.Marshal(SerializableShapes{group})
	assert.NoError(t, err)

	var decoded SerializableShapes
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, SerializableShapes{group}, decoded)
}

func TestSerializableShapes_UnknownType(t *testing.T) {
	var decoded SerializableShapes
	assert.Error(t, json.Unmarshal([]byte(`[{"type":"circle"}]`), &decoded))
}

func TestSerializableLineConnector_JSON(t *testing.T) {
	connector := SerializableLineConnector{
		LineID:   "line",
		Anchor:   AnchorStart,
		TargetID: "target",
		Ratio:    geo.PointF{Left: 1.2, Top: 4.5},
		Offset:   point(5, 10),
	}
	data, err := json.Marshal(connector)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"lineId":"line","anchor":"start","targetId":"target","ratio":{"left":1.2,"top":4.5},"offset":{"left":5,"top":10}}`, string(data))

	var decoded SerializableLineConnector
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, connector, decoded)
}
