// Keyboard shortcuts of the editor, the same keys as the original KeyCommand. See ADR-0005.
// Keys are KeyboardEvent.code, so they work regardless of the keyboard layout or IME.
//
// command and shift: true (pressed), false (not pressed), undefined (either).
// preventDefault stops the browser default action, such as selecting the page.
export const shortcuts = [
	{codes: ["KeyA"], command: true, action: {type: "selectAllShapes"}, preventDefault: true, label: "Select all"},
	{codes: ["Escape"], action: {type: "deselectShapes"}, label: "Deselect"},
	{codes: ["Backspace", "Delete"], action: {type: "deleteSelectedShapes"}, label: "Delete"},

	{codes: ["ArrowLeft"], shift: false, action: {type: "moveShapes", offsetRow: 0, offsetColumn: -1}, label: "Move left"},
	{codes: ["ArrowUp"], shift: false, action: {type: "moveShapes", offsetRow: -1, offsetColumn: 0}, label: "Move up"},
	{codes: ["ArrowRight"], shift: false, action: {type: "moveShapes", offsetRow: 0, offsetColumn: 1}, label: "Move right"},
	{codes: ["ArrowDown"], shift: false, action: {type: "moveShapes", offsetRow: 1, offsetColumn: 0}, label: "Move down"},
	{codes: ["ArrowLeft"], shift: true, action: {type: "moveShapes", offsetRow: 0, offsetColumn: -5}, label: "Move left fast"},
	{codes: ["ArrowUp"], shift: true, action: {type: "moveShapes", offsetRow: -5, offsetColumn: 0}, label: "Move up fast"},
	{codes: ["ArrowRight"], shift: true, action: {type: "moveShapes", offsetRow: 0, offsetColumn: 5}, label: "Move right fast"},
	{codes: ["ArrowDown"], shift: true, action: {type: "moveShapes", offsetRow: 5, offsetColumn: 0}, label: "Move down fast"},

	{codes: ["KeyR"], retainable: "addRectangle", label: "Rectangle"},
	{codes: ["KeyT"], retainable: "addText", label: "Text"},
	{codes: ["KeyL"], retainable: "addLine", label: "Line"},
	{codes: ["KeyV"], command: false, retainable: "idle", label: "Select"},

	{codes: ["Enter"], action: {type: "editSelectedShapes"}, label: "Edit text"},

	{codes: ["KeyC"], command: true, shift: false, action: {type: "copy"}, label: "Copy"},
	{codes: ["KeyX"], command: true, action: {type: "cut"}, label: "Cut"},
	{codes: ["KeyD"], command: true, action: {type: "duplicate"}, preventDefault: true, label: "Duplicate"},
	{codes: ["KeyC"], command: true, shift: true, action: {type: "copyText"}, preventDefault: true, label: "Copy as text"},

	{codes: ["KeyG"], command: true, shift: false, action: {type: "groupShapes"}, preventDefault: true, label: "Group"},
	{codes: ["KeyG"], command: true, shift: true, action: {type: "ungroupShapes"}, preventDefault: true, label: "Ungroup"},

	{codes: ["KeyZ"], command: true, shift: false, action: {type: "undo"}, label: "Undo"},
	{codes: ["KeyZ"], command: true, shift: true, action: {type: "redo"}, label: "Redo"},
];

const isMac = /mac/i.test(navigator.userAgentData?.platform ?? navigator.platform);

// hasCommandKey is Cmd on macOS and Ctrl on the others.
export function hasCommandKey(event) {
	return isMac ? event.metaKey : event.ctrlKey;
}

const accepts = (state, isPressed) => state === undefined || state === isPressed;

// findShortcut returns the first shortcut matching the event, or undefined.
export function findShortcut(event) {
	const command = hasCommandKey(event);
	return shortcuts.find(shortcut =>
		shortcut.codes.includes(event.code) &&
		accepts(shortcut.command, command) &&
		accepts(shortcut.shift, event.shiftKey)
	);
}

// keyLabel returns how the keys of a shortcut are shown to the user, such as "⌘ Shift Z".
export function keyLabel(shortcut) {
	const keys = shortcut.codes.map(code => code.replace(/^Key/, "").replace(/^Arrow/, "Arrow ")).join(" / ");
	return [
		shortcut.command ? (isMac ? "⌘" : "Ctrl") : "",
		shortcut.shift ? "Shift" : "",
		keys,
	].filter(Boolean).join(" ");
}
