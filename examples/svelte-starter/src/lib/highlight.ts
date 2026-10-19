/** A deliberately tiny syntax highlighter: ordered sticky regex rules per language. */
type Rule = [RegExp, string];

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

const kw = (words: string) => new RegExp(`(?<![\\w$.])(?:${words})\\b`, 'y');

const jsKeywords =
  'const|let|var|function|return|if|else|for|while|await|async|import|from|export|default|new|class|extends|type|interface|try|catch|throw|of|in|as|null|undefined|true|false|typeof|void|switch|case|break|continue|struct|func|package|go|defer|range|map|chan|nil|string|int|bool|error|any';

const rules: Record<string, Rule[]> = {
  bash: [
    [/#.*/y, 'c'],
    [/"(?:[^"\\]|\\.)*"|'[^']*'/y, 's'],
    [/\$\{[^}]*\}|\$[A-Za-z_][\w]*/y, 'p'],
    [/(?<![\w-])--?[A-Za-z][\w-]*/y, 'n'],
    [/(?<![\w./-])(?:hydro|curl|docker|npm|npx|go|sudo|systemctl|cd|export|cp|mkdir|sh|git|make|irm|iex|journalctl|chmod|chown|node|pg_dump|mysqldump|tar|crontab|ls|cat|echo|set|setx)(?![\w-])/y, 'k'],
    [/\b\d+(?:\.\d+)*\b/y, 'f'],
  ],
  yaml: [
    [/#.*/y, 'c'],
    [/"(?:[^"\\]|\\.)*"|'[^']*'/y, 's'],
    [/[A-Za-z_][\w.-]*(?=\s*:(?:\s|$))/y, 'n'],
    [/(?<=\{|,)\s*[A-Za-z_][\w.-]*(?=\s*:)/y, 'n'],
    [/\b(?:true|false|null|yes|no)\b/y, 'k'],
    [/-?\b\d+(?:\.\d+)?\b/y, 'f'],
  ],
  json: [
    [/"(?:[^"\\]|\\.)*"(?=\s*:)/y, 'n'],
    [/"(?:[^"\\]|\\.)*"/y, 's'],
    [/\b(?:true|false|null)\b/y, 'k'],
    [/-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b/y, 'f'],
  ],
  ts: [
    [/\/\/.*|\/\*[\s\S]*?\*\//y, 'c'],
    [/"(?:[^"\\\n]|\\.)*"|'(?:[^'\\\n]|\\.)*'|`(?:[^`\\]|\\.)*`/y, 's'],
    [kw(jsKeywords), 'k'],
    [/\b[A-Za-z_$][\w$]*(?=\()/y, 'f'],
    [/\b\d+(?:\.\d+)?\b/y, 'f'],
  ],
  sql: [
    [/--.*/y, 'c'],
    [/'(?:[^']|'')*'/y, 's'],
    [/\b(?:select|from|where|insert|into|values|update|set|delete|create|table|alter|drop|index|and|or|not|null|primary|key|join|on|order|by|limit|grant|all|privileges|database|user|identified|to)\b/iy, 'k'],
    [/\b\d+\b/y, 'f'],
  ],
  ini: [
    [/[#;].*/y, 'c'],
    [/^\[[^\]\n]+\]/my, 'k'],
    [/"(?:[^"\\]|\\.)*"/y, 's'],
    [/^[A-Za-z_][\w.-]*(?=\s*=)/my, 'n'],
    [/\b\d+\b/y, 'f'],
  ],
  conf: [
    [/#.*/y, 'c'],
    [/"(?:[^"\\]|\\.)*"|'[^']*'/y, 's'],
    [/\$[A-Za-z_][\w]*|\{[A-Za-z_.]+\}/y, 'p'],
    [/^\s*[A-Za-z_][\w.-]*/my, 'n'],
    [/\b\d+[smhd]?\b/y, 'f'],
  ],
  http: [
    [/^(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\b/my, 'k'],
    [/^HTTP\/[\d.]+ \d+/my, 'k'],
    [/^[A-Za-z][\w-]*(?=:)/my, 'n'],
    [/"(?:[^"\\]|\\.)*"/y, 's'],
  ],
};

const aliases: Record<string, string> = {
  sh: 'bash', shell: 'bash', zsh: 'bash', powershell: 'bash', ps1: 'bash', console: 'bash', cmd: 'bash',
  yml: 'yaml', js: 'ts', javascript: 'ts', typescript: 'ts', svelte: 'ts', go: 'ts', jsx: 'ts', tsx: 'ts',
  env: 'ini', toml: 'ini', systemd: 'ini', caddyfile: 'conf', nginx: 'conf', apache: 'conf',
};

export function languageLabel(lang: string): string {
  const labels: Record<string, string> = {
    bash: 'Shell', sh: 'Shell', shell: 'Shell', powershell: 'PowerShell', ps1: 'PowerShell', yaml: 'YAML', yml: 'YAML',
    json: 'JSON', ts: 'TypeScript', typescript: 'TypeScript', js: 'JavaScript', javascript: 'JavaScript', svelte: 'Svelte',
    go: 'Go', sql: 'SQL', ini: 'INI', env: 'Env', systemd: 'systemd', nginx: 'nginx', caddyfile: 'Caddyfile', http: 'HTTP',
    text: 'Text', html: 'HTML', css: 'CSS', apache: 'Apache', dockerfile: 'Dockerfile', xml: 'XML', console: 'Shell',
  };
  return labels[lang] ?? lang;
}

export function highlight(code: string, lang: string): string {
  const set = rules[aliases[lang] ?? lang];
  if (!set) return esc(code);
  let out = '';
  let i = 0;
  while (i < code.length) {
    let matched = false;
    for (const [re, cls] of set) {
      re.lastIndex = i;
      const m = re.exec(code);
      if (m && m.index === i && m[0].length > 0) {
        out += `<span class="tok-${cls}">${esc(m[0])}</span>`;
        i += m[0].length;
        matched = true;
        break;
      }
    }
    if (!matched) {
      out += esc(code[i]);
      i++;
    }
  }
  return out;
}
