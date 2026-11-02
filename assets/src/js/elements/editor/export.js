// Exporting the shapes as Markdown, SVG and PNG, from what Go exports. Go picks the shapes: the
// selected ones, or all of them if none is selected.

// wideCharTail is the second cell of a wide char in the rows. See ADR-0008.
const wideCharTail = "\0";

// markdown wraps text in a code block, so it keeps its layout when pasted in Markdown. The
// spaces at the end of lines are left out.
export function markdown(text) {
	return "```\n" + text.split("\n").map(line => line.trimEnd()).join("\n") + "\n```\n";
}

// runs splits a row into runs of chars to draw at once. A wide char is a run of its own,
// two cells wide. Spaces are left out.
function runs(row) {
	const chars = Array.from(row);
	const result = [];
	let current = null;
	chars.forEach((char, column) => {
		if (char === " " || char === wideCharTail) {
			current = null;
			return;
		}
		if (chars[column + 1] === wideCharTail) {
			result.push({column, text: char, cells: 2});
			current = null;
			return;
		}
		if (current) {
			current.text += char;
			current.cells++;
			return;
		}
		current = {column, text: char, cells: 1};
		result.push(current);
	});
	return result;
}

function escapeXml(text) {
	return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// svg draws the rows as an SVG image. textLength keeps each run on the cells even when the
// viewer draws it with another font.
export function svg(rows, {cellSizePx, fontSizePx, fontFamily, background, color}) {
	const width = Math.max(...rows.map(row => Array.from(row).length)) * cellSizePx.width;
	const height = rows.length * cellSizePx.height;
	const texts = rows.flatMap((row, rowIndex) => runs(row).map(run => {
		const y = (rowIndex + 0.5) * cellSizePx.height;
		return `<text x="${run.column * cellSizePx.width}" y="${y}" textLength="${run.cells * cellSizePx.width}" lengthAdjust="spacingAndGlyphs">${escapeXml(run.text)}</text>`;
	}));
	return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
<style>text { font-family: ${escapeXml(fontFamily)}; font-size: ${fontSizePx}px; fill: ${color}; dominant-baseline: central; white-space: pre; }</style>
<rect width="100%" height="100%" fill="${background}"/>
${texts.join("\n")}
</svg>
`;
}

// png draws the rows on a canvas as the editor does, and resolves with a PNG blob.
export function png(rows, {cellSizePx, charOffsetTop, font, background, color}) {
	const ratio = Math.max(devicePixelRatio, 2);
	const width = Math.max(...rows.map(row => Array.from(row).length)) * cellSizePx.width;
	const height = rows.length * cellSizePx.height;
	const canvas = document.createElement("canvas");
	canvas.width = Math.ceil(width * ratio);
	canvas.height = Math.ceil(height * ratio);
	const context = canvas.getContext("2d");
	context.scale(ratio, ratio);
	context.fillStyle = background;
	context.fillRect(0, 0, width, height);
	context.font = font;
	context.textBaseline = "top";
	context.fillStyle = color;
	rows.forEach((row, rowIndex) => {
		const chars = Array.from(row);
		const y = rowIndex * cellSizePx.height + charOffsetTop;
		chars.forEach((char, column) => {
			if (char === " " || char === wideCharTail) {
				return;
			}
			const x = column * cellSizePx.width;
			if (chars[column + 1] !== wideCharTail) {
				context.fillText(char, x, y);
				return;
			}
			// A wide char fits in two cells even if the font draws it wider or narrower.
			const wideWidth = 2 * cellSizePx.width;
			const charWidth = Math.min(context.measureText(char).width, wideWidth);
			context.fillText(char, x + (wideWidth - charWidth) / 2, y, wideWidth);
		});
	});
	return new Promise((resolve, reject) => canvas.toBlob(blob => blob ? resolve(blob) : reject(new Error("The image cannot be made.")), "image/png"));
}
