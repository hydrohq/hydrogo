import { mkdtempSync, readFileSync, rmSync, writeFileSync, existsSync } from "node:fs";
import { randomBytes } from "node:crypto";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import type { ChildProcess } from "node:child_process";
import { runCommand, spawnServer, stopServer } from "../custom/server-process";

const repo = resolve(process.cwd(), "..");

export const ADMIN_EMAIL = "admin@journeys.test";
// A fresh throwaway password per run. Set before the specs load, so the Playwright workers inherit it.
process.env.HYDRO_JOURNEYS_ADMIN_PASSWORD ||= `${randomBytes(18).toString("base64url")}-Aa1`;
export const ADMIN_PASSWORD: string = process.env.HYDRO_JOURNEYS_ADMIN_PASSWORD!;

function freePort(): Promise<number> {
  return new Promise((ok, fail) => {
    const server = createServer();
    server.once("error", fail);
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address() as { port: number };
      server.close(() => ok(port));
    });
  });
}

async function waitFor(url: string, what: string, server: ChildProcess, log: () => string) {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    if (server.exitCode !== null) throw new Error(`${what} exited early:\n${log()}`);
    try {
      if ((await fetch(url)).ok) return;
    } catch {
      /* not up yet */
    }
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error(`${what} did not become ready at ${url}:\n${log()}`);
}

/**
 * Builds the server, scaffolds a CRM project with a staff account, starts the
 * server and the CRM starter, and tells the specs where they are through
 * JOURNEY_HYDRO_URL and JOURNEY_CRM_URL. Everything is removed afterwards.
 */
export default async function globalSetup() {
  const work = mkdtempSync(join(tmpdir(), "hydro-journeys-"));
  const bin = join(work, process.platform === "win32" ? "hydro.exe" : "hydro");
  // HYDRO_BUILD_TAGS lets an overlay (the private hosted platform) run its own journeys against its own build.
  const tags = process.env.HYDRO_BUILD_TAGS ? ["-tags", process.env.HYDRO_BUILD_TAGS] : [];
  runCommand("go", ["build", ...tags, "-o", bin, "./cmd/hydro"], { cwd: repo });

  const hydroPort = await freePort();
  const hydroUrl = `http://127.0.0.1:${hydroPort}`;

  runCommand(bin, ["new", "proj", "--preset", "crm", "--admin-email", ADMIN_EMAIL, "--admin-password-stdin"], { cwd: work, input: `${ADMIN_PASSWORD}\n` });
  const project = join(work, "proj");
  const configPath = join(project, "config.yaml");
  const tenancyConfig = `
tenancy:
  enabled: true
  require_host: false
  hosts:
    127.0.0.1: "tenant-main"
    localhost: "tenant-main"
    alpha.example.test: "tenant-alpha"
    beta.example.test: "tenant-beta"
`;
  writeFileSync(
    configPath,
    readFileSync(configPath, "utf8").replace(/address: ".*"/, `address: "127.0.0.1:${hydroPort}"`).replace(/public_origin: ".*"/, `public_origin: "${hydroUrl}"`) + tenancyConfig,
  );
  runCommand(bin, ["migrate", "--config", configPath], { cwd: project });
  runCommand(bin, ["create-user", "--email", "staff@journeys.test", "--role", "viewer", "--permissions", "admin.site.manage", "--password-stdin", "--config", configPath], {
    cwd: project,
    input: "Staff-Journey-Pass-2026-Zq!\n",
  });
  const accountOutput = runCommand(bin, ["service-account", "create", "--name", "journeys", "--permissions", "admin.content.read,admin.content.write", "--expires-days", "1", "--config", configPath], { cwd: project });
  process.env.JOURNEY_HYDRO_TOKEN = /^(hyd_\S+)$/m.exec(accountOutput)?.[1] ?? "";

  runCommand(bin, ["tenant", "backfill", "--tenant-id", "tenant-main", "--confirm", "--config", configPath], { cwd: project });
  runCommand(bin, ["tenant", "provision", "--tenant-id", "tenant-alpha", "--admin-email", "admin@alpha.test", "--admin-password-stdin", "--config", configPath], {
    cwd: project,
    input: `${ADMIN_PASSWORD}\n`,
  });
  runCommand(bin, ["tenant", "provision", "--tenant-id", "tenant-beta", "--admin-email", "admin@beta.test", "--admin-password-stdin", "--config", configPath], {
    cwd: project,
    input: `${ADMIN_PASSWORD}\n`,
  });

  const servers: ChildProcess[] = [];
  const logs: string[] = [];
  const capture = (child: ChildProcess) => {
    child.stdout?.on("data", (d) => logs.push(String(d)));
    child.stderr?.on("data", (d) => logs.push(String(d)));
    return child;
  };
  const log = () => logs.join("").slice(-3000);
  try {
    const hydro = capture(spawnServer(bin, ["serve", "--config", configPath], { cwd: project, stdio: ["ignore", "pipe", "pipe"] }));
    servers.push(hydro);
    await waitFor(`${hydroUrl}/healthz`, "hydro", hydro, log);
  } catch (error) {
    console.error("SERVER LOGS:\n", log());
    for (const s of servers) await stopServer(s);
    rmSync(work, { recursive: true, force: true });
    throw error;
  }

  process.env.JOURNEY_HYDRO_URL = hydroUrl;
  process.env.JOURNEY_HYDRO_PORT = String(hydroPort);
  process.env.JOURNEY_HYDRO_ALPHA_URL = `http://alpha.example.test:${hydroPort}`;
  process.env.JOURNEY_HYDRO_BETA_URL = `http://beta.example.test:${hydroPort}`;
  return async () => {
    for (const s of servers.reverse()) await stopServer(s);
    rmSync(work, { recursive: true, force: true });
  };
}
