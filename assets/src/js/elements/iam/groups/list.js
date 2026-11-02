import * as $ from "bm.js/bm.module.js";
//import {html, render} from '/lib/lit-html/lit-html.js';
import {html, render} from 'lit-html';
import {css} from "@/common.js";

class CustomElement extends HTMLElement {
	template() {
		return html`
			<style>
				${css}

				:host {
				}
			</style>
			<table>
				<thead>
					<tr>
						<th>Name</th>
						<th>Roles</th>
					</tr>
				</thead>
				<tbody>
				${this.items.map((item) => html`
					<tr>
						<td>${item.name}</td>
						<td>${item.roles.join(", ")}</td>
					</tr>
				`)}
				</tbody>
			</table>
		`
	}
	constructor() {
		super();

		this.attachShadow({mode:"open"});

		this.items = [];
	}

	async render() {
		render(this.template(), this.shadowRoot);
	}

	async onConnected() {
		let res = await $.request("GET", "/api/v1/groups");

		this.items = res.json;

		this.render();
	}
}
customElements.define("iam-groups-list", CustomElement);
