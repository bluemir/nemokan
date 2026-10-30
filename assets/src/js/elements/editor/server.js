// Requests to the server documents. See ADR-0011.
//
// Each returns {status, serverDocument}. status is the HTTP status, or 0 if the server cannot
// be reached. serverDocument is the response body on success.

async function send(method, path, body) {
	try {
		const response = await fetch(`/api/v1${path}`, {
			method,
			headers: {"Content-Type": "application/json", "Accept": "application/json"},
			body: body === undefined ? undefined : JSON.stringify(body),
		});
		const serverDocument = response.ok && response.status !== 204 ? await response.json() : null;
		return {status: response.status, serverDocument};
	} catch (error) {
		console.error("server", error);
		return {status: 0, serverDocument: null};
	}
}

export function createServerDocument(content) {
	return send("POST", "/documents", {content});
}

export function getServerDocument(id) {
	return send("GET", `/documents/${encodeURIComponent(id)}`);
}

// updateServerDocument answers 409 if revision is not the latest one.
export function updateServerDocument(id, revision, content) {
	return send("PUT", `/documents/${encodeURIComponent(id)}`, {revision, content});
}

export function setServerDocumentVisibility(id, visibility) {
	return send("PUT", `/documents/${encodeURIComponent(id)}/visibility`, {visibility});
}

// isLoggedIn reports whether the user is logged in, so a copy becomes their own document.
export async function isLoggedIn() {
	try {
		const response = await fetch("/api/v1/users/me", {headers: {"Accept": "application/json"}});
		return response.ok;
	} catch {
		return false;
	}
}
