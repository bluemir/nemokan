import {html, render} from 'lit-html';
import {findShortcut, hasCommandKey, keyLabel, shortcuts} from "./shortcuts.js";
import {defaultName, findDocumentByServerId, getDocument, hasUnsyncedChanges, listDocuments, removeDocument, saveBackup, updateDocument} from "./storage.js";
import {createServerDocument, getServerDocument, isLoggedIn, setServerDocumentVisibility, updateServerDocument} from "./server.js";
import {download, openFile, saveFile} from "./files.js";
import {markdown, png, svg} from "./export.js";
import {shapeToolTemplate} from "./shapetool.js";
import {layersTemplate} from "./layers.js";
import {findWidget, retainableOf, widgets} from "./widgets.js";

// The Go engine does the editing. This element forwards events to it and draws what it
// reads on the next frame. Documents, files and settings are handled here. See ADR-0001,
// ADR-0006 and ADR-0010.

// D2Coding draws a wide char exactly two cells wide. See ADR-0008 and ADR-0009.
const fontFamily = `D2Coding, ui-monospace, Menlo, Consolas, monospace`;
// Font sizes are the same as the original.
const minFontSize = 13;
const maxFontSize = 25;
const fontSizeKey = "nemokan.fontSize";
const shapeToolHiddenKey = "nemokan.shapeToolHidden";
const backupDelayMillis = 300;
// exportStatusMillis is how long the result of an export is shown.
const exportStatusMillis = 2000;
// syncDelayMillis is how long after the last change a server document is saved. See ADR-0011.
const syncDelayMillis = 2000;
const syncRetryMillis = 10000;
const visibilities = [
	{visibility: "private", label: "Only me"},
	{visibility: "view", label: "Anyone with the link can view"},
	{visibility: "edit", label: "Anyone with the link can edit"},
];
const scrollSpeedRatio = 1 / 1.3;
// interactionDotRadiusPx is the radius of the dots the user drags to scale or move shapes.
const interactionDotRadiusPx = 3.2;

// The axes are drawn on the top and left edges of the canvas, and the board is drawn inside
// them, as the original.
const axisWidthPx = 33;
const axisHeightPx = 18;
const axisRulerSizePx = 12;
const axisFontSizePx = 10.5;
// axisColumnStep is how often the top axis shows a column number.
const axisColumnStep = 20;

// wideCharTail is the second cell of a wide char in the board rows. See ADR-0008.
const wideCharTail = "\0";

// highlightColors are the CSS variables of board.Highlight values.
const highlightColors = ["--shape", "--shape-selected", "--shape-text-editing", "--shape-line-connect-target"];

// The icons of the mouse actions are from the original MouseActionToolbar, in a 24 x 24 box.
const retainableTools = [
	{retainable: "idle", label: "Select", key: "V", path: "M7.436 20.61L7.275 3.914l12.296 11.29-7.165.235-4.97 5.168z"},
	{retainable: "addRectangle", label: "Rectangle", key: "R", path: "M22 19H2V5h20v14zM4 7v10h16V7z"},
	{retainable: "addText", label: "Text", key: "T", path: "M5.635 21v-2h12.731v2zm3.27-4v-1.12h2.005V4.12H7.425l-.39.44v2.58h-1.4V3h12.731v4.14h-1.4V4.56l-.39-.44h-3.485v11.76h2.005V17z"},
	{retainable: "addLine", label: "Line", key: "L", path: "M18 15v-2H6v2H0V9h6v2h12V9h6v6z"},
];

// The original has no undo and redo buttons, so their icons are drawn with strokes here.
const historyTools = [
	{type: "undo", label: "Undo", path: "M9 14L4 9l5-5M4 9h11a5 5 0 0 1 0 10h-3"},
	{type: "redo", label: "Redo", path: "M15 14l5-5-5-5M20 9H9a5 5 0 0 0 0 10h3"},
];

// scrollModes limit the wheel scroll to a direction, in the order the button cycles. The
// icons are from the original RightToolbar, in a 40 x 40 box.
const scrollModeArrows = {
	vertical: "M19.01.582a1.134 1.134 0 0 1 1.98 0l4.87 8.834c.41.744-.133 1.651-.99 1.651h-9.74c-.857 0-1.4-.907-.99-1.651L19.01.582ZM20.99 39.418a1.134 1.134 0 0 1-1.98 0l-4.87-8.834c-.41-.744.133-1.651.99-1.651h9.74c.857 0 1.4.907.99 1.651l-4.87 8.834Z",
	horizontal: "M.582 20.99a1.134 1.134 0 0 1 0-1.98l8.834-4.87c.744-.41 1.651.133 1.651.99v9.74c0 .857-.907 1.4-1.651.99L.582 20.99ZM39.418 19.01a1.134 1.134 0 0 1 0 1.98l-8.834 4.87c-.744.41-1.651-.133-1.651-.99v-9.74c0-.857.907-1.4 1.651-.99l8.834 4.87Z",
};
const scrollModeCenter = "M25 20a5 5 0 1 1-10 0 5 5 0 0 1 10 0Z";
const scrollModes = [
	{mode: "both", label: "Scroll in both directions", path: scrollModeCenter + scrollModeArrows.vertical + scrollModeArrows.horizontal},
	{mode: "vertical", label: "Scroll vertically", path: scrollModeCenter + scrollModeArrows.vertical},
	{mode: "horizontal", label: "Scroll horizontally", path: scrollModeCenter + scrollModeArrows.horizontal},
];

// isTyping reports whether a key or paste event goes to a text field or a dialog, such as
// the text editor. Then it is not for the board. composedPath looks into the shadow root.
function isTyping(event) {
	const target = event.composedPath()[0];
	return target instanceof Element && target.closest("input, textarea, select, [contenteditable], dialog[open]") !== null;
}

// serverErrorMessage tells the user why a request to the server failed.
function serverErrorMessage(status) {
	switch (status) {
	case 0:
		return "The server cannot be reached. The document is kept in this browser.";
	case 404:
		return "The document is not on the server, or you cannot see it. It may have expired.";
	case 413:
		return "The document is too large to save on the server.";
	case 429:
		return "Too many documents were made from your network. Try again later.";
	default:
		return `The server could not save the document (${status}).`;
	}
}

function readFontSize() {
	try {
		const fontSize = Number(localStorage.getItem(fontSizeKey));
		return fontSize >= minFontSize && fontSize <= maxFontSize ? fontSize : minFontSize;
	} catch {
		return minFontSize;
	}
}

const themeKey = "nemokan.theme";
const themes = ["system", "light", "dark"];
const darkSchemeQuery = matchMedia("(prefers-color-scheme: dark)");

// readTheme returns "system", "light" or "dark".
function readTheme() {
	try {
		const theme = localStorage.getItem(themeKey);
		return themes.includes(theme) ? theme : "system";
	} catch {
		return "system";
	}
}

function readShapeToolHidden() {
	try {
		return localStorage.getItem(shapeToolHiddenKey) === "true";
	} catch {
		return false;
	}
}

class Editor extends HTMLElement {
	#engine;
	#changes = new Set();
	#isFrameRequested = false;
	#backupTimer;
	#offsetTimer;

	#fontSize = readFontSize();
	#cellSizePx = {width: 1, height: 1};
	// charOffsetTop puts the top of a glyph at the top of its cell.
	#charOffsetTop = 0;
	#offsetPx = {left: 0, top: 0};
	#canvasSizePx = {width: 1, height: 1};
	#boardSizePx = {width: 1, height: 1};
	// engineCursor is the mouse cursor Go asks for. The axes corner shows its own.
	#engineCursor = "default";
	#wheelDelta = {left: 0, top: 0};
	// scrollMode is kept only for this page, as the original.
	#scrollMode = scrollModes[0];

	// pointerCell is the board cell under the mouse, shown in the status bar.
	#pointerCell = null;

	#interactionBounds = [];
	#selectionBound = null;
	#textEditing = null;
	#shapeTool = {};
	#layers = {items: [], workingGroupId: ""};
	// renamingLayerId is the layer being renamed in place.
	#renamingLayerId = null;
	#styleOptions = {fill: [], stroke: [], head: []};
	#isShapeToolHidden = readShapeToolHidden();
	#theme = readTheme();

	#documentName = defaultName;
	#documents = [];
	#exportedText = "";
	// exportStatus tells how the last export went, next to the export buttons.
	#exportStatus = "";
	#exportStatusTimer;
	// openingFile is the file waiting for the user to choose how to open it.
	#openingFile = null;
	#removingDocument = null;
	#errorMessage = "";

	// serverId is the server document of this page (/p/<id>), or null on /edit. See ADR-0011.
	#serverId = null;
	// server is the last response for the server document: {canEdit, isMine, visibility, ...}.
	#server = null;
	#syncStatus = "";
	#syncTimer;
	#isSyncing = false;
	#isSyncRequested = false;
	#isLoggedIn = false;
	// conflictingDocument is the newer server document while the user chooses what to keep.
	#conflictingDocument = null;

	constructor() {
		super();
		this.attachShadow({mode: 'open'});
	}

	get #font() {
		return `${this.#fontSize}px ${fontFamily}`;
	}

	template() {
		const retainable = this.#engine?.retainable();
		return html`
			<style>
				:host {
					--background: #FFFFFF;
					--shape: #000000;
					--shape-selected: #0D7CFF;
					--shape-text-editing: #0767C6;
					--shape-line-connect-target: #E86A33;
					--selection-area-stroke: #858585;
					--selection-bound-stroke: #64b5f6;
					--selection-dot-stroke: #64b5f6;
					--selection-dot-fill: #FFFFFF;
					--axis-background: #EEEEEE;
					--axis-text: #666666;
					--axis-rule: #444444;
					--grid-line: #d9d9d9;
					--grid-line-zero: #BBBBBB;

					/* The look of the page around the canvas follows Mockdown. */
					--border: #E5E5E5;
					--muted: #737373;
					--hover: rgb(0 0 0 / 5%);
					--accent: #2563EB;
					--on-accent: #FFFFFF;

					color-scheme: light;
					display: grid;
					grid-template-columns: 12rem minmax(0, 1fr) auto;
					grid-template-rows: minmax(0, 1fr) auto;
					grid-template-areas: "tools board panel" "tools status panel";
					overflow: hidden;
					background: var(--background);
					color: var(--shape);
					font: 0.75rem ${fontFamily};
				}
				:host([theme="dark"]) {
					color-scheme: dark;
					--border: rgb(255 255 255 / 10%);
					--muted: #A3A3A3;
					--hover: rgb(255 255 255 / 6%);
					--accent: #3B82F6;
					--background: #121212;
					--shape: #F0F0F0;
					--shape-selected: #FFD82F;
					--shape-text-editing: #956d04;
					--shape-line-connect-target: #00FFF6;
					--selection-bound-stroke: #FDF7C3;
					--selection-dot-stroke: #f2ae00;
					--selection-dot-fill: #1E1E1E;
					--axis-background: #1E1E1E;
					--axis-rule: #666666;
					--grid-line: #282828;
					--grid-line-zero: #323232;
				}
				button, input, select, kbd {
					font: inherit;
					color: inherit;
				}
				button {
					padding: 0.25rem 0.5rem;
					border: 1px solid transparent;
					background: none;
					cursor: pointer;
				}
				button:hover:not(:disabled) {
					background: var(--hover);
				}
				button:disabled {
					opacity: 0.3;
					cursor: default;
				}
				button[aria-pressed="true"], button.primary, button:is([aria-pressed="true"], .primary):hover {
					background: var(--accent);
					color: var(--on-accent);
				}
				input, select {
					border: 1px solid var(--border);
					background: var(--background);
				}
				dialog, [popover] {
					border: 1px solid var(--border);
					background: var(--background);
					color: var(--shape);
				}
				h2, legend {
					margin: 0;
					padding: 0;
					color: var(--muted);
					font-size: 0.625rem;
					font-weight: bold;
					letter-spacing: 0.08em;
					text-transform: uppercase;
				}
				menu {
					display: flex;
					gap: 0.25rem;
					margin: 0;
					padding: 0;
					list-style: none;
				}
				main {
					grid-area: board;
					position: relative;
				}
				canvas {
					display: block;
					width: 100%;
					height: 100%;
				}
				#tools {
					grid-area: tools;
					display: flex;
					flex-direction: column;
					gap: 1rem;
					padding: 0.5rem;
					border-right: 1px solid var(--border);
					overflow-y: auto;
				}
				#tools menu, [popover] menu {
					flex-direction: column;
					gap: 0.125rem;
				}
				#tools h2 {
					padding: 0 0.5rem 0.25rem;
				}
				#tools menu button, [popover] menu button {
					display: flex;
					align-items: center;
					gap: 0.5rem;
					width: 100%;
					text-align: left;
				}
				#tools kbd {
					margin-left: auto;
					opacity: 0.5;
				}
				#tools .controls {
					flex-direction: row;
					margin-top: auto;
				}
				#tools .export {
					display: flex;
					flex-direction: column;
					gap: 0.25rem;
				}
				#tools .export menu {
					flex-direction: row;
				}
				#tools .export menu li {
					flex: 1;
				}
				#tools .export menu button {
					justify-content: center;
					border-color: var(--border);
				}
				#tools .export output {
					min-height: 1lh;
					color: var(--muted);
					text-align: center;
				}
				#tools .controls button {
					width: auto;
				}
				[popovertarget="document-menu"] {
					display: flex;
					width: 100%;
					font-weight: bold;
					text-align: left;
					anchor-name: --document-menu;
				}
				[popovertarget="document-menu"] span {
					flex: 1;
					overflow: hidden;
					text-overflow: ellipsis;
				}
				#document-menu {
					position-anchor: --document-menu;
					inset: auto;
					top: anchor(bottom);
					left: anchor(left);
					min-width: 16rem;
					margin: 0.25rem 0 0;
					padding: 0.25rem;
				}
				#document-menu li {
					display: flex;
					align-items: center;
					gap: 0.25rem;
				}
				#document-menu li > :is(label, span, a) {
					padding: 0.25rem 0.5rem;
				}
				#document-menu li > button:not(:first-child) {
					width: auto;
				}
				#document-menu hr {
					border: none;
					border-top: 1px solid var(--border);
				}
				#status {
					grid-area: status;
					display: flex;
					align-items: center;
					gap: 1rem;
					padding: 0.25rem 1rem;
					border-top: 1px solid var(--border);
					color: var(--muted);
				}
				#status .tool {
					padding: 0.125rem 0.5rem;
					background: var(--accent);
					color: var(--on-accent);
					font-size: 0.625rem;
					font-weight: bold;
					letter-spacing: 0.08em;
					text-transform: uppercase;
				}
				#status output {
					margin-left: auto;
				}
				#panel {
					grid-area: panel;
					display: flex;
					flex-direction: column;
					width: 15rem;
					border-left: 1px solid var(--border);
				}
				#panel h2 {
					padding: 0.5rem 0.75rem;
					border-bottom: 1px solid var(--border);
				}
				#layers {
					flex: none;
					display: flex;
					flex-direction: column;
					max-height: 40%;
					border-bottom: 1px solid var(--border);
				}
				#layers ul {
					margin: 0;
					padding: 0;
					list-style: none;
				}
				#layers > ul {
					min-height: 0;
					overflow-y: auto;
				}
				#layers > menu {
					padding: 0.25rem 0.5rem;
					border-top: 1px solid var(--border);
				}
				#layers > menu li {
					flex: 1;
				}
				#layers > menu button {
					width: 100%;
				}
				#layers > p {
					margin: 0.75rem;
					color: var(--muted);
				}
				.layer {
					display: flex;
					align-items: center;
					gap: 0.375rem;
					padding: 0.125rem 0.25rem 0.125rem 0.5rem;
					border: 1px solid transparent;
					cursor: default;
				}
				.layer:hover {
					background: var(--hover);
				}
				.layer[aria-selected="true"] {
					background: color-mix(in srgb, var(--accent) 15%, transparent);
				}
				.layer[data-working] {
					box-shadow: inset 2px 0 var(--accent);
				}
				.layer[data-hidden] > :is(svg, span) {
					opacity: 0.4;
				}
				.layer span, .layer input {
					flex: 1;
					min-width: 0;
					overflow: hidden;
					text-overflow: ellipsis;
					white-space: nowrap;
				}
				.layer .default-name {
					color: var(--muted);
				}
				.layer button {
					padding: 0.125rem;
					color: var(--muted);
				}
				.layer button[aria-pressed="true"] {
					background: none;
					color: var(--shape);
				}
				.layer.drop-above {
					border-top-color: var(--accent);
				}
				.layer.drop-below {
					border-bottom-color: var(--accent);
				}
				.layer.drop-into {
					border-color: var(--accent);
				}
				#shape-tool {
					flex: 1;
					overflow-y: auto;
				}
				.widget-icon {
					flex: none;
					width: 3ch;
					font-style: normal;
					white-space: pre;
				}
				fieldset.widget label {
					display: grid;
					gap: 0.125rem;
					margin-top: 0.5rem;
				}
				fieldset.widget label.check {
					display: flex;
					align-items: center;
					gap: 0.5rem;
				}
				fieldset.widget textarea {
					position: static;
					outline: none;
					border: 1px solid var(--border);
					font: inherit;
					line-height: inherit;
					field-sizing: fixed;
				}
				fieldset {
					margin: 0;
					padding: 0.75rem;
					border: none;
					border-bottom: 1px solid var(--border);
				}
				#shape-tool p {
					margin: 0.5rem 0 0.25rem;
				}
				#shape-tool > p {
					margin: 1rem;
				}
				.styles {
					flex-wrap: wrap;
					font: 1.125rem ${fontFamily};
				}
				.styles button {
					min-width: 2em;
					border-color: var(--border);
				}
				.numbers {
					display: grid;
					grid-template-columns: 1fr 1fr;
					gap: 0.25rem 0.75rem;
				}
				.numbers input {
					width: 4em;
				}
				.row {
					display: flex;
					align-items: center;
					gap: 0.75rem;
				}
				button svg {
					display: block;
					flex: none;
				}
				textarea {
					position: absolute;
					margin: 0;
					padding: 0;
					border: none;
					outline: 1px solid var(--shape-text-editing);
					resize: none;
					overflow: hidden;
					/* It grows with the lines, as the text shape does. */
					field-sizing: content;
					background: var(--background);
					color: var(--shape);
					font: ${this.#font};
					line-height: ${this.#cellSizePx.height}px;
				}
				pre {
					font: ${this.#font};
					line-height: ${this.#cellSizePx.height}px;
					max-height: 60vh;
					overflow: auto;
				}
			</style>
			<aside id="tools">
				<button popovertarget="document-menu" title="Document"><span>${this.#documentName}</span>▾</button>
				<section>
					<h2>Basics</h2>
					<menu>
						${this.#retainableTools.map(tool => html`
							<li><button aria-pressed=${tool.retainable === retainable} @click=${() => this.#engine.setRetainable(tool.retainable)}>
								<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d=${tool.path}/></svg>
								${tool.label}<kbd>${tool.key}</kbd>
							</button></li>
						`)}
					</menu>
				</section>
				${this.#isViewOnly ? "" : html`
					<section>
						<h2>UI Elements</h2>
						<menu>
							${widgets.map(widget => html`
								<li><button aria-pressed=${retainableOf(widget.kind) === retainable} @click=${() => this.#engine.setRetainable(retainableOf(widget.kind))}>
									<i class="widget-icon">${widget.icon}</i>${widget.label}
								</button></li>
							`)}
						</menu>
					</section>
				`}
				<menu class="controls">
					${this.#isViewOnly ? "" : historyTools.map(tool => html`
						<li><button aria-label=${tool.label} title=${tool.label} @click=${() => this.#action({type: tool.type})}>
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d=${tool.path}/></svg>
						</button></li>
					`)}
					<li><button aria-label=${this.#scrollMode.label} title="${this.#scrollMode.label} (Shift swaps a direction)" @click=${() => this.#nextScrollMode()}>
						<svg width="16" height="16" viewBox="0 0 40 40" fill="currentColor"><path d=${this.#scrollMode.path}/></svg>
					</button></li>
				</menu>
				<section class="export" title="The selected shapes, or all shapes if none is selected">
					<button class="primary" @click=${() => this.#copyMarkdown()}>Copy Markdown</button>
					<menu>
						<li><button @click=${() => this.#downloadSvg()}>SVG</button></li>
						<li><button @click=${() => this.#downloadPng()}>PNG</button></li>
					</menu>
					<output>${this.#exportStatus}</output>
				</section>
			</aside>
			<main>
				<canvas
					@pointerdown=${event => this.#mouseDown(event)}
					@pointerup=${event => this.#mouse("mouseUp", event)}
					@pointermove=${event => this.#mouseMove(event)}
					@pointerleave=${() => this.#setPointerCell(null)}
					@wheel=${event => this.#wheel(event)}
				></canvas>
				${this.#textEditing ? html`
					<textarea
						style="left: ${axisWidthPx + this.#textEditing.boundPx.left}px; top: ${axisHeightPx + this.#textEditing.boundPx.top}px; width: ${this.#textEditing.boundPx.width}px; min-height: ${this.#textEditing.boundPx.height}px;"
						.value=${this.#textEditing.text}
						@input=${event => this.#engine.updateEditingText(event.target.value)}
						@keydown=${event => this.#textKeyDown(event)}
						@blur=${() => this.#engine.finishTextEditing()}
					></textarea>
				` : ""}
			</main>
			<footer id="status">
				<span class="tool">${this.#retainableLabel(retainable)}</span>
				<span>${this.#pointerCell ? `Row ${this.#pointerCell.row}, Col ${this.#pointerCell.column}` : ""}</span>
				${this.#serverId ? html`<output title="Server">${this.#isViewOnly ? "Read only. Make a copy to edit." : this.#syncStatus}</output>` : ""}
			</footer>
			${this.#isShapeToolHidden || this.#isViewOnly ? "" : html`
				<aside id="panel">
					${layersTemplate(this.#layers, {
						send: action => this.#action(action),
						renamingId: this.#renamingLayerId,
						startRename: id => this.#startRenameLayer(id),
						finishRename: action => this.#finishRenameLayer(action),
						goTo: layer => this.#goToLayer(layer),
					})}
					${shapeToolTemplate(this.#shapeTool, this.#styleOptions, action => this.#action(action))}
				</aside>
			`}
			${this.#documentMenuTemplate()}
			${this.#dialogsTemplate()}
		`;
	}

	#retainableLabel(retainable) {
		const widgetKind = retainable?.match(/^widget:(.+)$/)?.[1];
		if (widgetKind) {
			return findWidget(widgetKind)?.label;
		}
		return retainableTools.find(tool => tool.retainable === retainable)?.label;
	}

	// retainableTools are the mouse actions the user can choose. Only selecting is left for a
	// view only document.
	get #retainableTools() {
		return retainableTools.filter(tool => !this.#isViewOnly || tool.retainable === "idle");
	}

	#documentMenuTemplate() {
		const rootId = this.#engine?.rootId();
		return html`
			<div id="document-menu" popover @toggle=${event => event.newState === "open" && this.#refreshDocuments()}>
				<menu>
					<li><button @click=${() => this.#newDocument()}>New document</button></li>
					${this.#serverId ? "" : html`<li><button @click=${() => this.#openFile()}>Open file…</button></li>`}
					<li><button @click=${() => this.#saveFile()}>Save as file…</button></li>
					<li><button @click=${() => this.#showExport()}>Export as text…</button></li>
					${this.#isViewOnly ? "" : html`<li><button @click=${() => this.#showDialog("rename-dialog")}>Rename…</button></li>`}
					${this.#serverMenuTemplate()}
					<li><button @click=${() => this.#showDialog("shortcuts-dialog")}>Keyboard shortcuts</button></li>
					<li>
						<label>Theme
							<select @change=${event => this.#setTheme(event.target.value)}>
								${themes.map(theme => html`<option value=${theme} ?selected=${theme === this.#theme}>${theme[0].toUpperCase() + theme.slice(1)}</option>`)}
							</select>
						</label>
					</li>
					<li><label><input type="checkbox" .checked=${!this.#isShapeToolHidden} @change=${event => this.#setShapeToolHidden(!event.target.checked)}> Layers and properties</label></li>
					<li>
						<span>Font size</span>
						<button title="Smaller" ?disabled=${this.#fontSize <= minFontSize} @click=${() => this.#setFontSize(this.#fontSize - 1)}>−</button>
						${this.#fontSize}
						<button title="Larger" ?disabled=${this.#fontSize >= maxFontSize} @click=${() => this.#setFontSize(this.#fontSize + 1)}>+</button>
					</li>
				</menu>
				<hr>
				<menu>
					${this.#documents.map(localDocument => html`
						<li>
							<button aria-pressed=${localDocument.rootId === rootId} title=${localDocument.serverId ? "On the server" : "In this browser"} @click=${() => this.#openDocument(localDocument)}>${localDocument.serverId ? "☁ " : ""}${localDocument.name}</button>
							<button title="Remove" @click=${() => this.#confirmRemove(localDocument)}>×</button>
						</li>
					`)}
				</menu>
			</div>
		`;
	}

	#serverMenuTemplate() {
		if (!this.#serverId) {
			const sharedId = this.#documents.find(localDocument => localDocument.rootId === this.#engine?.rootId())?.serverId;
			return sharedId
				? html`<li><a href="/p/${sharedId}">Open on the server</a></li>`
				: html`<li><button @click=${() => this.#share()}>Save on the server</button></li>`;
		}
		return html`
			<li><button @click=${() => navigator.clipboard.writeText(location.href)}>Copy link</button></li>
			<li><button @click=${() => this.#copyToServer()}>${this.#isLoggedIn ? "Copy to my documents" : "Make a copy"}</button></li>
			${this.#server?.isMine ? html`
				<li>
					<label>Access
						<select @change=${event => this.#setVisibility(event.target.value)}>
							${visibilities.map(({visibility, label}) => html`<option value=${visibility} ?selected=${visibility === this.#server.visibility}>${label}</option>`)}
						</select>
					</label>
				</li>
			` : ""}
		`;
	}

	#dialogsTemplate() {
		return html`
			<dialog id="rename-dialog">
				<form method="dialog" @submit=${event => this.#rename(new FormData(event.target).get("name"))}>
					<label>Name <input name="name" required .value=${this.#documentName}></label>
					<button>Rename</button>
					<button type="button" @click=${event => event.target.closest("dialog").close()}>Cancel</button>
				</form>
			</dialog>
			<dialog id="export-dialog">
				<pre>${this.#exportedText}</pre>
				<button @click=${() => navigator.clipboard.writeText(this.#exportedText)}>Copy</button>
				<button @click=${event => event.target.closest("dialog").close()}>Close</button>
			</dialog>
			<dialog id="existing-document-dialog">
				<p>A document with the same ID already exists. Keep both, or replace the existing document?</p>
				<form method="dialog">
					<button @click=${() => this.#applyOpeningFile({isCopy: true})}>Keep both</button>
					<button @click=${() => this.#applyOpeningFile({isCopy: false})}>Replace</button>
					<button>Cancel</button>
				</form>
			</dialog>
			<dialog id="remove-dialog">
				<p>Remove "${this.#removingDocument?.name}"? This cannot be undone.</p>
				<form method="dialog">
					<button @click=${() => this.#remove(this.#removingDocument.rootId)}>Remove</button>
					<button>Cancel</button>
				</form>
			</dialog>
			<dialog id="conflict-dialog">
				<p>This document was changed on the server while you were editing it.</p>
				<form method="dialog">
					<button @click=${() => this.#keepMine()}>Overwrite with mine</button>
					<button @click=${() => this.#takeServers()}>Take the server's</button>
					<button @click=${() => this.#saveMineAsCopy()}>Save mine as a copy</button>
				</form>
			</dialog>
			<dialog id="error-dialog">
				<p>${this.#errorMessage}</p>
				<form method="dialog"><button>Close</button></form>
			</dialog>
			<dialog id="shortcuts-dialog">
				<table>
					${shortcuts.map(shortcut => html`<tr><td>${shortcut.label}</td><td><kbd>${keyLabel(shortcut)}</kbd></td></tr>`)}
				</table>
				<form method="dialog"><button>Close</button></form>
			</dialog>
		`;
	}

	render() {
		render(this.template(), this.shadowRoot);
	}

	get #canvas() {
		return this.shadowRoot.querySelector("canvas");
	}

	#showDialog(id) {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		this.render();
		this.shadowRoot.getElementById(id).showModal();
	}

	async connectedCallback() {
		// The cell size is measured with the font, so wait for it.
		await Promise.all([this.#loadEngine(), document.fonts.load(`${minFontSize}px D2Coding`, "가a")]);
		this.render();
		this.#measureCell();

		new ResizeObserver(() => this.#resize()).observe(this.#canvas);
		this.#applyTheme();
		darkSchemeQuery.addEventListener("change", () => this.#applyTheme());
		document.addEventListener("keydown", event => this.#keyDown(event));
		document.addEventListener("keyup", event => {
			if (event.key === "Shift") {
				this.#engine.shiftKey(false);
			}
		});
		document.addEventListener("paste", event => this.#paste(event));

		this.#serverId = this.getAttribute("server-id");
		if (this.#serverId) {
			this.#isLoggedIn = await isLoggedIn();
			await this.#openServerDocument();
			return;
		}
		// The other pages ask for a new document with #new, as it is made on /edit.
		if (location.hash === "#new") {
			history.replaceState(null, "", location.pathname);
			await this.#newDocument();
			return;
		}
		// /edit opens documents in this browser only. Server documents open on their pages.
		const lastDocument = (await listDocuments()).find(localDocument => !localDocument.serverId);
		if (lastDocument) {
			await this.#restoreDocument(lastDocument);
		} else {
			await this.#newDocument();
		}
	}

	async #loadEngine() {
		if (!window.nemokan) {
			const go = new Go();
			const {instance} = await WebAssembly.instantiateStreaming(fetch(this.getAttribute("wasm-src")), go.importObject);
			go.run(instance);
		}
		this.#engine = window.nemokan.start(change => this.#onChange(change));
		this.#styleOptions = JSON.parse(this.#engine.styleOptions());
	}

	// measureCell makes a cell as tall as a vertical line glyph, so vertical lines have no gaps.
	// It is rounded down to overlap the glyphs a little. The original uses 1.312 times the font
	// size, which fits JetBrains Mono. See ADR-0009.
	#measureCell() {
		const context = this.#canvas.getContext("2d");
		context.font = this.#font;
		context.textBaseline = "top";
		const vertical = context.measureText("│");
		this.#cellSizePx = {
			width: context.measureText("M").width,
			height: Math.floor(vertical.actualBoundingBoxAscent + vertical.actualBoundingBoxDescent),
		};
		this.#charOffsetTop = vertical.actualBoundingBoxAscent;
	}

	#setTheme(theme) {
		this.#theme = theme;
		try {
			localStorage.setItem(themeKey, theme);
		} catch {
			// The theme is kept only for this page.
		}
		this.#applyTheme();
	}

	// applyTheme sets the theme attribute the colors depend on, and redraws the canvas with them.
	#applyTheme() {
		const isDark = this.#theme === "dark" || (this.#theme === "system" && darkSchemeQuery.matches);
		this.setAttribute("theme", isDark ? "dark" : "light");
		this.#onChange("board");
	}

	#setShapeToolHidden(isHidden) {
		this.#isShapeToolHidden = isHidden;
		try {
			localStorage.setItem(shapeToolHiddenKey, String(isHidden));
		} catch {
			// The setting is kept only for this page.
		}
		this.render();
	}

	async #setFontSize(fontSize) {
		await document.fonts.load(`${fontSize}px D2Coding`, "가a");
		const oldCellSize = this.#cellSizePx;
		this.#fontSize = fontSize;
		try {
			localStorage.setItem(fontSizeKey, String(fontSize));
		} catch {
			// The size is kept only for this page.
		}
		this.#measureCell();
		// Keep the same board position at the top-left, as the original.
		this.#offsetPx = {
			left: Math.round(this.#offsetPx.left * this.#cellSizePx.width / oldCellSize.width),
			top: Math.round(this.#offsetPx.top * this.#cellSizePx.height / oldCellSize.height),
		};
		this.render();
		this.#setDrawingInfo();
	}

	#onChange(change) {
		this.#changes.add(change);
		if (this.#isFrameRequested) {
			return;
		}
		this.#isFrameRequested = true;
		requestAnimationFrame(() => this.#frame());
	}

	// frame reads what changed since the last frame, once.
	#frame() {
		this.#isFrameRequested = false;
		const changes = this.#changes;
		this.#changes = new Set();

		if (changes.has("interactionBounds")) {
			this.#interactionBounds = JSON.parse(this.#engine.interactionBounds());
		}
		if (changes.has("selectionBound")) {
			const bound = this.#engine.selectionBound();
			this.#selectionBound = bound ? JSON.parse(bound) : null;
		}
		if (changes.has("cursor")) {
			this.#engineCursor = this.#engine.cursor();
			this.#canvas.style.cursor = this.#engineCursor;
		}
		if (changes.has("clipboard")) {
			const text = this.#engine.takeClipboardText();
			if (text !== null) {
				navigator.clipboard.writeText(text).catch(error => console.error("clipboard", error));
			}
		}
		if (changes.has("backup")) {
			clearTimeout(this.#backupTimer);
			this.#backupTimer = setTimeout(() => this.#backup(), backupDelayMillis);
		}
		if (changes.has("textEditing")) {
			const editing = this.#engine.textEditing();
			this.#textEditing = editing ? JSON.parse(editing) : null;
		}
		if (changes.has("shapeTool")) {
			this.#shapeTool = JSON.parse(this.#engine.shapeTool());
		}
		if (changes.has("layers")) {
			this.#layers = JSON.parse(this.#engine.layers());
		}
		if (changes.has("textEditing") || changes.has("retainableAction") || changes.has("shapeTool") || changes.has("layers")) {
			this.render();
			if (this.#textEditing) {
				this.shadowRoot.querySelector("textarea").focus();
			}
		}
		this.#draw();
	}

	async #backup() {
		const backup = this.#engine.backup();
		if (backup !== null) {
			await saveBackup(JSON.parse(backup));
			this.#requestSync();
		}
	}

	// Documents

	async #refreshDocuments() {
		this.#documents = await listDocuments();
		this.render();
	}

	#showDocument(localDocument) {
		this.#documentName = localDocument?.name ?? defaultName;
		this.#offsetPx = localDocument?.offset ?? {left: 0, top: 0};
		this.#setDrawingInfo();
		this.render();
	}

	async #newDocument() {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		if (this.#serverId) {
			location.href = "/edit#new";
			return;
		}
		this.#engine.newDocument();
		await updateDocument(this.#engine.rootId(), {openedAt: Date.now()});
		this.#showDocument(null);
	}

	// openDocument opens a document of the list. A server document opens on its page, and a
	// document in this browser opens on /edit.
	async #openDocument(localDocument) {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		if ((localDocument.serverId ?? null) === this.#serverId) {
			await this.#restoreDocument(localDocument);
			return;
		}
		await updateDocument(localDocument.rootId, {openedAt: Date.now()});
		location.href = localDocument.serverId ? `/p/${localDocument.serverId}` : "/edit";
	}

	async #restoreDocument(localDocument) {
		this.#engine.restore(localDocument.rootId, localDocument.root ? JSON.stringify(localDocument) : null);
		// Restoring asks for a backup of what is already kept. Taking it now keeps the document
		// from looking changed, which would save it on the server again.
		this.#engine.backup();
		await updateDocument(localDocument.rootId, {openedAt: Date.now()});
		this.#showDocument(localDocument);
	}

	async #rename(name) {
		this.#documentName = name;
		await updateDocument(this.#engine.rootId(), {name, updatedAt: Date.now()});
		this.render();
		this.#requestSync();
	}

	#confirmRemove(localDocument) {
		this.#removingDocument = localDocument;
		this.#showDialog("remove-dialog");
	}

	// remove removes the document from this browser. A server document stays on the server.
	async #remove(rootId) {
		await removeDocument(rootId);
		if (rootId !== this.#engine.rootId()) {
			return;
		}
		if (this.#serverId) {
			location.href = "/edit";
			return;
		}
		// The current document is removed, so open the next one.
		const nextDocument = (await listDocuments()).find(localDocument => !localDocument.serverId);
		if (nextDocument) {
			await this.#restoreDocument(nextDocument);
		} else {
			await this.#newDocument();
		}
	}

	// Server documents. See ADR-0011.

	#currentFile() {
		return this.#engine.file(this.#documentName, this.#offsetPx.left, this.#offsetPx.top);
	}

	#setSyncStatus(status) {
		this.#syncStatus = status;
		this.render();
	}

	// isViewOnly is true for a server document the user cannot edit. See ADR-0012.
	get #isViewOnly() {
		return this.#server !== null && !this.#server.canEdit;
	}

	// setServer is the only place that changes both the server permission and the engine access,
	// so the two never differ. The server response decides both. See ADR-0012.
	#setServer(serverDocument) {
		this.#server = serverDocument;
		this.#engine.setAccess(serverDocument.canEdit ? "edit" : "view");
		this.render();
	}

	async #openServerDocument() {
		const local = await findDocumentByServerId(this.#serverId);
		const {status, serverDocument} = await getServerDocument(this.#serverId);
		if (status !== 200) {
			this.#errorMessage = serverErrorMessage(status);
			this.#showDialog("error-dialog");
			if (local) {
				await this.#restoreDocument(local);
			}
			this.#setSyncStatus(status === 0 ? "Offline" : "Not on the server");
			return;
		}
		this.#setServer(serverDocument);
		if (!local || (serverDocument.revision !== local.revision && !hasUnsyncedChanges(local))) {
			await this.#applyServerDocument(serverDocument);
			return;
		}
		await this.#restoreDocument(local);
		if (serverDocument.revision === local.revision) {
			this.#setSyncStatus(hasUnsyncedChanges(local) ? "Not saved yet" : "Saved");
			this.#requestSync();
			return;
		}
		// Both the server and this browser have changes.
		this.#showConflict(serverDocument);
	}

	// applyServerDocument shows the server document and keeps it as synced in this browser.
	async #applyServerDocument(serverDocument) {
		const info = this.#engine.fileInfo(serverDocument.content);
		if (info === null) {
			this.#errorMessage = "The document on the server cannot be opened.";
			this.#showDialog("error-dialog");
			return;
		}
		const local = await getDocument(info.rootId);
		this.#engine.openFile(serverDocument.content);
		// Opening asks for a backup. Taking it now keeps it older than syncedAt.
		const backup = this.#engine.backup();
		if (backup !== null) {
			await saveBackup(JSON.parse(backup));
		}
		await updateDocument(info.rootId, {
			name: info.name || defaultName,
			offset: local?.offset ?? info.offset,
			serverId: serverDocument.id,
			revision: serverDocument.revision,
			syncedAt: Date.now(),
			openedAt: Date.now(),
		});
		this.#showDocument(await getDocument(info.rootId));
		this.#setSyncStatus("Saved");
	}

	#requestSync() {
		if (!this.#serverId) {
			return;
		}
		clearTimeout(this.#syncTimer);
		this.#syncTimer = setTimeout(() => this.#sync(), syncDelayMillis);
	}

	// sync sends the changes not sent yet. A conflict asks the user what to keep.
	async #sync() {
		if (!this.#server?.canEdit) {
			return;
		}
		if (this.#isSyncing) {
			this.#isSyncRequested = true;
			return;
		}
		const rootId = this.#engine.rootId();
		const local = await getDocument(rootId);
		if (!local || !hasUnsyncedChanges(local)) {
			return;
		}
		this.#isSyncing = true;
		this.#setSyncStatus("Saving…");
		// Changes made while saving are newer than this, so they are sent next time.
		const capturedAt = Date.now();
		const {status, serverDocument} = await updateServerDocument(this.#serverId, local.revision, this.#currentFile());
		this.#isSyncing = false;

		switch (status) {
		case 200:
			await updateDocument(rootId, {revision: serverDocument.revision, syncedAt: capturedAt});
			this.#setServer(serverDocument);
			this.#setSyncStatus("Saved");
			break;
		case 409: {
			const latest = await getServerDocument(this.#serverId);
			if (latest.status === 200) {
				this.#showConflict(latest.serverDocument);
			} else {
				this.#setSyncStatus("Not saved");
			}
			return;
		}
		case 403:
			this.#setServer({...this.#server, canEdit: false});
			return;
		case 0:
			this.#setSyncStatus("Offline. Saved in this browser.");
			clearTimeout(this.#syncTimer);
			this.#syncTimer = setTimeout(() => this.#sync(), syncRetryMillis);
			return;
		default:
			this.#setSyncStatus("Not saved");
			this.#errorMessage = serverErrorMessage(status);
			this.#showDialog("error-dialog");
			return;
		}
		if (this.#isSyncRequested) {
			this.#isSyncRequested = false;
			await this.#sync();
		}
	}

	#showConflict(serverDocument) {
		this.#conflictingDocument = serverDocument;
		this.#setSyncStatus("Conflict");
		this.#showDialog("conflict-dialog");
	}

	// keepMine saves this browser's document over the newer server one.
	async #keepMine() {
		const serverDocument = this.#conflictingDocument;
		this.#conflictingDocument = null;
		await updateDocument(this.#engine.rootId(), {revision: serverDocument.revision});
		await this.#sync();
	}

	async #takeServers() {
		const serverDocument = this.#conflictingDocument;
		this.#conflictingDocument = null;
		await this.#applyServerDocument(serverDocument);
	}

	// saveMineAsCopy makes a new server document of this browser's changes, and leaves the
	// server document as it is. The changes now live in the copy, so the old local one goes.
	// If the copy cannot be made, the user chooses again.
	async #saveMineAsCopy() {
		const originalRootId = this.#engine.rootId();
		const copy = await this.#createServerCopy();
		if (!copy) {
			this.shadowRoot.getElementById("error-dialog").addEventListener("close", () => this.#showDialog("conflict-dialog"), {once: true});
			return;
		}
		this.#conflictingDocument = null;
		await removeDocument(originalRootId);
		location.href = `/p/${copy.id}`;
	}

	// copyToServer makes a new server document of this one, owned by the user if logged in.
	async #copyToServer() {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		const copy = await this.#createServerCopy();
		if (copy) {
			location.href = `/p/${copy.id}`;
		}
	}

	// createServerCopy saves a copy of this document on the server, with a new root id since
	// this browser keeps documents by their root id. The copy opens from the server on its
	// page. It returns the new server document, or null if the copy cannot be made. Then this
	// document stays open as it was.
	async #createServerCopy() {
		const {status, serverDocument} = await createServerDocument(this.#engine.fileAsCopy(this.#currentFile()));
		if (status !== 201) {
			this.#errorMessage = "The copy could not be made. " + serverErrorMessage(status);
			this.#showDialog("error-dialog");
			return null;
		}
		return serverDocument;
	}

	// share saves this browser's document on the server and opens its page.
	async #share() {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		const rootId = this.#engine.rootId();
		const capturedAt = Date.now();
		const {status, serverDocument} = await createServerDocument(this.#currentFile());
		if (status !== 201) {
			this.#errorMessage = serverErrorMessage(status);
			this.#showDialog("error-dialog");
			return;
		}
		await updateDocument(rootId, {serverId: serverDocument.id, revision: serverDocument.revision, syncedAt: capturedAt});
		location.href = `/p/${serverDocument.id}`;
	}

	async #setVisibility(visibility) {
		const {status, serverDocument} = await setServerDocumentVisibility(this.#serverId, visibility);
		if (status !== 200) {
			this.#errorMessage = serverErrorMessage(status);
			this.#showDialog("error-dialog");
			return;
		}
		this.#setServer(serverDocument);
	}

	#saveFile() {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		saveFile(this.#documentName, this.#currentFile());
	}

	async #openFile() {
		this.shadowRoot.getElementById("document-menu").hidePopover();
		const content = await openFile();
		if (content === null) {
			return;
		}
		const info = this.#engine.fileInfo(content);
		if (info === null) {
			this.#errorMessage = "The file cannot be opened. It may be broken or from another app.";
			this.#showDialog("error-dialog");
			return;
		}
		this.#openingFile = {content, info};
		const existing = await getDocument(info.rootId);
		if (existing?.root) {
			this.#showDialog("existing-document-dialog");
			return;
		}
		await this.#applyOpeningFile({isCopy: false});
	}

	async #applyOpeningFile({isCopy}) {
		const {content, info} = this.#openingFile;
		this.#openingFile = null;
		if (isCopy) {
			this.#engine.openFileAsCopy(content);
		} else {
			this.#engine.openFile(content);
		}
		const localDocument = {name: info.name || defaultName, offset: info.offset, openedAt: Date.now()};
		await updateDocument(this.#engine.rootId(), localDocument);
		this.#showDocument(localDocument);
	}

	#setExportStatus(status) {
		this.#exportStatus = status;
		this.render();
		clearTimeout(this.#exportStatusTimer);
		this.#exportStatusTimer = setTimeout(() => {
			this.#exportStatus = "";
			this.render();
		}, exportStatusMillis);
	}

	async #copyMarkdown() {
		const text = this.#engine.exportText();
		if (text === "") {
			this.#setExportStatus("Nothing to export");
			return;
		}
		try {
			await navigator.clipboard.writeText(markdown(text));
			this.#setExportStatus("Copied");
		} catch (error) {
			console.error("clipboard", error);
			this.#setExportStatus("Cannot copy");
		}
	}

	// imageOptions are how the exported image looks: as the canvas, in the current theme.
	get #imageOptions() {
		const style = getComputedStyle(this);
		return {
			cellSizePx: this.#cellSizePx,
			charOffsetTop: this.#charOffsetTop,
			fontSizePx: this.#fontSize,
			fontFamily,
			font: this.#font,
			background: style.getPropertyValue("--background"),
			color: style.getPropertyValue("--shape"),
		};
	}

	#downloadSvg() {
		const rows = this.#engine.exportRows();
		if (rows.length === 0) {
			this.#setExportStatus("Nothing to export");
			return;
		}
		download(`${this.#documentName}.svg`, new Blob([svg(rows, this.#imageOptions)], {type: "image/svg+xml"}));
	}

	async #downloadPng() {
		const rows = this.#engine.exportRows();
		if (rows.length === 0) {
			this.#setExportStatus("Nothing to export");
			return;
		}
		download(`${this.#documentName}.png`, await png(rows, this.#imageOptions));
	}

	#showExport() {
		this.#exportedText = this.#engine.exportText();
		this.#showDialog("export-dialog");
	}

	// Drawing

	#resize() {
		const canvas = this.#canvas;
		this.#canvasSizePx = {width: canvas.clientWidth, height: canvas.clientHeight};
		// Go only knows the board area, inside the axes.
		this.#boardSizePx = {width: Math.max(this.#canvasSizePx.width - axisWidthPx, 1), height: Math.max(this.#canvasSizePx.height - axisHeightPx, 1)};
		const ratio = Math.max(devicePixelRatio, 2);
		canvas.width = this.#canvasSizePx.width * ratio;
		canvas.height = this.#canvasSizePx.height * ratio;
		canvas.getContext("2d").setTransform(ratio, 0, 0, ratio, 0, 0);
		this.#setDrawingInfo();
	}

	#setDrawingInfo() {
		this.#engine.setDrawingInfo(JSON.stringify({
			offsetPx: this.#offsetPx,
			cellSizePx: this.#cellSizePx,
			canvasSizePx: this.#boardSizePx,
		}));
		// Interaction bounds are in board units, so they move with the offset.
		this.#onChange("board");
	}

	#toXPx(column) {
		return Math.floor(this.#offsetPx.left + this.#cellSizePx.width * column);
	}

	#toYPx(row) {
		return Math.floor(this.#offsetPx.top + this.#cellSizePx.height * row);
	}

	#draw() {
		const context = this.#canvas.getContext("2d");
		const style = getComputedStyle(this);
		context.fillStyle = style.getPropertyValue("--background");
		context.fillRect(0, 0, this.#canvasSizePx.width, this.#canvasSizePx.height);

		context.save();
		context.translate(axisWidthPx, axisHeightPx);
		context.beginPath();
		context.rect(0, 0, this.#boardSizePx.width, this.#boardSizePx.height);
		context.clip();
		this.#drawGrid(context, style);

		const board = this.#engine.board();
		context.font = this.#font;
		context.textBaseline = "top";
		const colors = highlightColors.map(name => style.getPropertyValue(name));
		board.rows.forEach((row, rowIndex) => {
			const chars = Array.from(row);
			chars.forEach((char, columnIndex) => {
				if (char === " " || char === wideCharTail) {
					return;
				}
				const x = this.#toXPx(board.left + columnIndex);
				const y = this.#toYPx(board.top + rowIndex) + this.#charOffsetTop;
				context.fillStyle = colors[board.highlights[rowIndex * board.width + columnIndex]];
				if (chars[columnIndex + 1] !== wideCharTail) {
					context.fillText(char, x, y);
					return;
				}
				// A wide char fits in two cells even if the font draws it wider or narrower.
				const wideWidth = this.#toXPx(board.left + columnIndex + 2) - x;
				const charWidth = Math.min(context.measureText(char).width, wideWidth);
				context.fillText(char, x + (wideWidth - charWidth) / 2, y, wideWidth);
			});
		});

		this.#drawSelectionBound(context, style);
		this.#drawInteractionBounds(context, style);
		context.restore();

		this.#drawAxes(context, style);
	}

	// boardRange returns the first and the last row or column on the canvas.
	#boardRange(offsetPx, cellSizePx, canvasSizePx) {
		const first = Math.floor(-offsetPx / cellSizePx);
		return {first, last: first + Math.ceil(canvasSizePx / cellSizePx)};
	}

	#drawGrid(context, style) {
		const rows = this.#boardRange(this.#offsetPx.top, this.#cellSizePx.height, this.#boardSizePx.height);
		const columns = this.#boardRange(this.#offsetPx.left, this.#cellSizePx.width, this.#boardSizePx.width);
		const grid = new Path2D();
		for (let row = rows.first; row <= rows.last; row++) {
			grid.moveTo(0, this.#toYPx(row));
			grid.lineTo(this.#boardSizePx.width, this.#toYPx(row));
		}
		for (let column = columns.first; column <= columns.last; column++) {
			grid.moveTo(this.#toXPx(column), 0);
			grid.lineTo(this.#toXPx(column), this.#boardSizePx.height);
		}
		context.strokeStyle = style.getPropertyValue("--grid-line");
		context.lineWidth = 0.25;
		context.stroke(grid);

		const zeroLines = new Path2D();
		zeroLines.moveTo(0, this.#toYPx(0));
		zeroLines.lineTo(this.#boardSizePx.width, this.#toYPx(0));
		zeroLines.moveTo(this.#toXPx(0), 0);
		zeroLines.lineTo(this.#toXPx(0), this.#boardSizePx.height);
		context.strokeStyle = style.getPropertyValue("--grid-line-zero");
		context.lineWidth = 1;
		context.stroke(zeroLines);
	}

	// drawAxes draws the row numbers on the left and every 20th column number on the top.
	#drawAxes(context, style) {
		const rows = this.#boardRange(this.#offsetPx.top, this.#cellSizePx.height, this.#boardSizePx.height);
		const columns = this.#boardRange(this.#offsetPx.left, this.#cellSizePx.width, this.#boardSizePx.width);
		context.font = `${axisFontSizePx}px ${fontFamily}`;
		context.textBaseline = "top";
		const rules = new Path2D();

		context.fillStyle = style.getPropertyValue("--axis-background");
		context.fillRect(0, 0, axisWidthPx, this.#canvasSizePx.height);
		context.fillStyle = style.getPropertyValue("--axis-text");
		context.textAlign = "right";
		for (let row = rows.first; row <= rows.last; row++) {
			context.fillText(String(row), axisWidthPx - 0.5 * this.#cellSizePx.width, axisHeightPx + this.#toYPx(row) + 3);
		}
		rules.moveTo(axisWidthPx, axisHeightPx);
		rules.lineTo(axisWidthPx, this.#canvasSizePx.height);

		context.fillStyle = style.getPropertyValue("--axis-background");
		context.fillRect(0, 0, this.#canvasSizePx.width, axisHeightPx);
		context.fillStyle = style.getPropertyValue("--axis-text");
		context.textAlign = "left";
		const firstColumn = Math.ceil(columns.first / axisColumnStep) * axisColumnStep;
		for (let column = firstColumn; column <= columns.last; column += axisColumnStep) {
			const x = axisWidthPx + this.#toXPx(column);
			context.fillText(String(column), x + 2, 7);
			rules.moveTo(x, axisHeightPx - axisRulerSizePx);
			rules.lineTo(x, axisHeightPx);
		}
		rules.moveTo(0, axisHeightPx);
		rules.lineTo(this.#canvasSizePx.width, axisHeightPx);

		context.strokeStyle = style.getPropertyValue("--axis-rule");
		context.lineWidth = 1;
		context.stroke(rules);
	}

	#drawSelectionBound(context, style) {
		const bound = this.#selectionBound;
		if (!bound) {
			return;
		}
		context.strokeStyle = style.getPropertyValue("--selection-area-stroke");
		context.lineWidth = 1;
		context.setLineDash([8, 6]);
		const left = this.#toXPx(bound.left);
		const top = this.#toYPx(bound.top);
		context.strokeRect(left, top, this.#toXPx(bound.left + bound.width) - left, this.#toYPx(bound.top + bound.height) - top);
		context.setLineDash([]);
	}

	#drawInteractionBounds(context, style) {
		for (const bound of this.#interactionBounds) {
			if (bound.type === "scalable") {
				context.strokeStyle = style.getPropertyValue("--selection-bound-stroke");
				context.lineWidth = 1;
				const left = this.#toXPx(bound.left);
				const top = this.#toYPx(bound.top);
				context.strokeRect(left, top, this.#toXPx(bound.right) - left, this.#toYPx(bound.bottom) - top);
			}
			context.strokeStyle = style.getPropertyValue("--selection-dot-stroke");
			context.fillStyle = style.getPropertyValue("--selection-dot-fill");
			context.lineWidth = 2;
			for (const point of bound.points) {
				context.beginPath();
				context.arc(this.#toXPx(point.left), this.#toYPx(point.top), interactionDotRadiusPx, 0, 2 * Math.PI);
				context.stroke();
				context.fill();
			}
		}
	}

	// Input

	// mouseDown takes the focus from a field, such as a number in the shape tool, as the canvas
	// cannot be focused. Otherwise key shortcuts keep going to the field.
	// It captures the pointer, so the canvas gets the moves and the release even when the mouse
	// leaves it during a drag. Otherwise a drag released outside goes on until the next click.
	#mouseDown(event) {
		this.shadowRoot.activeElement?.blur();
		if (this.#isOnAxesCorner(event)) {
			this.#setOffset({left: 0, top: 0});
			return;
		}
		if (event.offsetX < axisWidthPx || event.offsetY < axisHeightPx) {
			return;
		}
		this.#canvas.setPointerCapture(event.pointerId);
		this.#mouse("mouseDown", event);
	}

	#mouseMove(event) {
		this.#canvas.style.cursor = this.#isOnAxesCorner(event) ? "pointer" : this.#engineCursor;
		this.#mouse("mouseMove", event);
		this.#setPointerCell({
			row: Math.floor((event.offsetY - axisHeightPx - this.#offsetPx.top) / this.#cellSizePx.height),
			column: Math.floor((event.offsetX - axisWidthPx - this.#offsetPx.left) / this.#cellSizePx.width),
		});
	}

	#setPointerCell(cell) {
		if (cell?.row === this.#pointerCell?.row && cell?.column === this.#pointerCell?.column) {
			return;
		}
		this.#pointerCell = cell;
		this.render();
	}

	// isOnAxesCorner reports whether the mouse is on the corner of the axes, which jumps to (0, 0).
	#isOnAxesCorner(event) {
		return event.offsetX < axisWidthPx && event.offsetY < axisHeightPx;
	}

	#mouse(name, event) {
		this.#engine[name](event.offsetX - axisWidthPx, event.offsetY - axisHeightPx, event.shiftKey, event.timeStamp);
	}

	// wheel scrolls the board. Alt swaps the directions, for a mouse with one wheel. The scroll
	// mode limits it to a direction.
	#wheel(event) {
		event.preventDefault();
		const deltaX = event.altKey ? event.deltaY : event.deltaX;
		const deltaY = event.altKey ? event.deltaX : event.deltaY;
		// Shift swaps a limited direction, as the original.
		let mode = this.#scrollMode.mode;
		if (event.shiftKey && mode !== "both") {
			mode = mode === "vertical" ? "horizontal" : "vertical";
		}
		const deltaLeft = mode === "vertical" ? 0 : deltaX;
		const deltaTop = mode === "horizontal" ? 0 : deltaY;

		const left = this.#wheelDelta.left + deltaLeft * scrollSpeedRatio;
		const top = this.#wheelDelta.top + deltaTop * scrollSpeedRatio;
		const usableLeft = Math.trunc(left);
		const usableTop = Math.trunc(top);
		this.#wheelDelta = {left: left - usableLeft, top: top - usableTop};

		if (usableLeft !== 0 || usableTop !== 0) {
			this.#setOffset({left: this.#offsetPx.left - usableLeft, top: this.#offsetPx.top - usableTop});
		}
	}

	#nextScrollMode() {
		this.#scrollMode = scrollModes[(scrollModes.indexOf(this.#scrollMode) + 1) % scrollModes.length];
		this.render();
	}

	#setOffset(offsetPx) {
		this.#offsetPx = offsetPx;
		this.#setDrawingInfo();
		// The offset is kept with the document, after scrolling stops.
		clearTimeout(this.#offsetTimer);
		const rootId = this.#engine.rootId();
		this.#offsetTimer = setTimeout(() => updateDocument(rootId, {offset: offsetPx}), backupDelayMillis);
	}

	#keyDown(event) {
		if (event.key === "Shift") {
			this.#engine.shiftKey(true);
		}
		if (isTyping(event)) {
			return;
		}
		const shortcut = findShortcut(event);
		if (!shortcut) {
			return;
		}
		if (shortcut.preventDefault) {
			event.preventDefault();
		}
		if (shortcut.retainable) {
			this.#engine.setRetainable(shortcut.retainable);
		} else {
			this.#action(shortcut.action);
		}
	}

	// textKeyDown finishes editing. A field of a widget has one line, so Enter finishes it.
	#textKeyDown(event) {
		const isEnterFinishing = event.key === "Enter" && (hasCommandKey(event) || this.#textEditing?.isSingleLine);
		if (event.key === "Escape" || isEnterFinishing) {
			event.preventDefault();
			event.target.blur();
		}
	}

	#paste(event) {
		if (isTyping(event)) {
			return;
		}
		event.preventDefault();
		this.#engine.paste(event.clipboardData.getData("text/plain"));
	}

	#action(action) {
		this.#engine.action(JSON.stringify(action));
	}

	#startRenameLayer(id) {
		this.#renamingLayerId = id;
		this.render();
		const input = this.shadowRoot.querySelector("#layers input");
		input?.focus();
		input?.select();
	}

	// finishRenameLayer sends action, or nothing if renaming is canceled. Escape and the blur
	// following it both finish, so only the first one counts.
	#finishRenameLayer(action) {
		if (this.#renamingLayerId === null) {
			return;
		}
		this.#renamingLayerId = null;
		if (action) {
			this.#action(action);
		}
		this.render();
	}

	// goToLayer selects the shape of layer. If the shape is not all on the canvas, it scrolls
	// the shape to the middle of the canvas.
	#goToLayer(layer) {
		this.#action({type: "selectShape", shapeId: layer.id});
		const cell = this.#cellSizePx;
		const leftPx = layer.position.left * cell.width + this.#offsetPx.left;
		const topPx = layer.position.top * cell.height + this.#offsetPx.top;
		const isOnCanvas = leftPx >= 0 && topPx >= 0
			&& leftPx + layer.size.width * cell.width <= this.#boardSizePx.width
			&& topPx + layer.size.height * cell.height <= this.#boardSizePx.height;
		if (isOnCanvas) {
			return;
		}
		this.#setOffset({
			left: Math.round(this.#boardSizePx.width / 2 - (layer.position.left + layer.size.width / 2) * cell.width),
			top: Math.round(this.#boardSizePx.height / 2 - (layer.position.top + layer.size.height / 2) * cell.height),
		});
	}
}
customElements.define("mono-editor", Editor);
