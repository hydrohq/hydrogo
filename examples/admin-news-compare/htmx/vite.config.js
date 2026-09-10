import { defineConfig } from "vite";
export default defineConfig({ server: { proxy: { "/admin": "http://127.0.0.1:8090", "/vendor": "http://127.0.0.1:8090" } } });
