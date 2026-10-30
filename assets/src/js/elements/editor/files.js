// Saving and opening document files with the browser file API. See ADR-0010.

// extension of document files. The content is not compatible with the original MonoSketch.
export const extension = "mono";

export function saveFile(name, content) {
	const link = document.createElement("a");
	link.href = URL.createObjectURL(new Blob([content], {type: "application/json"}));
	link.download = `${name}.${extension}`;
	link.click();
	URL.revokeObjectURL(link.href);
}

// openFile resolves with the text of the chosen file, or null if nothing is chosen.
export function openFile() {
	return new Promise(resolve => {
		const input = document.createElement("input");
		input.type = "file";
		input.accept = `.${extension}`;
		input.addEventListener("change", async () => resolve(input.files[0] ? await input.files[0].text() : null));
		input.addEventListener("cancel", () => resolve(null));
		input.click();
	});
}
