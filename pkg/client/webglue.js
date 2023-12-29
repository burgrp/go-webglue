/* global Function, DIV, io */

import "jquery";

/**** tag factories ****/

let tags = {};

function createFactory(fncName, htmlTag) {
	tags[fncName] = (...args) => {
		let el = $(`<${htmlTag}>`);

		function process(arg) {
			if (arg instanceof Array) {
				el.append(arg);
			} else if (arg instanceof Function) {
				function maybeProcess(result) {
					if (result != el && result != undefined && result != null) {
						process(result);
					}
				}
				let result = arg(el);
				if (result instanceof Promise) {
					asy(async () => {
						maybeProcess(await result);
					})
				} else {
					maybeProcess(result);
				}
			} else if (arg instanceof Object) {
				el.attr(arg);
			} else if (typeof arg === "string") {
				el.addClass(arg);
			}
		}

		args.forEach(arg => process(arg));
		return el;
	};
}

createFactory("DIV", "div");
createFactory("SPAN", "span");
createFactory("H1", "h1");
createFactory("H2", "h2");
createFactory("H3", "h3");
createFactory("AHREF", "a");
createFactory("BUTTON", "button");
createFactory("LABEL", "label");
createFactory("PAR", "p");
createFactory("IMG", "img");
createFactory("SETOFF", "i");
createFactory("FORM", "form");
createFactory("SELECT", "select");
createFactory("OPTION", "option");
createFactory("INPUT", "input");
createFactory("TEXT", 'input type="text"');
createFactory("PASSWORD", "input type='password'");
createFactory("NUMBER", 'input type="number"');
createFactory("CHECKBOX", 'input type="checkbox"');
createFactory("RADIO", 'input type="radio"');
createFactory("TEXTAREA", "textarea");
createFactory("TABLE", "table");
createFactory("TR", "tr");
createFactory("TD", "td");
createFactory("TH", "th");
createFactory("ICON", "i");
createFactory("FIELDSET", "fieldset");

let api = {};

let page;

function error(e) {
	console.error(e);
	if (page?.error) {
		page.error(e);
	} else {
		alert(e);
	}
}

function asy(action) {
	async function wrap() {
		await action()
	}
	wrap().catch(e => error(e))
}

async function goto(url, current) {

	if (current) {
		history.replaceState(url, url, url);
	} else {
		history.pushState(url, url, url);
	}

	let params = {};

	if (url.startsWith("/")) {
		url = url.slice(1);
	}

	if (!url || url === "") {
		url = "home";
	}

	let qmPos = url.indexOf("?");
	if (qmPos !== -1) {
		let vars = url.slice(qmPos + 1).split('&');
		vars.forEach(avar => {
			let pair = avar.split('=');
			if (pair.length === 2) {
				params[pair[0]] = decodeURIComponent(pair[1]);
			}
		});
		url = url.slice(0, qmPos);
	}

	if (!url || url === "") {
		url = "home";
	}

	let root = $("body");
	root.empty();

	try {
		page = (await import("./" + url + ".page.js")).default;
	} catch (e) {
		console.error(e);
		page = {
			title: "Not found",
			render: path => [
				tags.DIV("error").text(`Page '${path}' not found.`)
			]
		}
	}

	let render = true;
	if (page.check) {
		let redirection = await page.check(url, params);
		if (redirection) {
			render = false;
			await this.goto(redirection, url);
		}
	}

	if (render) {
		console.info("Rendering", url, params);
		document.title = page.title || url;
		let children = await page.render(url, params);
		if (children) {
			root.append(children);
		}
	}
}


function start() {
	startAsync().catch(err => {
		console.error("Unhandled Webglue error:", err);
	});
}

async function startAsync() {

	console.info("Webglue application starting...");

	/**** navigation ****/

	$(document).on("click", "a", e => {
		let page = $(e.currentTarget).attr("href");
		if (
			!page.startsWith("http") &&
			!page.startsWith("blob") &&
			!page.startsWith("data:")
		) {
			e.preventDefault();
			let page = $(e.currentTarget).attr("href");
			goto(page);
		}
	});

	window.onpopstate = e => {
		if (e.state) {
			goto(e.state, true);
		}
	};

	/**** server API ****/

	let pingIntervalSec;

	async function callApiEndpoint({ method, suffix, body }) {
		let sessionId = localStorage.getItem("Webglue-Session");
		let apiResponse = await fetch("api/" + suffix, {
			method,
			headers: {
				"Content-Type": "application/json",
				...sessionId ? {
					"Webglue-Session": sessionId
				} : {}
			},
			body: JSON.stringify(body),
		});
		sessionId = apiResponse.headers.get("Webglue-Session");
		localStorage.setItem("Webglue-Session", sessionId);
		pingIntervalSec = Number.parseInt(apiResponse.headers.get("Webglue-Ping"));
		return method == "HEAD" ? undefined : await apiResponse.json();
	}

	async function callApiFunction(moduleName, functionName, ...params) {
		let apiResponse = await callApiEndpoint({
			method: "POST",
			suffix: moduleName + "/" + functionName,
			body: params
		});
		if (apiResponse.error) {
			throw new Error(apiResponse.error);
		} else {
			return apiResponse.result;
		}
	}

	let modules = await callApiFunction("webglue", "discover");

	api = Object.fromEntries(
		Object.entries(modules).map(([moduleName, module]) => [
			moduleName,
			Object.fromEntries(
				(module.functions || []).map(functionName => [
					functionName,
					(...params) => callApiFunction(moduleName, functionName, ...params)
				])
			)
		])
	)

	Object.entries(modules).forEach(([moduleName, module]) => {
		(module.events || []).forEach(eventName => {
			let initcap = s => s.charAt(0).toUpperCase() + s.slice(1);
			let methodName = "on" + (moduleName ? initcap(moduleName) : "") + initcap(eventName);
			let jqName = "webglue." + (moduleName ? moduleName + "." : "") + eventName;
			$.fn[methodName] = function (handler) {
				this.on(jqName, (e, ...args) => {
					if (e.currentTarget === e.target) {
						let receiver = $(e.target);
						handler.apply(receiver, [receiver, ...args]);
					}
				});
				return this;
			};
		});
	});

	new EventSource("events?stream=webglue").onmessage = e => {
		let data = JSON.parse(e.data);
		$("*").trigger(`webglue.${data.module}.${data.name}`, data.params);
	}

	// DISABLED TILL WE NEED IT FOR SESSION KEEPALIVE
	// async function pingLoop() {
	// 	while (true) {
	// 		try {
	// 			await callApiEndpoint({
	// 				method: "HEAD",
	// 			});
	// 		} catch (err) {
	// 			console.error("Error in ping loop:", err);
	// 		}
	// 		await new Promise((resolve, reject) => { setTimeout(resolve, pingIntervalSec * 1000); });
	// 	}
	// }

	// pingLoop().catch(err => {
	// 	console.error("Unhandled error in ping loop:", err);
	// });

	/**** and the UI ****/

	goto(window.location.pathname + window.location.search, true);

}

export {
	start,
	api,
	asy,
	error,
	goto,
	tags
};
