import { createHydroGoClient } from "./hydrogo.js";

const environment = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
const client = createHydroGoClient({ origin: environment?.HYDRO_API_ORIGIN ?? "http://127.0.0.1:8090" });
const [page, products] = await Promise.all([client.getPage("tr", "demo"), client.listProducts()]);

console.log({ page: page.title, products: products.map((product) => product.title) });
