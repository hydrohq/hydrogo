import { spawn, spawnSync, type ChildProcess, type SpawnOptions } from "node:child_process";

export function spawnServer(command: string, args: string[], options: SpawnOptions): ChildProcess {
	return spawn(command, args, { ...options, detached: process.platform !== "win32" });
}

export async function stopServer(server?: ChildProcess): Promise<void> {
	if (!server) return;
	const processGroupExists = (): boolean => {
		if (!server.pid) return false;
		try {
			process.kill(-server.pid, 0);
			return true;
		} catch {
			return false;
		}
	};
	if ((server.exitCode !== null || server.signalCode !== null) && !processGroupExists()) return;

	const exited = server.exitCode !== null || server.signalCode !== null
		? Promise.resolve()
		: new Promise<void>((resolve) => server.once("exit", () => resolve()));
	if (process.platform === "win32") {
		if (server.exitCode === null && server.signalCode === null) {
			spawnSync("taskkill", ["/PID", String(server.pid), "/T", "/F"], { stdio: "ignore" });
		}
	} else if (server.pid) {
		try {
			process.kill(-server.pid, "SIGTERM");
		} catch {
			server.kill("SIGTERM");
		}
	}

	await Promise.race([exited, new Promise<void>((resolve) => setTimeout(resolve, 5_000))]);
	if (process.platform !== "win32" && processGroupExists()) {
		try {
			process.kill(-server.pid!, "SIGKILL");
		} catch {
			server.kill("SIGKILL");
		}
	} else if (server.exitCode === null && server.signalCode === null) {
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
