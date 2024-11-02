import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			"/random": "http://127.0.0.1:8080",
			"/obj": "http://127.0.0.1:8080",
			"/dir": "http://127.0.0.1:8080",
		},
	},
});
