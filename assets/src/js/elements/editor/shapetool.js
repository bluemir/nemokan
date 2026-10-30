import {html} from 'lit-html';

// The shape tool panel, the port of the original shapetool2. Go decides what to show
// (state.ShapeTool), and each change goes back to Go as a one time action.

const reorders = [
	{orderType: "front", label: "Bring to front", path: "M18,18h-9V15h-6V9h-3V0h9V3h6V9h3v9h0Zm-4-4V4H4V14h10Z"},
	{orderType: "forward", label: "Bring forward", path: "M18,18h-12v-5h-6v-13h13v6h5v12h0Zm-17-6h11v-11h-11Z"},
	{orderType: "backward", label: "Send backward", path: "M6,18V13h-6V0h13V6h5V18Zm-5-6h5V6h6V1H1Z"},
	{orderType: "back", label: "Send to back", path: "M9,18V15h-6V9h-3V0h9V3h6V9h3v9Zm-5-4h5V9h-5Zm10-5V4h-5V9Z"},
];

const horizontalAligns = [
	{align: "left", label: "Left", path: "M3 2h14v2H3zM3 6h8v2H3zM3 10h10v2H3z"},
	{align: "middle", label: "Middle", path: "M3 2h14v2H3zM6 6h8v2H6zM5 10h10v2H5z"},
	{align: "right", label: "Right", path: "M3 2h14v2H3zM9 6h8v2H9zM7 10h10v2H7z"},
];

const verticalAligns = [
	{align: "top", label: "Top", path: "M3 0h14v2H3zM3 4h14v2H3z"},
	{align: "middle", label: "Middle", path: "M3 4h14v2H3zM3 8h14v2H3z"},
	{align: "bottom", label: "Bottom", path: "M3 8h14v2H3zM3 12h14v2H3z"},
];

const roundedCornerPath = "M16 146V55c0-22 17-39 39-39h91M228 16h91c22 0 39 17 39 39v91M146 496H55c-22 0-39-17-39-39v-91M358 366v91c0 22-17 39-39 39h-91";

const nbsp = " ";

// numberValue returns the number in a number input, or undefined if it is empty.
function numberValue(input) {
	return input.value === "" ? undefined : input.valueAsNumber;
}

// styles shows the off option and the options of one style. selection is {isEnabled, selectedId}.
function styles(options, offLabel, selection, send) {
	return html`
		<menu class="styles">
			<li><button aria-pressed=${!selection.isEnabled} title="None" @click=${() => send({isEnabled: false})}>${offLabel}</button></li>
			${options.map(option => html`
				<li><button aria-pressed=${selection.isEnabled && selection.selectedId === option.id} @click=${() => send({isEnabled: true, id: option.id})}>${option.name}</button></li>
			`)}
		</menu>
	`;
}

function roundedCorner(isRoundedCorner, send) {
	if (isRoundedCorner === null) {
		return "";
	}
	return html`
		<button aria-pressed=${isRoundedCorner} title="Rounded corner" @click=${() => send(!isRoundedCorner)}>
			<svg width="19" height="26" viewBox="0 0 374 512" fill="none" stroke="currentColor" stroke-width="30"><path d=${roundedCornerPath}/></svg>
		</button>
	`;
}

function dashPattern(pattern, type, send) {
	const change = name => event => {
		const value = numberValue(event.target);
		if (value !== undefined) {
			send({type, [name]: value});
		}
	};
	return html`
		<p class="numbers">
			<label>Dash <input type="number" min="1" .value=${String(pattern.dash)} @change=${change("dash")}></label>
			<label>Gap <input type="number" min="0" .value=${String(pattern.gap)} @change=${change("gap")}></label>
			<label>Shift <input type="number" .value=${String(pattern.offset)} @change=${change("offset")}></label>
		</p>
	`;
}

function icon(path, width, height) {
	return html`<svg width=${width} height=${height} viewBox="0 0 ${width} ${height}" fill="currentColor"><path d=${path}/></svg>`;
}

function transformTemplate(tool, send) {
	const bound = tool.bound;
	const change = name => event => {
		const value = numberValue(event.target);
		if (value !== undefined) {
			send({type: "changeShapeBound", [name]: value});
		}
	};
	return html`
		<fieldset>
			<menu>
				${reorders.map(reorder => html`
					<li><button title=${reorder.label} @click=${() => send({type: "reorderShape", orderType: reorder.orderType})}>${icon(reorder.path, 18, 18)}</button></li>
				`)}
			</menu>
		</fieldset>
		<fieldset>
			<legend>Transform</legend>
			<p class="numbers">
				<label>X <input type="number" .value=${String(bound.left)} @change=${change("newLeft")}></label>
				<label>Y <input type="number" .value=${String(bound.top)} @change=${change("newTop")}></label>
				<label>W <input type="number" min="1" ?disabled=${!tool.isResizeable} .value=${String(bound.width)} @change=${change("newWidth")}></label>
				<label>H <input type="number" min="1" ?disabled=${!tool.isResizeable} .value=${String(bound.height)} @change=${change("newHeight")}></label>
			</p>
		</fieldset>
	`;
}

function rectangleTemplate(rectangle, options, send) {
	return html`
		<p>Fill</p>
		${styles(options.fill, "×", rectangle.fill, ({isEnabled, id}) => send({type: "changeShapeFillExtra", isEnabled, newFillStyleId: id}))}
		<p>Border</p>
		<div class="row">
			${styles(options.stroke, nbsp, rectangle.border, ({isEnabled, id}) => send({type: "changeShapeBorderExtra", isEnabled, newBorderStyleId: id}))}
			${roundedCorner(rectangle.isRoundedCorner, isRoundedCorner => send({type: "changeShapeBorderCornerExtra", isRoundedCorner}))}
		</div>
		${dashPattern(rectangle.dashPattern, "changeShapeBorderDashPatternExtra", send)}
	`;
}

function lineTemplate(line, options, send) {
	return html`
		<p>Stroke</p>
		<div class="row">
			${styles(options.stroke, nbsp, line.stroke, ({isEnabled, id}) => send({type: "changeLineStrokeExtra", isEnabled, newStrokeStyleId: id}))}
			${roundedCorner(line.isRoundedCorner, isRoundedCorner => send({type: "changeLineStrokeCornerExtra", isRoundedCorner}))}
		</div>
		${dashPattern(line.dashPattern, "changeLineStrokeDashPatternExtra", send)}
		<p>Start head</p>
		${styles(options.head, nbsp, line.startHead, ({isEnabled, id}) => send({type: "changeLineStartAnchorExtra", isEnabled, newHeadId: id}))}
		<p>End head</p>
		${styles(options.head, nbsp, line.endHead, ({isEnabled, id}) => send({type: "changeLineEndAnchorExtra", isEnabled, newHeadId: id}))}
	`;
}

function textAlignTemplate(textAlign, send) {
	return html`
		<fieldset>
			<legend>Text</legend>
			<p>Alignment</p>
			<menu class="styles">
				${horizontalAligns.map(align => html`
					<li><button title=${align.label} aria-pressed=${textAlign.horizontal === align.align} @click=${() => send({type: "textAlignment", newHorizontalAlign: align.align})}>${icon(align.path, 20, 14)}</button></li>
				`)}
			</menu>
			<p>Position</p>
			<menu class="styles">
				${verticalAligns.map(align => html`
					<li><button title=${align.label} aria-pressed=${textAlign.vertical === align.align} @click=${() => send({type: "textAlignment", newVerticalAlign: align.align})}>${icon(align.path, 20, 14)}</button></li>
				`)}
			</menu>
		</fieldset>
	`;
}

// shapeToolTemplate renders the panel. send takes a one time action.
export function shapeToolTemplate(tool, options, send) {
	const hasAppearance = tool.rectangle || tool.line;
	return html`
		<aside id="shape-tool">
			${tool.bound ? transformTemplate(tool, send) : ""}
			${hasAppearance ? html`
				<fieldset>
					<legend>Appearance</legend>
					${tool.rectangle ? rectangleTemplate(tool.rectangle, options, send) : ""}
					${tool.line ? lineTemplate(tool.line, options, send) : ""}
				</fieldset>
			` : ""}
			${tool.textAlign ? textAlignTemplate(tool.textAlign, send) : ""}
			${!tool.bound && !hasAppearance && !tool.textAlign ? html`<p>Select a shape for updating its properties here</p>` : ""}
		</aside>
	`;
}
