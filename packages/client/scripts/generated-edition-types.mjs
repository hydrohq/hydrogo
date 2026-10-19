import ts from 'typescript';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const rootNames = [
  ...['project', 'article'].flatMap((name) => [
    path.join(root, `${name}-edition`, 'generated', name, 'client.ts'),
    path.join(root, `${name}-edition`, 'generated', name, 'query.ts'),
    path.join(root, `${name}-edition`, 'generated', name, 'bulk.ts'),
    path.join(root, `${name}-edition`, 'generated', name, 'htmx.ts'),
  ]),
];
const options = { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext, strict: true, skipLibCheck: true, noEmit: true };
const program = ts.createProgram(rootNames, options);
const diagnostics = ts.getPreEmitDiagnostics(program);
if (diagnostics.length) {
  console.error(ts.formatDiagnosticsWithColorAndContext(diagnostics, {
    getCanonicalFileName: (file) => file,
    getCurrentDirectory: () => process.cwd(),
    getNewLine: () => '\n',
  }));
  process.exit(1);
}
console.log(`PASS: generated TypeScript contracts (${rootNames.length} files)`);

