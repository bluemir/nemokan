// Package cursor defines the mouse cursors of the board, as CSS cursor values.
package cursor

type Cursor string

const (
	Default    Cursor = "default"
	Text       Cursor = "text"
	Crosshair  Cursor = "crosshair"
	Move       Cursor = "move"
	ResizeNWSE Cursor = "nwse-resize"
	ResizeNS   Cursor = "ns-resize"
	ResizeNESW Cursor = "nesw-resize"
	ResizeEW   Cursor = "ew-resize"
	ResizeRow  Cursor = "row-resize"
	ResizeCol  Cursor = "col-resize"
)
