import * as $ from "bm.js/bm.module.js";
import {html, render} from 'lit-html';
import {css} from "@/common.js";

// The bar on the top of the pages. It shows the account links for the logged in user.
class GlobalNavigationBar extends HTMLElement {
	template() {
		return html`
			<style>
				${css}

				:host {
					display: flex;
					align-items: center;
					gap: 1.5rem;
					min-height: 3rem;
				}
				nav {
					display: flex;
					align-items: center;
					gap: 1rem;
				}
				nav:last-child {
					margin-left: auto;
				}
				a, button {
					padding: 0;
					border: none;
					background: none;
					color: var(--muted);
					text-decoration: none;
				}
				a:hover, button:hover:not(:disabled) {
					background: none;
					color: var(--text);
				}
				#logo {
					color: var(--text);
					font-weight: bold;
				}
			</style>
			<nav>
				<a id="logo" href="/">monosketch</a>
				<a href="/edit">Editor</a>
			</nav>
			<nav>
				${this.user ? html`
					<user-can action="create" resource="server"><a href="/admin">Admin</a></user-can>
					<a href="/users/passkeys">Passkeys</a>
					<a href="/users/settings">${this.user.name}</a>
					<button @click=${() => this.logout()}>Logout</button>
				` : html`
					<a href="/users/login">Login</a>
					<a href="/users/register">Register</a>
				`}
			</nav>
		`;
	}
	constructor() {
		super();

		this.attachShadow({mode: 'open'})
	}

	async render() {
		render(this.template(), this.shadowRoot);
	}

	async onConnected() {
		try {
			const res = await $.request("GET", `/api/v1/users/me`);
			this.user = res.json;
		} catch {
			// Not logged in.
			this.user = null;
		}
		this.render();
	}

	async logout() {
		await $.request("GET", `/api/v1/logout`);
		location.href = "/";
	}
}
customElements.define("global-navigation-bar", GlobalNavigationBar);
