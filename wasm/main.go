//go:build js && wasm

// Package main is the browser entry point. It exposes the editor engine to JS
// as the global `monosketch` object. See ADR-0005, ADR-0006 and ADR-0007.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/bluemir/monosketch/internal/action"
	"github.com/bluemir/monosketch/internal/buildinfo"
	"github.com/bluemir/monosketch/internal/geo"
	"github.com/bluemir/monosketch/internal/interactionbound"
	"github.com/bluemir/monosketch/internal/shape"
	"github.com/bluemir/monosketch/internal/state"
)

// changeNames are the names JS receives in the change callback.
var changeNames = map[state.Change]string{
	state.ChangeBoard:             "board",
	state.ChangeInteractionBounds: "interactionBounds",
	state.ChangeSelectionBound:    "selectionBound",
	state.ChangeCursor:            "cursor",
	state.ChangeTextEditing:       "textEditing",
	state.ChangeClipboard:         "clipboard",
	state.ChangeBackup:            "backup",
	state.ChangeRetainableAction:  "retainableAction",
	state.ChangeShapeTool:         "shapeTool",
}

type engine struct {
	actions *action.Manager
	state   *state.Manager
}

func main() {
	monosketch := js.Global().Get("Object").New()
	monosketch.Set("version", js.FuncOf(version))
	monosketch.Set("start", js.FuncOf(start))
	js.Global().Set("monosketch", monosketch)

	// Keep the Go runtime alive so JS can keep calling the exported functions.
	select {}
}

func version(this js.Value, args []js.Value) any {
	return map[string]any{
		"app":       buildinfo.AppName,
		"version":   buildinfo.Version,
		"buildTime": buildinfo.BuildTime,
	}
}

// start creates an editor. args[0] is called with a change name whenever something JS
// should read changes.
func start(this js.Value, args []js.Value) any {
	onChange := args[0]
	actions := action.NewManager()
	editor := &engine{actions: actions}
	editor.state = state.New(shape.NewManager(), actions, func(change state.Change) {
		onChange.Invoke(changeNames[change])
	})

	functions := map[string]func(args []js.Value) any{
		"setDrawingInfo":    editor.setDrawingInfo,
		"mouseDown":         editor.mouseEvent(editor.state.OnMouseDown),
		"mouseUp":           editor.mouseEvent(editor.state.OnMouseUp),
		"mouseMove":         editor.mouseEvent(editor.state.OnMouseMove),
		"shiftKey":          editor.shiftKey,
		"action":            editor.action,
		"setRetainable":     editor.setRetainable,
		"setAccess":         editor.setAccess,
		"retainable":        editor.retainable,
		"board":             editor.board,
		"interactionBounds": editor.interactionBounds,
		"selectionBound":    editor.selectionBound,
		"cursor":            editor.cursor,
		"textEditing":       editor.textEditing,
		"shapeTool":         editor.shapeTool,
		"styleOptions":      editor.styleOptions,
		"updateEditingText": editor.updateEditingText,
		"finishTextEditing": editor.finishTextEditing,
		"paste":             editor.paste,
		"takeClipboardText": editor.takeClipboardText,
		"backup":            editor.backup,
		"restore":           editor.restore,
		"rootId":            editor.rootID,
		"newDocument":       editor.newDocument,
		"exportText":        editor.exportText,
		"file":              editor.file,
		"fileInfo":          editor.fileInfo,
		"openFile":          editor.openFile(editor.state.OpenFile),
		"fileAsCopy":        editor.fileAsCopy,
		"openFileAsCopy":    editor.openFile(editor.state.OpenFileAsCopy),
	}
	object := js.Global().Get("Object").New()
	for name, function := range functions {
		object.Set(name, js.FuncOf(func(this js.Value, args []js.Value) any { return function(args) }))
	}
	return object
}

// parse reads a JSON string from JS. It returns false and logs the error on failure.
func parse(value js.Value, target any) bool {
	if err := json.Unmarshal([]byte(value.String()), target); err != nil {
		js.Global().Get("console").Call("error", "monosketch: invalid JSON", err.Error())
		return false
	}
	return true
}

// stringify returns a JSON string for JS, which JS reads with JSON.parse.
func stringify(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		js.Global().Get("console").Call("error", "monosketch: marshal", err.Error())
		return js.Null()
	}
	return string(data)
}

func (e *engine) setDrawingInfo(args []js.Value) any {
	var drawingInfo state.DrawingInfo
	if parse(args[0], &drawingInfo) {
		e.state.SetDrawingInfo(drawingInfo)
	}
	return nil
}

// mouseEvent takes (offsetX, offsetY, isWithShiftKey, timeMillis).
func (e *engine) mouseEvent(handle func(state.MouseEvent)) func(args []js.Value) any {
	return func(args []js.Value) any {
		handle(state.MouseEvent{
			OffsetPx:       geo.Point{Left: args[0].Int(), Top: args[1].Int()},
			IsWithShiftKey: args[2].Bool(),
			TimeMillis:     args[3].Float(),
		})
		return nil
	}
}

func (e *engine) shiftKey(args []js.Value) any {
	e.state.OnShiftKey(args[0].Bool())
	return nil
}

// action takes a JSON string of a one time action, such as {"type":"undo"}.
func (e *engine) action(args []js.Value) any {
	oneTime, err := action.ParseOneTime([]byte(args[0].String()))
	if err != nil {
		js.Global().Get("console").Call("error", "monosketch: action", err.Error())
		return nil
	}
	e.actions.SetOneTime(oneTime)
	return nil
}

func (e *engine) setRetainable(args []js.Value) any {
	var retainable action.Retainable
	if err := retainable.UnmarshalText([]byte(args[0].String())); err != nil {
		js.Global().Get("console").Call("error", "monosketch: retainable action", err.Error())
		return nil
	}
	e.actions.SetRetainable(retainable)
	return nil
}

// setAccess takes "edit" or "view". See ADR-0012.
func (e *engine) setAccess(args []js.Value) any {
	var access state.Access
	if err := access.UnmarshalText([]byte(args[0].String())); err != nil {
		js.Global().Get("console").Call("error", "monosketch: access", err.Error())
		return nil
	}
	e.state.SetAccess(access)
	return nil
}

func (e *engine) retainable(args []js.Value) any {
	text, _ := e.actions.Retainable.Value().MarshalText()
	return string(text)
}

// board returns {left, top, width, height, rows, highlights}. rows are strings and
// highlights is a Uint8Array of board.Highlight for each cell, row by row.
func (e *engine) board(args []js.Value) any {
	view := e.state.Board()
	rows := make([]any, len(view.Rows))
	for i, row := range view.Rows {
		rows[i] = row
	}
	highlights := js.Global().Get("Uint8Array").New(len(view.Highlights))
	js.CopyBytesToJS(highlights, view.Highlights)
	return map[string]any{
		"left":       view.Bound.Left(),
		"top":        view.Bound.Top(),
		"width":      view.Bound.Width(),
		"height":     view.Bound.Height(),
		"rows":       rows,
		"highlights": highlights,
	}
}

type pointJSON struct {
	Left float64 `json:"left"`
	Top  float64 `json:"top"`
}

type boundJSON struct {
	Type   string      `json:"type"`
	Left   float64     `json:"left,omitempty"`
	Top    float64     `json:"top,omitempty"`
	Right  float64     `json:"right,omitempty"`
	Bottom float64     `json:"bottom,omitempty"`
	Points []pointJSON `json:"points"`
}

// interactionBounds returns a JSON string of the bounds, in board units.
func (e *engine) interactionBounds(args []js.Value) any {
	bounds := []boundJSON{}
	for _, bound := range e.state.InteractionBounds() {
		var points []pointJSON
		for _, point := range bound.Points() {
			points = append(points, pointJSON{Left: point.Position().Left, Top: point.Position().Top})
		}
		switch bound := bound.(type) {
		case *interactionbound.ScalableBound:
			bounds = append(bounds, boundJSON{Type: "scalable", Left: bound.Left, Top: bound.Top, Right: bound.Right, Bottom: bound.Bottom, Points: points})
		case *interactionbound.LineBound:
			bounds = append(bounds, boundJSON{Type: "line", Points: points})
		}
	}
	return stringify(bounds)
}

// selectionBound returns a JSON string of the selecting area, or null.
func (e *engine) selectionBound(args []js.Value) any {
	bound, ok := e.state.SelectionBound()
	if !ok {
		return js.Null()
	}
	return stringify(bound)
}

func (e *engine) cursor(args []js.Value) any {
	return string(e.state.MouseCursor())
}

// textEditing returns a JSON string of {text, boundPx}, or null.
func (e *engine) textEditing(args []js.Value) any {
	editing := e.state.TextEditing()
	if editing == nil {
		return js.Null()
	}
	return stringify(map[string]any{"text": editing.Text, "boundPx": editing.BoundPx})
}

// shapeTool returns a JSON string of state.ShapeTool.
func (e *engine) shapeTool(args []js.Value) any {
	return stringify(e.state.ShapeTool())
}

type styleOptionJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// styleOptions returns a JSON string of the styles the shape tool offers, by kind.
func (e *engine) styleOptions(args []js.Value) any {
	var fills, strokes, heads []styleOptionJSON
	for _, style := range shape.PredefinedRectangleFillStyles() {
		fills = append(fills, styleOptionJSON{ID: style.ID, Name: style.DisplayName})
	}
	for _, style := range shape.PredefinedStrokeStyles() {
		strokes = append(strokes, styleOptionJSON{ID: style.ID, Name: style.DisplayName})
	}
	for _, anchor := range shape.PredefinedAnchorChars() {
		heads = append(heads, styleOptionJSON{ID: anchor.ID, Name: anchor.DisplayName})
	}
	return stringify(map[string]any{"fill": fills, "stroke": strokes, "head": heads})
}

func (e *engine) updateEditingText(args []js.Value) any {
	e.state.UpdateEditingText(args[0].String())
	return nil
}

func (e *engine) finishTextEditing(args []js.Value) any {
	e.state.FinishTextEditing()
	return nil
}

func (e *engine) paste(args []js.Value) any {
	e.state.Paste(args[0].String())
	return nil
}

func (e *engine) takeClipboardText(args []js.Value) any {
	text, ok := e.state.TakeClipboardText()
	if !ok {
		return js.Null()
	}
	return text
}

type backupJSON struct {
	RootID     string                            `json:"rootId"`
	Root       *shape.SerializableGroup          `json:"root"`
	Connectors []shape.SerializableLineConnector `json:"connectors"`
}

// backup returns a JSON string to keep in the local storage, or null.
func (e *engine) backup(args []js.Value) any {
	backup, ok := e.state.Backup()
	if !ok {
		return js.Null()
	}
	return stringify(backupJSON{RootID: backup.Root.OriginalID(), Root: backup.Root, Connectors: backup.Connectors})
}

// restore takes (rootId, backup JSON string or null).
func (e *engine) restore(args []js.Value) any {
	rootID := args[0].String()
	if args[1].IsNull() || args[1].IsUndefined() {
		e.state.Restore(rootID, nil)
		return nil
	}
	var backup backupJSON
	if !parse(args[1], &backup) || backup.Root == nil {
		e.state.Restore(rootID, nil)
		return nil
	}
	e.state.Restore(rootID, &state.Backup{Root: backup.Root, Connectors: backup.Connectors})
	return nil
}

func (e *engine) rootID(args []js.Value) any {
	return e.state.ShapeManager().Root().ID()
}

func (e *engine) newDocument(args []js.Value) any {
	e.state.Restore("", nil)
	return nil
}

func (e *engine) exportText(args []js.Value) any {
	return e.state.ExportText()
}

// file takes (name, offsetLeft, offsetTop) and returns the JSON string of the document file.
func (e *engine) file(args []js.Value) any {
	return stringify(e.state.File(args[0].String(), geo.Point{Left: args[1].Int(), Top: args[2].Int()}))
}

// fileInfo takes the JSON string of a file and returns {rootId, name, offset}, or null if
// the file cannot be read.
func (e *engine) fileInfo(args []js.Value) any {
	file, err := shape.ParseFile([]byte(args[0].String()))
	if err != nil {
		js.Global().Get("console").Call("error", "monosketch: file", err.Error())
		return js.Null()
	}
	return map[string]any{
		"rootId": file.Root.OriginalID(),
		"name":   file.Name,
		"offset": map[string]any{"left": file.Offset.Left, "top": file.Offset.Top},
	}
}

// fileAsCopy takes the JSON string of a file and returns it as a new document with a new
// root id, without opening it. It returns null if the file cannot be read.
func (e *engine) fileAsCopy(args []js.Value) any {
	file, err := shape.ParseFile([]byte(args[0].String()))
	if err != nil {
		js.Global().Get("console").Call("error", "monosketch: file", err.Error())
		return js.Null()
	}
	return stringify(file.AsCopy())
}

// openFile takes the JSON string of a file. JS checks it with fileInfo first.
func (e *engine) openFile(open func(shape.File)) func(args []js.Value) any {
	return func(args []js.Value) any {
		file, err := shape.ParseFile([]byte(args[0].String()))
		if err != nil {
			js.Global().Get("console").Call("error", "monosketch: file", err.Error())
			return nil
		}
		open(file)
		return nil
	}
}
