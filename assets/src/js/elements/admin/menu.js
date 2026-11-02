import * as $ from "bm.js/bm.module.js";
import {html, render} from 'lit-html';
import {css} from "@/common.js";

var tmpl = (elem) => html`
	<style>
		${css}

		:host {
		}
		a {
			display: block;
			padding: 0.375rem 0.75rem;
			color: inherit;
			text-decoration: none;
		}
		a:hover {
			background: var(--hover);
		}
		a.selected {
			box-shadow: inset 2px 0 var(--accent);
		}
	</style>
	<a href="/admin" exact>Overview</a>
	<a href="/admin/iam">IAM</a>
`;

class AdminMenu extends HTMLElement {
	constructor() {
		super();
		this.attachShadow({mode: "open"});
	}

	async render() {
		render(tmpl(this), this.shadowRoot);
	}

	onConnected() {
		$.all(this.shadowRoot, "a").
			filter((elem, index) => elem.hasAttribute("exact") ? elem.attr("href") == location.pathname: location.pathname.startsWith(elem.attr("href"))).
			forEach(elem => elem.classList.add("selected"));
	}
}
customElements.define("admin-menu", AdminMenu);
