import { check } from "k6";
import http from "k6/http";

export default function() {
	let urls = [
		"https://grafana.com/",
		"https://grafana.com/web/app.min.css",
		"https://grafana.com/web/shared.min.css",
		"https://grafana.com/at.js",
		"https://grafana.com/web/app.js",
		"https://grafana.com/web/shared.js",
	];

	urls.forEach(url => {
		let res = http.get(url);
		check(res, {
			"status is 200": (r) => r.status === 200,
		});
	});
}
