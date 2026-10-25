import { mkdtempSync, readFileSync, rmSync, writeFileSync, existsSync } from "node:fs";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import type { ChildProcess } from "node:child_process";
import { runCommand, spawnServer, stopServer } from "../custom/server-process";

const repo = resolve(process.cwd(), "..");

export const ADMIN_EMAIL = "admin@journeys.test";
export const ADMIN_PASSWORD = "Quartz-Meadow-4417-Lantern";

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
  runCommand("go", ["build", "-o", bin, "./cmd/hydro"], { cwd: repo });

  const hydroPort = await freePort();
  const crmPort = await freePort();
  const hydroUrl = `http://127.0.0.1:${hydroPort}`;
  const crmUrl = `http://127.0.0.1:${crmPort}`;

  runCommand(bin, ["new", "proj", "--preset", "crm", "--admin-email", ADMIN_EMAIL, "--admin-password-stdin"], { cwd: work, input: `${ADMIN_PASSWORD}\n` });
  const project = join(work, "proj");
  const configPath = join(project, "config.yaml");
  writeFileSync(
    configPath,
    readFileSync(configPath, "utf8").replace(/address: ".*"/, `address: "127.0.0.1:${hydroPort}"`).replace(/public_origin: ".*"/, `public_origin: "${hydroUrl}"`),
  );
  runCommand(bin, ["migrate", "--config", configPath], { cwd: project });
  runCommand(bin, ["create-user", "--email", "staff@journeys.test", "--role", "viewer", "--permissions", "admin.site.manage", "--password-stdin", "--config", configPath], {
    cwd: project,
    input: "Staff-Journey-Pass-2026-Zq!\n",
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

    const crmDir = join(repo, "examples", "crm-starter");
    if (!existsSync(join(crmDir, "node_modules"))) runCommand("npm", ["ci"], { cwd: crmDir });
    runCommand("npm", ["run", "build"], { cwd: crmDir });
    const env = { ...process.env, HYDRO_API_ORIGIN: hydroUrl, HYDRO_ADMIN_EMAIL: ADMIN_EMAIL, HYDRO_ADMIN_PASSWORD: ADMIN_PASSWORD };
    runCommand("node", ["scripts/seed.mjs"], { cwd: crmDir, env });
    const crm = capture(
      spawnServer("node", ["build"], {
        cwd: crmDir,
        env: { ...env, PORT: String(crmPort), HOST: "127.0.0.1", ORIGIN: crmUrl, CRM_LANG: "en", CRM_SECURE_COOKIES: "false" },
        stdio: ["ignore", "pipe", "pipe"],
      }),
    );
    servers.push(crm);
    await waitFor(`${crmUrl}/health`, "the CRM starter", crm, log);
  } catch (error) {
    for (const s of servers) await stopServer(s);
    rmSync(work, { recursive: true, force: true });
    throw error;
  }

  process.env.JOURNEY_HYDRO_URL = hydroUrl;
  process.env.JOURNEY_CRM_URL = crmUrl;
  return async () => {
    for (const s of servers.reverse()) await stopServer(s);
    rmSync(work, { recursive: true, force: true });
  };
}
