package action

import (
	"encoding/json"
	"fmt"

	"github.com/bluemir/monosketch/internal/shape"
)

// OneTime is an action which has effect only once. In JSON, "type" tells which one.
// Optional fields are nil when unchanged.
type OneTime interface {
	isOneTime()
}

type Idle struct{}

// Document and app setting actions are handled by JS. See ADR-0010.

// Shape actions

type SelectAllShapes struct{}
type DeselectShapes struct{}
type DeleteSelectedShapes struct{}
type EditSelectedShapes struct{}

// EditSelectedShape is only created in Go, so it has no JSON form.
type EditSelectedShape struct {
	Shape shape.Shape
}

type TextAlignment struct {
	NewHorizontalAlign *shape.HorizontalAlign `json:"newHorizontalAlign,omitempty"`
	NewVerticalAlign   *shape.VerticalAlign   `json:"newVerticalAlign,omitempty"`
}

type MoveShapes struct {
	OffsetRow    int `json:"offsetRow"`
	OffsetColumn int `json:"offsetColumn"`
}

type ChangeShapeBound struct {
	NewLeft   *int `json:"newLeft,omitempty"`
	NewTop    *int `json:"newTop,omitempty"`
	NewWidth  *int `json:"newWidth,omitempty"`
	NewHeight *int `json:"newHeight,omitempty"`
}

type ChangeShapeFillExtra struct {
	IsEnabled      *bool   `json:"isEnabled,omitempty"`
	NewFillStyleID *string `json:"newFillStyleId,omitempty"`
}

type ChangeShapeBorderExtra struct {
	IsEnabled        *bool   `json:"isEnabled,omitempty"`
	NewBorderStyleID *string `json:"newBorderStyleId,omitempty"`
}

type ChangeShapeBorderDashPatternExtra struct {
	Dash   *int `json:"dash,omitempty"`
	Gap    *int `json:"gap,omitempty"`
	Offset *int `json:"offset,omitempty"`
}

type ChangeShapeBorderCornerExtra struct {
	IsRoundedCorner bool `json:"isRoundedCorner"`
}

type ChangeLineStrokeExtra struct {
	IsEnabled        *bool   `json:"isEnabled,omitempty"`
	NewStrokeStyleID *string `json:"newStrokeStyleId,omitempty"`
}

type ChangeLineStrokeDashPatternExtra struct {
	Dash   *int `json:"dash,omitempty"`
	Gap    *int `json:"gap,omitempty"`
	Offset *int `json:"offset,omitempty"`
}

type ChangeLineStrokeCornerExtra struct {
	IsRoundedCorner bool `json:"isRoundedCorner"`
}

type ChangeLineStartAnchorExtra struct {
	IsEnabled *bool   `json:"isEnabled,omitempty"`
	NewHeadID *string `json:"newHeadId,omitempty"`
}

type ChangeLineEndAnchorExtra struct {
	IsEnabled *bool   `json:"isEnabled,omitempty"`
	NewHeadID *string `json:"newHeadId,omitempty"`
}

type ReorderShape struct {
	OrderType shape.OrderType `json:"orderType"`
}

type Copy struct{}
type Cut struct{}
type Duplicate struct{}
type CopyText struct{}
type Undo struct{}
type Redo struct{}

func (Idle) isOneTime()                              {}
func (SelectAllShapes) isOneTime()                   {}
func (DeselectShapes) isOneTime()                    {}
func (DeleteSelectedShapes) isOneTime()              {}
func (EditSelectedShapes) isOneTime()                {}
func (EditSelectedShape) isOneTime()                 {}
func (TextAlignment) isOneTime()                     {}
func (MoveShapes) isOneTime()                        {}
func (ChangeShapeBound) isOneTime()                  {}
func (ChangeShapeFillExtra) isOneTime()              {}
func (ChangeShapeBorderExtra) isOneTime()            {}
func (ChangeShapeBorderDashPatternExtra) isOneTime() {}
func (ChangeShapeBorderCornerExtra) isOneTime()      {}
func (ChangeLineStrokeExtra) isOneTime()             {}
func (ChangeLineStrokeDashPatternExtra) isOneTime()  {}
func (ChangeLineStrokeCornerExtra) isOneTime()       {}
func (ChangeLineStartAnchorExtra) isOneTime()        {}
func (ChangeLineEndAnchorExtra) isOneTime()          {}
func (ReorderShape) isOneTime()                      {}
func (Copy) isOneTime()                              {}
func (Cut) isOneTime()                               {}
func (Duplicate) isOneTime()                         {}
func (CopyText) isOneTime()                          {}
func (Undo) isOneTime()                              {}
func (Redo) isOneTime()                              {}

// oneTimeParsers are the actions JS can send, by their "type". See ADR-0005.
var oneTimeParsers = map[string]func(data []byte) (OneTime, error){
	"selectAllShapes":                   parse[SelectAllShapes],
	"deselectShapes":                    parse[DeselectShapes],
	"deleteSelectedShapes":              parse[DeleteSelectedShapes],
	"editSelectedShapes":                parse[EditSelectedShapes],
	"textAlignment":                     parse[TextAlignment],
	"moveShapes":                        parse[MoveShapes],
	"changeShapeBound":                  parse[ChangeShapeBound],
	"changeShapeFillExtra":              parse[ChangeShapeFillExtra],
	"changeShapeBorderExtra":            parse[ChangeShapeBorderExtra],
	"changeShapeBorderDashPatternExtra": parse[ChangeShapeBorderDashPatternExtra],
	"changeShapeBorderCornerExtra":      parse[ChangeShapeBorderCornerExtra],
	"changeLineStrokeExtra":             parse[ChangeLineStrokeExtra],
	"changeLineStrokeDashPatternExtra":  parse[ChangeLineStrokeDashPatternExtra],
	"changeLineStrokeCornerExtra":       parse[ChangeLineStrokeCornerExtra],
	"changeLineStartAnchorExtra":        parse[ChangeLineStartAnchorExtra],
	"changeLineEndAnchorExtra":          parse[ChangeLineEndAnchorExtra],
	"reorderShape":                      parse[ReorderShape],
	"copy":                              parse[Copy],
	"cut":                               parse[Cut],
	"duplicate":                         parse[Duplicate],
	"copyText":                          parse[CopyText],
	"undo":                              parse[Undo],
	"redo":                              parse[Redo],
}

func parse[T OneTime](data []byte) (OneTime, error) {
	var action T
	err := json.Unmarshal(data, &action)
	return action, err
}

// ParseOneTime reads an action sent from JS.
func ParseOneTime(data []byte) (OneTime, error) {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	parser, ok := oneTimeParsers[header.Type]
	if !ok {
		return nil, fmt.Errorf("unknown action type: %q", header.Type)
	}
	return parser(data)
}
