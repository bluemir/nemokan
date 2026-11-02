import {html} from 'lit-html';

// The widgets of wireframes, from Mockdown. Go draws them (shape.WidgetContent), and the form
// of each kind here edits the JSON of its content. See ADR-0014.
//
// Field types: text, list (one item per line), check, number, choice (an item of another list
// field, kept as its index).
export const widgets = [
	{kind: "button", label: "Button", icon: "[ ]", fields: [{name: "label", label: "Label", type: "text"}]},
	{kind: "input", label: "Input", icon: "[_]", fields: [{name: "placeholder", label: "Placeholder", type: "text"}]},
	{kind: "card", label: "Card", icon: "┌─┐", fields: [{name: "title", label: "Title", type: "text"}]},
	{kind: "table", label: "Table", icon: "┼─┼", fields: [{name: "columns", label: "Columns", type: "list"}]},
	{kind: "modal", label: "Modal", icon: "┌x┐", fields: [{name: "title", label: "Title", type: "text"}]},
	{kind: "checkbox", label: "Checkbox", icon: "[x]", fields: [
		{name: "label", label: "Label", type: "text"},
		{name: "isChecked", label: "Checked", type: "check"},
	]},
	{kind: "radio", label: "Radio", icon: "(o)", fields: [
		{name: "label", label: "Label", type: "text"},
		{name: "isSelected", label: "Selected", type: "check"},
	]},
	{kind: "dropdown", label: "Dropdown", icon: "[v]", fields: [{name: "label", label: "Label", type: "text"}]},
	{kind: "toggle", label: "Toggle", icon: "o==", fields: [
		{name: "label", label: "Label", type: "text"},
		{name: "isOn", label: "On", type: "check"},
	]},
	{kind: "tabs", label: "Tabs", icon: "[]─", fields: [
		{name: "tabs", label: "Tabs", type: "list"},
		{name: "active", label: "Active", type: "choice", of: "tabs"},
	]},
	{kind: "search", label: "Search", icon: "[/]", fields: [{name: "placeholder", label: "Placeholder", type: "text"}]},
	{kind: "progress", label: "Progress", icon: "##-", fields: [{name: "value", label: "Value (%)", type: "number", min: 0, max: 100}]},
	{kind: "breadcrumb", label: "Breadcrumb", icon: "a>b", fields: [{name: "items", label: "Items", type: "list"}]},
	{kind: "pagination", label: "Pagination", icon: "<1>", fields: [
		{name: "current", label: "Current", type: "number", min: 1},
		{name: "total", label: "Total", type: "number", min: 1},
	]},
	{kind: "nav", label: "Nav bar", icon: "──[", fields: [
		{name: "logo", label: "Logo", type: "text"},
		{name: "links", label: "Links", type: "list"},
		{name: "action", label: "Action", type: "text"},
	]},
	{kind: "list", label: "List", icon: "- -", fields: [{name: "items", label: "Items", type: "list"}]},
	{kind: "placeholder", label: "Placeholder", icon: "┌·┐", fields: [{name: "label", label: "Label", type: "text"}]},
	{kind: "split", label: "Split", icon: "┌┬┐", fields: [{name: "split", label: "Split (%)", type: "number", min: 0, max: 100}]},
	{kind: "image", label: "Image", icon: "┌/┐", fields: []},
];

// retainableOf is the retainable action adding widgets of kind.
export function retainableOf(kind) {
	return `widget:${kind}`;
}

export function findWidget(kind) {
	return widgets.find(widget => widget.kind === kind);
}

function fieldTemplate(field, content, change) {
	const value = content[field.name];
	switch (field.type) {
	case "check":
		return html`<label class="check"><input type="checkbox" .checked=${value} @change=${event => change(event.target.checked)}> ${field.label}</label>`;
	case "number":
		return html`<label>${field.label} <input type="number" min=${field.min ?? ""} max=${field.max ?? ""} .value=${String(value)} @change=${event => {
			if (event.target.value !== "") {
				change(event.target.valueAsNumber);
			}
		}}></label>`;
	case "list":
		return html`<label>${field.label} <textarea rows=${Math.max(value.length, 2)} .value=${value.join("\n")} @change=${event => change(event.target.value.split("\n"))}></textarea></label>`;
	case "choice":
		return html`<label>${field.label} <select @change=${event => change(Number(event.target.value))}>
			${content[field.of].map((item, index) => html`<option value=${index} ?selected=${index === value}>${item}</option>`)}
		</select></label>`;
	default:
		return html`<label>${field.label} <input .value=${value} @change=${event => change(event.target.value)}></label>`;
	}
}

// widgetToolTemplate is the widget section of the shape tool panel. send takes a one time
// action.
export function widgetToolTemplate(tool, send) {
	const widget = findWidget(tool.kind);
	if (!widget || widget.fields.length === 0) {
		return "";
	}
	return html`
		<fieldset class="widget">
			<legend>${widget.label}</legend>
			${widget.fields.map(field => fieldTemplate(field, tool.content, value => send({type: "changeWidgetContent", content: {...tool.content, [field.name]: value}})))}
		</fieldset>
	`;
}
