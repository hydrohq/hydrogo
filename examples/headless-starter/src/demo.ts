import { createHydroGoClient } from "./hydrogo.js";

const environment = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
const client = createHydroGoClient({ origin: environment?.HYDRO_API_ORIGIN ?? "http://127.0.0.1:8090" });
const [page, pages, news, products] = await Promise.all([client.getPage("tr", "demo"), client.listPages("tr"), client.listNews("tr"), client.listProducts()]);

console.log({ page: page.title, pages: pages.map((item) => item.title), news: news.map((item) => item.title), products: products.map((product) => product.title) });
