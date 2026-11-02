import {html} from 'lit-html';
import {findWidget} from "./widgets.js";

// The layers panel. Go keeps the layers (state.Layers) and each change goes back to Go as a
// one time action. See ADR-0013.

const typeIcons = {
	rectangle: "M22 19H2V5h20v14zM4 7v10h16V7z",
	text: "M5.635 21v-2h12.731v2zm3.27-4v-1.12h2.005V4.12H7.425l-.39.44v2.58h-1.4V3h12.731v4.14h-1.4V4.56l-.39-.44h-3.485v11.76h2.005V17z",
	line: "M18 15v-2H6v2H0V9h6v2h12V9h6v6z",
	group: "M2 4h8l2 2h10v14H2zm2 4v10h16V8z",
};

const typeLabels = {rectangle: "Rectangle", text: "Text", line: "Line", group: "Group"};

// The eye and lock icons are drawn with strokes in a 24 x 24 box.
const eyePath = "M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12zm10 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6z";
const eyeOffPath = eyePath + "M3 3l18 18";
const lockPath = "M5 11h14v10H5zM8 11V7a4 4 0 0 1 8 0v4";
const unlockPath = "M5 11h14v10H5zM8 11V7a4 4 0 0 1 7.5-2";

// defaultName is shown when the user has not named the shape: the first line of a text, or
// the kind of the shape.
function defaultName(layer) {
	if (layer.type === "widget") {
		return findWidget(layer.kind)?.label ?? layer.kind;
	}
	const firstLine = layer.text?.split("\n").find(line => line.trim() !== "");
	return firstLine?.trim() ?? typeLabels[layer.type];
}

function strokeIcon(path) {
	return html`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d=${path}/></svg>`;
}

// dropPlace tells where a dragged layer goes on the row under the mouse: "above", "below", or
// "into" the middle of a group row.
function dropPlace(event, isGroup) {
	const rect = event.currentTarget.getBoundingClientRect();
	const ratio = (event.clientY - rect.top) / rect.height;
	if (isGroup && ratio > 0.25 && ratio < 0.75) {
		return "into";
	}
	return ratio < 0.5 ? "above" : "below";
}

function clearDropPlace(row) {
	row.classList.remove("drop-above", "drop-below", "drop-into");
}

// layersTemplate renders the panel. handlers has send, which takes a one time action, and
// renamingId with startRename and finishRename for renaming a layer in place.
export function layersTemplate(layers, handlers) {
	// rows renders the items of one group. parentId is empty for the root. Each item knows the
	// one below it, as a layer dropped under a row goes above the next lower item.
	const rows = (items, parentId, depth) => items.map((layer, index) => {
		const lowerId = items[index + 1]?.id ?? "";
		const isGroup = layer.type === "group";
		const drop = event => {
			event.preventDefault();
			clearDropPlace(event.currentTarget);
			const shapeId = event.dataTransfer.getData("text/plain");
			if (!shapeId || shapeId === layer.id) {
				return;
			}
			switch (dropPlace(event, isGroup)) {
			case "into":
				handlers.send({type: "moveShapeInLayers", shapeId, parentId: layer.id, belowId: layer.items?.[0]?.id ?? ""});
				break;
			case "above":
				handlers.send({type: "moveShapeInLayers", shapeId, parentId, belowId: layer.id});
				break;
			default:
				handlers.send({type: "moveShapeInLayers", shapeId, parentId, belowId: lowerId});
			}
		};
		return html`
			<li>
				<div
					class="layer"
					style="padding-left: ${0.5 + depth}rem"
					draggable=${handlers.renamingId === layer.id ? "false" : "true"}
					aria-selected=${layer.isSelected}
					?data-hidden=${layer.isHidden}
					?data-working=${layer.id === layers.workingGroupId}
					@click=${event => handlers.send({type: event.shiftKey || event.metaKey || event.ctrlKey ? "toggleShapeSelection" : "selectShape", shapeId: layer.id})}
					@dblclick=${() => handlers.startRename(layer.id)}
					@dragstart=${event => {
						event.dataTransfer.setData("text/plain", layer.id);
						event.dataTransfer.effectAllowed = "move";
					}}
					@dragover=${event => {
						event.preventDefault();
						clearDropPlace(event.currentTarget);
						event.currentTarget.classList.add(`drop-${dropPlace(event, isGroup)}`);
					}}
					@dragleave=${event => clearDropPlace(event.currentTarget)}
					@drop=${drop}
				>
					${layer.type === "widget"
						? html`<i class="widget-icon">${findWidget(layer.kind)?.icon}</i>`
						: html`<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d=${typeIcons[layer.type]}/></svg>`}
					${handlers.renamingId === layer.id ? html`
						<input
							.value=${layer.name || defaultName(layer)}
							@click=${event => event.stopPropagation()}
							@keydown=${event => {
								if (event.key === "Enter") {
									event.target.blur();
								} else if (event.key === "Escape") {
									handlers.finishRename(null);
								}
							}}
							@blur=${event => handlers.finishRename({type: "renameShape", shapeId: layer.id, name: event.target.value.trim()})}
						>
					` : html`<span class=${layer.name ? "" : "default-name"}>${layer.name || defaultName(layer)}</span>`}
					<button
						aria-pressed=${layer.isHidden}
						title=${layer.isHidden ? "Show" : "Hide"}
						@click=${event => {
							event.stopPropagation();
							handlers.send({type: "setShapeHidden", shapeId: layer.id, isHidden: !layer.isHidden});
						}}
					>${strokeIcon(layer.isHidden ? eyeOffPath : eyePath)}</button>
					<button
						aria-pressed=${layer.isLocked}
						title=${layer.isLocked ? "Unlock" : "Lock"}
						@click=${event => {
							event.stopPropagation();
							handlers.send({type: "setShapeLocked", shapeId: layer.id, isLocked: !layer.isLocked});
						}}
					>${strokeIcon(layer.isLocked ? lockPath : unlockPath)}</button>
				</div>
				${isGroup ? html`<ul>${rows(layer.items ?? [], layer.id, depth + 1)}</ul>` : ""}
			</li>
		`;
	});

	return html`
		<section id="layers">
			<h2>Layers</h2>
			${layers.items.length === 0
				? html`<p>No shapes yet.</p>`
				: html`<ul>${rows(layers.items, "", 0)}</ul>`}
		</section>
	`;
}
