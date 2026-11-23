import { DurableObject } from "cloudflare:workers";

// One Durable Object owns one HydroGo container. Every request is proxied to it.
const PORT = 8080;
const SLEEP_AFTER_MS = 10 * 60 * 1000;
const PASS_THROUGH = /^(HYDRO_|LITESTREAM_|RECOVER_SECRET$)/;

interface Env {
	SITE: DurableObjectNamespace<HydroContainer>;
	HYDRO_HTTP_PUBLIC_ORIGIN: string;
	// "lite" (256 MiB) is enough for a small site; use "basic" (1 GiB) if the container is OOM-killed.
	CONTAINER_INSTANCE?: "lite" | "basic" | "standard-1" | "standard-2";
	[key: string]: unknown;
}

export class HydroContainer extends DurableObject<Env> {
	private starting: Promise<void> | undefined;

	async fetch(request: Request): Promise<Response> {
		this.starting ??= this.start().finally(() => {
			this.starting = undefined;
		});
		await this.starting;

		// HydroGo rejects any Host other than its public origin (421), so always
		// present that one, whatever hostname the visitor used to reach the Worker.
		const url = new URL(request.url);
		url.protocol = "http:";
		url.host = new URL(this.env.HYDRO_HTTP_PUBLIC_ORIGIN).host;
		const forwarded = new Request(url, request);
		forwarded.headers.delete("host");
		return this.ctx.container!.getTcpPort(PORT).fetch(forwarded);
	}

	private async start(): Promise<void> {
		const container = this.ctx.container!;
		if (!container.running) {
			const env: Record<string, string> = {
				// all traffic reaches the container through Cloudflare
				HYDRO_HTTP_TRUST_CF_CONNECTING_IP: "true",
			};
			for (const [key, value] of Object.entries(this.env)) {
				if (PASS_THROUGH.test(key) && typeof value === "string") env[key] = value;
			}
			container.start({
				image: container.images.base,
				instance: this.env.CONTAINER_INSTANCE ?? "lite",
				enableInternet: true, // Litestream and media talk to R2
				env,
			});
		}
		await container.setInactivityTimeout(SLEEP_AFTER_MS);

		const host = new URL(this.env.HYDRO_HTTP_PUBLIC_ORIGIN).host;
		const port = container.getTcpPort(PORT);
		for (let attempt = 0; attempt < 150; attempt++) {
			try {
				const res = await port.fetch(`http://${host}/healthz`, { signal: AbortSignal.timeout(1000) });
				await res.body?.cancel();
				if (res.ok) return;
			} catch {
				// not listening yet
			}
			await scheduler.wait(200);
		}
		throw new Error("HydroGo did not become ready on port 8080");
	}
}

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		return env.SITE.getByName("site").fetch(request);
	},
} satisfies ExportedHandler<Env>;
