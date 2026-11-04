package shape

import (
	"encoding/json"
	"fmt"

	"github.com/bluemir/nemokan/internal/geo"
)

// SerializableShape is the stored form of a shape: *SerializableRectangle, *SerializableText,
// *SerializableLine or *SerializableGroup. In JSON, "type" tells which one.
type SerializableShape interface {
	// OriginalID is the id of the shape when it was serialized, even if the id is temporary.
	OriginalID() string
	shapeType() string
}

type serializableBase struct {
	// ID is empty if the shape has no id yet.
	ID string `json:"id,omitempty"`
	// IsIDTemporary means ID is only a reference, for example for copy and paste. A new id
	// is assigned when the shape is restored.
	IsIDTemporary bool `json:"isIdTemporary,omitempty"`
	VersionCode   int  `json:"versionCode"`

	// Name, IsHidden and IsLocked are set in the layers. They are left out when unset, so
	// files without them are still read. See ADR-0013.
	Name     string `json:"name,omitempty"`
	IsHidden bool   `json:"isHidden,omitempty"`
	IsLocked bool   `json:"isLocked,omitempty"`
}

func (b serializableBase) OriginalID() string { return b.ID }

func (b serializableBase) actualID() string {
	if b.IsIDTemporary {
		return ""
	}
	return b.ID
}

type SerializableRectangle struct {
	serializableBase
	Bound geo.Rect                   `json:"bound"`
	Extra SerializableRectangleExtra `json:"extra"`
}

type SerializableText struct {
	serializableBase
	Bound          geo.Rect              `json:"bound"`
	Text           string                `json:"text"`
	Extra          SerializableTextExtra `json:"extra"`
	IsTextEditable bool                  `json:"isTextEditable"`
}

type SerializableLine struct {
	serializableBase
	StartPoint    geo.DirectedPoint     `json:"startPoint"`
	EndPoint      geo.DirectedPoint     `json:"endPoint"`
	JointPoints   []geo.Point           `json:"jointPoints"`
	Extra         SerializableLineExtra `json:"extra"`
	WasMovingEdge bool                  `json:"wasMovingEdge"`
}

// SerializableWidget keeps the content as the JSON of its kind. See ADR-0014.
type SerializableWidget struct {
	serializableBase
	Bound   geo.Rect                   `json:"bound"`
	Kind    WidgetKind                 `json:"kind"`
	Content json.RawMessage            `json:"content"`
	Extra   SerializableRectangleExtra `json:"extra"`
}

type SerializableGroup struct {
	serializableBase
	Shapes SerializableShapes `json:"shapes"`
}

func (*SerializableRectangle) shapeType() string { return "rectangle" }
func (*SerializableText) shapeType() string      { return "text" }
func (*SerializableLine) shapeType() string      { return "line" }
func (*SerializableGroup) shapeType() string     { return "group" }
func (*SerializableWidget) shapeType() string    { return "widget" }

// SerializableShapes adds "type" to each shape in JSON.
type SerializableShapes []SerializableShape

func (shapes SerializableShapes) MarshalJSON() ([]byte, error) {
	items := make([]json.RawMessage, len(shapes))
	for i, shape := range shapes {
		item, err := marshalSerializableShape(shape)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	return json.Marshal(items)
}

func (shapes *SerializableShapes) UnmarshalJSON(data []byte) error {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	result := make(SerializableShapes, len(items))
	for i, item := range items {
		shape, err := unmarshalSerializableShape(item)
		if err != nil {
			return err
		}
		result[i] = shape
	}
	*shapes = result
	return nil
}

func marshalSerializableShape(shape SerializableShape) ([]byte, error) {
	data, err := json.Marshal(shape)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	fields["type"], _ = json.Marshal(shape.shapeType())
	return json.Marshal(fields)
}

func unmarshalSerializableShape(data []byte) (SerializableShape, error) {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	var shape SerializableShape
	switch header.Type {
	case "rectangle":
		shape = &SerializableRectangle{}
	case "text":
		shape = &SerializableText{}
	case "line":
		shape = &SerializableLine{}
	case "group":
		shape = &SerializableGroup{}
	case "widget":
		shape = &SerializableWidget{}
	default:
		return nil, fmt.Errorf("unknown shape type: %q", header.Type)
	}
	if err := json.Unmarshal(data, shape); err != nil {
		return nil, err
	}
	if widget, ok := shape.(*SerializableWidget); ok {
		// An unknown kind or a broken content cannot be restored.
		if _, err := ParseWidgetContent(widget.Kind, widget.Content); err != nil {
			return nil, err
		}
	}
	return shape, nil
}

// SerializableRectangleExtra stores the style ids instead of the styles.
type SerializableRectangleExtra struct {
	IsFillEnabled   bool                         `json:"isFillEnabled"`
	FillStyleID     string                       `json:"fillStyleId"`
	IsBorderEnabled bool                         `json:"isBorderEnabled"`
	BorderStyleID   string                       `json:"borderStyleId"`
	DashPattern     StraightStrokeDashPattern    `json:"dashPattern"`
	Corner          RectangleBorderCornerPattern `json:"corner"`
}

func serializableRectangleExtra(extra RectangleExtra) SerializableRectangleExtra {
	return SerializableRectangleExtra{
		IsFillEnabled:   extra.IsFillEnabled,
		FillStyleID:     extra.UserSelectedFillStyle.ID,
		IsBorderEnabled: extra.IsBorderEnabled,
		BorderStyleID:   extra.UserSelectedBorderStyle.ID,
		DashPattern:     extra.DashPattern,
		Corner:          extra.Corner,
	}
}

// restore uses the current default style for an unknown style id.
func (e SerializableRectangleExtra) restore() RectangleExtra {
	return RectangleExtra{
		IsFillEnabled:           e.IsFillEnabled,
		UserSelectedFillStyle:   RectangleFillStyleOr(e.FillStyleID, defaultRectangleExtra.UserSelectedFillStyle),
		IsBorderEnabled:         e.IsBorderEnabled,
		UserSelectedBorderStyle: StrokeStyleOr(e.BorderStyleID, defaultRectangleExtra.UserSelectedBorderStyle),
		DashPattern:             e.DashPattern,
		Corner:                  e.Corner,
	}
}

type SerializableTextExtra struct {
	BoundExtra SerializableRectangleExtra `json:"boundExtra"`
	TextAlign  TextAlign                  `json:"textAlign"`
}

func serializableTextExtra(extra TextExtra) SerializableTextExtra {
	return SerializableTextExtra{
		BoundExtra: serializableRectangleExtra(extra.BoundExtra),
		TextAlign:  extra.TextAlign,
	}
}

func (e SerializableTextExtra) restore() TextExtra {
	return TextExtra{BoundExtra: e.BoundExtra.restore(), TextAlign: e.TextAlign}
}

type SerializableLineExtra struct {
	IsStrokeEnabled      bool                      `json:"isStrokeEnabled"`
	StrokeStyleID        string                    `json:"strokeStyleId"`
	IsStartAnchorEnabled bool                      `json:"isStartAnchorEnabled"`
	StartAnchorID        string                    `json:"startAnchorId"`
	IsEndAnchorEnabled   bool                      `json:"isEndAnchorEnabled"`
	EndAnchorID          string                    `json:"endAnchorId"`
	DashPattern          StraightStrokeDashPattern `json:"dashPattern"`
	IsRoundedCorner      bool                      `json:"isRoundedCorner"`
}

func serializableLineExtra(extra LineExtra) SerializableLineExtra {
	return SerializableLineExtra{
		IsStrokeEnabled:      extra.IsStrokeEnabled,
		StrokeStyleID:        extra.UserSelectedStrokeStyle.ID,
		IsStartAnchorEnabled: extra.IsStartAnchorEnabled,
		StartAnchorID:        extra.UserSelectedStartAnchor.ID,
		IsEndAnchorEnabled:   extra.IsEndAnchorEnabled,
		EndAnchorID:          extra.UserSelectedEndAnchor.ID,
		DashPattern:          extra.DashPattern,
		IsRoundedCorner:      extra.IsRoundedCorner,
	}
}

// restore uses the current default style for an unknown style id.
func (e SerializableLineExtra) restore() LineExtra {
	return LineExtra{
		IsStrokeEnabled:         e.IsStrokeEnabled,
		UserSelectedStrokeStyle: StrokeStyleOr(e.StrokeStyleID, defaultLineExtra.UserSelectedStrokeStyle),
		IsStartAnchorEnabled:    e.IsStartAnchorEnabled,
		UserSelectedStartAnchor: AnchorCharOr(e.StartAnchorID, defaultLineExtra.UserSelectedStartAnchor),
		IsEndAnchorEnabled:      e.IsEndAnchorEnabled,
		UserSelectedEndAnchor:   AnchorCharOr(e.EndAnchorID, defaultLineExtra.UserSelectedEndAnchor),
		DashPattern:             e.DashPattern,
		IsRoundedCorner:         e.IsRoundedCorner,
	}
}

// SerializableLineConnector is the stored form of a LineConnector.
type SerializableLineConnector struct {
	LineID   string     `json:"lineId"`
	Anchor   Anchor     `json:"anchor"`
	TargetID string     `json:"targetId"`
	Ratio    geo.PointF `json:"ratio"`
	Offset   geo.Point  `json:"offset"`
}
