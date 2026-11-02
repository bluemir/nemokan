import * as $ from "bm.js/bm.module.js";
import {html, render} from 'lit-html';
import {css} from "@/common.js";

var tmpl = (app) => html`
	<style>
		${css}

		:host {
			position: fixed;
			width: 100%;
			height: 100%;

			display: grid;
			grid-template-columns: auto 1fr;
			grid-template-rows: auto 1fr auto;
		}

		header, main, footer{
			padding: 0 1rem;
		}
		aside, main {
			overflow-y: scroll;
		}

		header {
			grid-column: span 2;
			border-bottom: 1px solid var(--border);
		}
		aside {
			min-width: 12rem;
			grid-row: span 2;
			padding: 1rem 0;
			border-right: 1px solid var(--border);
		}
		main {
			padding: 1rem 2rem;
		}
		footer {
			color: var(--muted);
			font-size: 0.75rem;
		}

	</style>
	<header>
		<global-navigation-bar></global-navigation-bar>
	</header>
	<aside>
		<admin-menu></admin-menu>
	</aside>
	<main>
		<slot></slot>
	</main>
	<footer>
		<admin-footer></admin-footer>
	</footer>
`;

class AdminPage extends HTMLElement {
	constructor() {
		super();

		this.attachShadow({mode: "open"});
	}

	async render() {
		render(tmpl(this), this.shadowRoot);
	}
}
customElements.define("admin-page", AdminPage);
