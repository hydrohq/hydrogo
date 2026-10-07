import { spawn, spawnSync, type ChildProcess, type SpawnOptions } from "node:child_process";

export function spawnServer(command: string, args: string[], options: SpawnOptions): ChildProcess {
	return spawn(command, args, { ...options, detached: process.platform !== "win32" });
}

export async function stopServer(server?: ChildProcess): Promise<void> {
	if (!server || server.exitCode !== null || server.signalCode !== null) return;

	const exited = new Promise<void>((resolve) => server.once("exit", () => resolve()));
	if (process.platform === "win32") {
		spawnSync("taskkill", ["/PID", String(server.pid), "/T", "/F"], { stdio: "ignore" });
	} else if (server.pid) {
		try {
			process.kill(-server.pid, "SIGTERM");
		} catch {
			server.kill("SIGTERM");
		}
	}

	await Promise.race([exited, new Promise<void>((resolve) => setTimeout(resolve, 5_000))]);
	if (server.exitCode === null && server.signalCode === null) {
		if (process.platform === "win32") {
			spawnSync("taskkill", ["/PID", String(server.pid), "/T", "/F"], { stdio: "ignore" });
		} else if (server.pid) {
			try {
				process.kill(-server.pid, "SIGKILL");
			} catch {
				server.kill("SIGKILL");
			}
		}
	}
}
