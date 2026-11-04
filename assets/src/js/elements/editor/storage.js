// Local storage of documents in IndexedDB. See ADR-0002, ADR-0007 and ADR-0010.
//
// A document is {rootId, name, offset, root, connectors, updatedAt, openedAt}. root and
// connectors are the backup from Go.
//
// A document synced with the server also has {serverId, revision, syncedAt}. revision is the
// server revision the local content is based on. It has changes not sent to the server yet
// if updatedAt > syncedAt. See ADR-0011.
const databaseName = "nemokan";
// databaseVersion 2 renamed the store from "projects" to "documents".
const databaseVersion = 2;
const storeName = "documents";
const oldStoreName = "projects";

// defaultName is the name of a new document, the same as the original.
export const defaultName = "Undefined";

function request(idbRequest) {
	return new Promise((resolve, reject) => {
		idbRequest.onsuccess = () => resolve(idbRequest.result);
		idbRequest.onerror = () => reject(idbRequest.error);
	});
}

let database;

function open() {
	database ??= new Promise((resolve, reject) => {
		const opening = indexedDB.open(databaseName, databaseVersion);
		opening.onupgradeneeded = () => {
			const upgrading = opening.result;
			const documents = upgrading.createObjectStore(storeName, {keyPath: "rootId"});
			if (!upgrading.objectStoreNames.contains(oldStoreName)) {
				return;
			}
			// Keep the drawings saved before the rename.
			const reading = opening.transaction.objectStore(oldStoreName).getAll();
			reading.onsuccess = () => {
				for (const localDocument of reading.result) {
					documents.put(localDocument);
				}
				upgrading.deleteObjectStore(oldStoreName);
			};
		};
		opening.onsuccess = () => resolve(opening.result);
		opening.onerror = () => reject(opening.error);
	});
	return database;
}

async function store(mode) {
	return (await open()).transaction(storeName, mode).objectStore(storeName);
}

export async function getDocument(rootId) {
	return request((await store("readonly")).get(rootId));
}

// updateDocument merges change into the document of rootId, creating it if there is none.
export async function updateDocument(rootId, change) {
	const current = await getDocument(rootId) ?? {rootId, name: defaultName, offset: {left: 0, top: 0}, openedAt: Date.now()};
	await request((await store("readwrite")).put({...current, ...change, rootId}));
}

// saveBackup keeps a backup from Go.
export async function saveBackup(backup) {
	await updateDocument(backup.rootId, {root: backup.root, connectors: backup.connectors, updatedAt: Date.now()});
}

// findDocumentByServerId returns the local document synced with the server document, if any.
export async function findDocumentByServerId(serverId) {
	const documents = await request((await store("readonly")).getAll());
	return documents.find(localDocument => localDocument.serverId === serverId);
}

// hasUnsyncedChanges reports whether the document changed after it was last synced.
export function hasUnsyncedChanges(localDocument) {
	return (localDocument.updatedAt ?? 0) > (localDocument.syncedAt ?? 0);
}

export async function removeDocument(rootId) {
	await request((await store("readwrite")).delete(rootId));
}

// listDocuments returns the documents, the most recently opened first.
export async function listDocuments() {
	const documents = await request((await store("readonly")).getAll());
	return documents.sort((a, b) => b.openedAt - a.openedAt);
}
